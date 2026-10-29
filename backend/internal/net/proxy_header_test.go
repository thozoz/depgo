package net

import (
	"net/http"
	"testing"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/stretchr/testify/assert"
)

// TestProcessHeaderStripsCredentialsEvenWithEmptyIgnoreList is the regression test
// for credentials leaking to the upstream provider: the only filter was the
// operator-configurable proxy_ignore_headers list, so an empty or misconfigured
// list forwarded the caller's session token, cookies and internal Referer.
func TestProcessHeaderStripsCredentialsEvenWithEmptyIgnoreList(t *testing.T) {
	conf.SetSlice(conf.ProxyIgnoreHeaders, nil) // worst case: nothing configured

	origin := http.Header{}
	origin.Set("Authorization", "Bearer SECRET-SESSION-JWT")
	origin.Set("Cookie", "session=SECRETCOOKIE")
	origin.Set("X-Api-Key", "SECRET-APIKEY")
	origin.Set("Referer", "https://internal.example/admin")
	origin.Set("X-Share-Token", "SECRET-SHARE-TOKEN")
	origin.Set("Accept", "*/*")

	got := ProcessHeader(origin, http.Header{})

	for _, header := range []string{
		"Authorization", "Cookie", "X-Api-Key", "Referer", "X-Share-Token",
	} {
		assert.Emptyf(t, got.Get(header), "%s must never be forwarded upstream", header)
	}
	// Harmless headers still pass through.
	assert.Equal(t, "*/*", got.Get("Accept"))
}

// TestProcessHeaderAppliesConfiguredIgnoreList keeps the operator setting working.
func TestProcessHeaderAppliesConfiguredIgnoreList(t *testing.T) {
	conf.SetSlice(conf.ProxyIgnoreHeaders, []string{"x-custom-trace"})
	t.Cleanup(func() { conf.SetSlice(conf.ProxyIgnoreHeaders, nil) })

	origin := http.Header{}
	origin.Set("X-Custom-Trace", "internal-trace-id")
	origin.Set("Accept", "text/html")

	got := ProcessHeader(origin, http.Header{})
	assert.Empty(t, got.Get("X-Custom-Trace"), "configured ignore list must still be honoured")
	assert.Equal(t, "text/html", got.Get("Accept"))
}

// TestProcessHeaderOverrideWins records that driver-supplied headers take
// precedence over the caller's, which is how drivers attach required auth.
func TestProcessHeaderOverrideWins(t *testing.T) {
	conf.SetSlice(conf.ProxyIgnoreHeaders, nil)
	t.Cleanup(func() { conf.SetSlice(conf.ProxyIgnoreHeaders, nil) })

	origin := http.Header{}
	origin.Set("User-Agent", "client-ua")

	override := http.Header{}
	override.Set("User-Agent", "driver-ua")
	override.Set("Authorization", "Bearer DRIVER-OWN-TOKEN")

	got := ProcessHeader(origin, override)
	assert.Equal(t, "driver-ua", got.Get("User-Agent"), "override must win over the client value")
	assert.Equal(t, "Bearer DRIVER-OWN-TOKEN", got.Get("Authorization"),
		"the driver's own upstream credential must survive: only client headers are stripped")
}
