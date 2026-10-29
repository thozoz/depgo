package middlewares

import (
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/gin-gonic/gin"
)

// NoStore marks a response as uncacheable. Any response that carries or depends
// on a credential (session JWT, share access token, download signature) must
// never be written to a shared cache: the token would outlive the request in a
// proxy or browser cache and could be replayed by another user of that cache.
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// SecurityHeaders applies a conservative default set of response headers to
// every request. HSTS is only emitted when the deployment terminates TLS
// (directly or behind a proxy that forwards X-Forwarded-Proto), because sending
// it over plain HTTP is meaningless and would lock users out of a non-TLS
// endpoint.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		// Stop browsers from guessing content types; a JSON error rendered as
		// HTML is a classic XSS foothold.
		h.Set("X-Content-Type-Options", "nosniff")
		// The SPA is never meant to be embedded by a third party.
		h.Set("X-Frame-Options", "DENY")
		// Share tokens travel in the URL of media routes; never leak them to a
		// cross-origin destination through the Referer header.
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")
		if csp := contentSecurityPolicy(); csp != "" {
			h.Set("Content-Security-Policy", csp)
		}
		if isHTTPS(c) {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// contentSecurityPolicy returns the CSP for the bundled SPA.
//
// 'unsafe-inline' for styles is required by React and Tailwind, which set style
// attributes at runtime; scripts do not need it because Vite emits external
// module files only. connect-src stays 'self' because every API call is
// same-origin. img/media allow data: and blob: for in-browser previews.
//
// object-src, frame-src and connect-src allow blob: because the built-in PDF
// reader renders a fetched PDF through <object data="blob:..."> (object-src
// 'none' would blank it) and a viewer may read that same-origin blob back.
//
// A deployment that injects custom markup through "customize head/body" may load
// third-party scripts the policy cannot know about. Rather than silently
// breaking such installations the policy is widened there; the operator opted
// into outside code, and the default install stays strict.
func contentSecurityPolicy() string {
	if hasCustomMarkup() {
		return "default-src 'self'; script-src 'self' 'unsafe-inline' https:; " +
			"style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; " +
			"media-src 'self' data: blob: https:; font-src 'self' data: https:; " +
			"connect-src 'self' blob: https:; frame-src 'self' blob: https:; object-src 'self' blob:; " +
			"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"
	}
	return strictContentSecurityPolicy()
}

// strictContentSecurityPolicy is the default policy of an install without custom
// markup. It reads no settings, so it can be tested on its own.
func strictContentSecurityPolicy() string {
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"media-src 'self' data: blob:",
		"font-src 'self' data:",
		"connect-src 'self' blob:",
		"worker-src 'self' blob:",
		"frame-src 'self' blob:",
		"object-src 'self' blob:",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

func hasCustomMarkup() bool {
	return strings.TrimSpace(setting.GetStr(conf.CustomizeHead)) != "" ||
		strings.TrimSpace(setting.GetStr(conf.CustomizeBody)) != ""
}

func isHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.EqualFold(strings.TrimSpace(strings.Split(proto, ",")[0]), "https")
	}
	return false
}
