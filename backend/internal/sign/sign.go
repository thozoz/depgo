package sign

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/sign"
)

const DefaultLinkExpirationHours = 4

// instance holds the signer built from the site token.
//
// It is read on the hot download path and rewritten whenever an administrator
// rotates the token, so it needs to be published atomically: a plain interface
// variable is two words wide, and a concurrent write can be observed torn, which
// at best signs with a mixed key and at worst yields a nil interface and panics.
var instance atomic.Pointer[sign.Sign]

// loadSigner returns the current signer, initialising it on first use.
func loadSigner() sign.Sign {
	if p := instance.Load(); p != nil {
		return *p
	}
	Instance()
	if p := instance.Load(); p != nil {
		return *p
	}
	// Never return nil: callers dereference the result immediately.
	s := sign.NewHMACSign([]byte(setting.GetStr(conf.Token)))
	return s
}

// storeSigner publishes s. It is the single write path, which keeps the atomic
// contract in one place and lets tests exercise publication without a database.
func storeSigner(s sign.Sign) {
	instance.Store(&s)
}

func Sign(data string) string {
	expire := setting.GetInt(conf.LinkExpiration, DefaultLinkExpirationHours)
	if expire <= 0 {
		expire = DefaultLinkExpirationHours
	}
	return WithDuration(data, time.Duration(expire)*time.Hour)
}

func WithDuration(data string, d time.Duration) string {
	return loadSigner().Sign(data, time.Now().Add(d).Unix())
}

func NotExpired(data string) string {
	return loadSigner().Sign(data, 0)
}

func Verify(data string, s string) error {
	// A ":0" expiry means "never expires", which must not be accepted: it would
	// let a leaked link stay valid forever.
	if strings.HasSuffix(s, ":0") {
		return sign.ErrSignExpired
	}
	return loadSigner().Verify(data, s)
}

// Instance rebuilds the signer from the current site token. It is called after
// the token is rotated so new signatures use the new key.
func Instance() {
	storeSigner(sign.NewHMACSign([]byte(setting.GetStr(conf.Token))))
}
