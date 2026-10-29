package middlewares

import (
	"strings"
	"testing"
)

// The built-in PDF reader renders a fetched PDF through <object data="blob:...">.
// A CSP with object-src 'none' blanks it, so the policy must keep allowing blob:
// for objects and frames while still refusing remote plugins.
func TestContentSecurityPolicyKeepsPdfReaderWorking(t *testing.T) {
	csp := strictContentSecurityPolicy()
	for _, want := range []string{"object-src 'self' blob:", "frame-src 'self' blob:", "script-src 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Fatalf("CSP %q is missing %q", csp, want)
		}
	}
	if strings.Contains(csp, "object-src *") || strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Fatalf("default CSP must stay strict, got %q", csp)
	}
}
