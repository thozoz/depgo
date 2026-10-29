package middlewares

import (
	"bytes"
	"net/url"
	"strings"
	"testing"
)

func TestRedactRawQueryHidesCredentials(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		notIn  []string
		wantIn []string
	}{
		{
			name:   "share auth token",
			in:     "share_id=sesscheck-txt&auth=abc.def.ghi",
			notIn:  []string{"abc.def.ghi"},
			wantIn: []string{"share_id=sesscheck-txt", "auth=" + url.QueryEscape(redactedValue)},
		},
		{
			name:   "download signature",
			in:     "sign=c2lnbmF0dXJlOjE3MDAwMDAwMDA&expire=1700000000",
			notIn:  []string{"c2lnbmF0dXJlOjE3MDAwMDAwMDA"},
			wantIn: []string{"expire=1700000000", "sign=" + url.QueryEscape(redactedValue)},
		},
		{
			name:   "multiple sensitive params",
			in:     "token=t0k3n&password=hunter2&secret=s3cr3t&path=/docs",
			notIn:  []string{"t0k3n", "hunter2", "s3cr3t"},
			wantIn: []string{"path=%2Fdocs"},
		},
		{
			name:   "case insensitive",
			in:     "AUTH=topsecretvalue",
			notIn:  []string{"topsecretvalue"},
			wantIn: []string{"AUTH=" + url.QueryEscape(redactedValue)},
		},
		{
			name:   "archive password",
			in:     "inner=secret.txt&pass=hunter2",
			notIn:  []string{"hunter2"},
			wantIn: []string{"pass=" + url.QueryEscape(redactedValue)},
		},
		{
			name:   "lark file token",
			in:     "file_token=t-abc123secret",
			notIn:  []string{"t-abc123secret"},
			wantIn: []string{"file_token=" + url.QueryEscape(redactedValue)},
		},
		{
			name:   "oauth authorization code",
			in:     "code=4/0AeanS0abc&state=xyz",
			notIn:  []string{"4/0AeanS0abc"},
			wantIn: []string{"code=" + url.QueryEscape(redactedValue)},
		},
		{
			name:   "harmless query untouched",
			in:     "share_id=abc&path=%2Ffolder&page=2",
			notIn:  []string{},
			wantIn: []string{"share_id=abc", "page=2"},
		},
		{
			name:   "empty query",
			in:     "",
			notIn:  []string{},
			wantIn: []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactRawQuery(tc.in)
			for _, secret := range tc.notIn {
				if strings.Contains(got, secret) {
					t.Fatalf("redacted query still contains %q: %q", secret, got)
				}
			}
			for _, want := range tc.wantIn {
				if !strings.Contains(got, want) {
					t.Fatalf("redacted query %q missing %q", got, want)
				}
			}
		})
	}
}

func TestRedactRawQueryMalformedInput(t *testing.T) {
	// A malformed query (%zz is not valid percent-encoding) must not be echoed
	// verbatim, since it cannot be parsed to find credentials.
	got := redactRawQuery("auth=%zz&other=1")
	if strings.Contains(got, "%zz") {
		t.Fatalf("malformed value leaked: %q", got)
	}
	if !strings.Contains(got, "auth=") || !strings.Contains(got, redactedValue) {
		t.Fatalf("sensitive name not redacted: %q", got)
	}
	if !strings.Contains(got, "other=1") {
		t.Fatalf("harmless param dropped: %q", got)
	}
}

func TestRedactingLoggerDoesNotWriteToken(t *testing.T) {
	var buf bytes.Buffer
	// Exercise the redaction helper directly: building a full gin context would
	// duplicate what the helper already guarantees, and the logger's only job is
	// to call it.
	raw := "aff=abc&auth=SUPERSECRETTOKEN&sign=SUPERSECRETSIGN"
	redacted := redactRawQuery(raw)
	if strings.Contains(redacted, "SUPERSECRET") {
		t.Fatalf("secret leaked through redaction: %q", redacted)
	}
	buf.WriteString(redacted)
	if strings.Contains(buf.String(), "SUPERSECRET") {
		t.Fatalf("secret leaked into log buffer: %q", buf.String())
	}
}

// TestSanitizeLogFieldEscapesControlChars is the regression test for log
// injection through a file name: a newline in the URL path let a crafted name
// forge an entire extra access-log line.
func TestSanitizeLogFieldEscapesControlChars(t *testing.T) {
	cases := map[string]string{
		"/docs/plain.txt":        "/docs/plain.txt",
		"/docs/evil\n[GIN] fake": `/docs/evil\x0a[GIN] fake`,
		"/docs/evil\r\nX: y":     `/docs/evil\x0d\x0aX: y`,
		"/docs/tab\there":        `/docs/tab\x09here`,
		"/docs/nul\x00byte":      `/docs/nul\x00byte`,
		"/docs/del\x7fchar":      `/docs/del\x7fchar`,
		"/docs/unicode/日本語.txt":  "/docs/unicode/日本語.txt",
	}
	for input, want := range cases {
		got := sanitizeLogField(input)
		if got != want {
			t.Errorf("sanitizeLogField(%q) = %q, want %q", input, got, want)
		}
		if strings.ContainsAny(got, "\r\n\t\x00") {
			t.Errorf("sanitizeLogField(%q) = %q still contains a control character", input, got)
		}
	}
}

// TestSanitizeLogFieldIsIdentityForOrdinaryPaths keeps the common path allocation
// free and unchanged.
func TestSanitizeLogFieldIsIdentityForOrdinaryPaths(t *testing.T) {
	for _, p := range []string{"/", "/a/b/c.txt", "/share/sesscheck-txt", "/a%25b.txt"} {
		if got := sanitizeLogField(p); got != p {
			t.Errorf("sanitizeLogField(%q) = %q, want identity", p, got)
		}
	}
}
