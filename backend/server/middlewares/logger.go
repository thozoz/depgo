package middlewares

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// redactedQueryParams lists query parameters whose value is a credential or a
// signature. They must never be written to an access log: a log line is copied
// into monitoring systems, support tickets and backups, and the value stays
// valid long after the request that produced it.
var redactedQueryParams = map[string]struct{}{
	"auth":          {},
	"token":         {},
	"sign":          {},
	"signature":     {},
	"sig":           {},
	"password":      {},
	"secret":        {},
	"key":           {},
	"api_key":       {},
	"access_token":  {},
	"refresh_token": {},
	// Archive passwords travel as ?pass= on the /ad, /ap and /ae routes.
	"pass": {},
	// Lark file access tokens.
	"file_token": {},
	// OAuth authorisation code: short-lived, but it is exchanged for an access
	// token and must not sit in a log file.
	"code": {},
}

const redactedValue = "[REDACTED]"

// sanitizeLogField removes control characters from a value before it is written
// to the log.
//
// The URL path arrives decoded, and a file name may legally contain a newline on
// Linux. Printing it verbatim lets a crafted name split the log line and forge a
// second, entirely believable entry ("... | 200 | ... /admin"). Control bytes are
// escaped so the value stays visible without becoming line-oriented.
func sanitizeLogField(value string) string {
	if value == "" {
		return value
	}
	needsEscape := false
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return value
	}
	var b strings.Builder
	b.Grow(len(value) + 8)
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < 0x20 || ch == 0x7f {
			fmt.Fprintf(&b, "\\x%02x", ch)
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// redactRawQuery returns the query string with credential values replaced. The
// parameter order is preserved so log diffing keeps working, and parameters are
// parsed leniently: a malformed query is treated as opaque and not echoed at
// all, because it cannot be proven safe.
func redactRawQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// Unparsable query: keep the parameter names only when they are
		// trivially split-able, otherwise drop the whole thing.
		parts := strings.Split(rawQuery, "&")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			name, _, found := strings.Cut(part, "=")
			if !found {
				out = append(out, part)
				continue
			}
			if _, sensitive := redactedQueryParams[strings.ToLower(name)]; sensitive {
				out = append(out, name+"="+redactedValue)
				continue
			}
			out = append(out, part)
		}
		return strings.Join(out, "&")
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	encoded := url.Values{}
	for _, name := range names {
		if _, sensitive := redactedQueryParams[strings.ToLower(name)]; sensitive {
			encoded.Set(name, redactedValue)
			continue
		}
		for _, v := range values[name] {
			encoded.Add(name, v)
		}
	}
	return encoded.Encode()
}

// RedactingLogger is a drop-in replacement for gin.Logger that never writes
// credentials to the log. The output format mirrors gin's default logger so
// existing log parsers keep working.
func RedactingLogger(out io.Writer) gin.HandlerFunc {
	if out == nil {
		out = gin.DefaultWriter
	}
	return func(c *gin.Context) {
		start := time.Now()
		path := sanitizeLogField(c.Request.URL.Path)
		raw := c.Request.URL.RawQuery
		if raw != "" {
			if redacted := redactRawQuery(raw); redacted != "" {
				path = path + "?" + redacted
			}
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String()

		line := fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			status,
			latency,
			clientIP,
			method,
			path,
		)
		if errMsg != "" {
			line += " | " + errMsg
		}
		_, _ = fmt.Fprintln(out, line)
	}
}
