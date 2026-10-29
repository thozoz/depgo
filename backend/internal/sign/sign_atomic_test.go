package sign

import (
	"sync"
	"testing"

	pkgsign "github.com/alist-org/alist/v3/pkg/sign"
	"github.com/stretchr/testify/assert"
)

// TestInstanceIsRaceFree exercises the pattern the server produces: download
// requests read the signer while an administrator rotates the site token and the
// handler rebuilds the signer. Run with -race; the previous plain-variable
// implementation fails here.
func TestInstanceIsRaceFree(t *testing.T) {
	// Seed the signer so loadSigner never needs the settings database, then race
	// publication against reads exactly as the server does.
	storeSigner(pkgsign.NewHMACSign([]byte("seed-key")))

	var wg sync.WaitGroup
	const iterations = 200

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			storeSigner(pkgsign.NewHMACSign([]byte("rotated-key")))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if signer := loadSigner(); signer == nil {
				t.Error("loadSigner returned nil")
				return
			}
			_ = loadSigner().Sign("data", 123)
		}
	}()
	wg.Wait()
}

// TestLoadSignerAlwaysReturnsAValue guards against the nil-interface case that a
// torn read could produce: every call must yield a usable signer.
func TestLoadSignerAlwaysReturnsAValue(t *testing.T) {
	// Seed so the fallback path does not need the settings database.
	storeSigner(pkgsign.NewHMACSign([]byte("fallback-key")))
	signer := loadSigner()
	assert.NotNil(t, signer, "loadSigner must never return nil")

	// A freshly loaded signer must actually work.
	sig := signer.Sign("payload", 4102444800) // year 2100
	assert.NoError(t, signer.Verify("payload", sig))
}

// TestVerifyRejectsPerpetualSignature keeps the ":0 means forever" hole closed.
func TestVerifyRejectsPerpetualSignature(t *testing.T) {
	storeSigner(pkgsign.NewHMACSign([]byte("verify-key")))
	t.Cleanup(func() { instance.Store(nil) })
	signer := loadSigner()
	sig := signer.Sign("payload", 0) // never-expiring
	assert.ErrorIs(t, Verify("payload", sig), pkgsign.ErrSignExpired,
		"a signature with expire=0 must be rejected")
}
