package conf

import "sync/atomic"

// indexHtml holds the rendered SPA shell.
//
// The shell is rebuilt from settings (site title, favicon, theme colour,
// operator-injected head/body markup) and read by every request that serves the
// app or a share page. The rebuild runs from the admin settings handler while
// requests are in flight, so a plain string field is a data race: the reader can
// observe a half-written string header, and the Go memory model gives no
// ordering guarantee at all.
//
// An atomic.Pointer[string] keeps the read lock-free on the hot path and makes
// each Update publish a complete, immutable snapshot.
var indexHtml atomic.Pointer[string]

func init() {
	empty := ""
	indexHtml.Store(&empty)
}

// IndexHtml returns the current rendered SPA shell. The returned string is safe
// to use for the lifetime of the request; it will not be mutated in place.
func IndexHtml() string {
	if p := indexHtml.Load(); p != nil {
		return *p
	}
	return ""
}

// SetIndexHtml publishes a new rendered shell. Callers must pass the complete
// document, not a fragment.
func SetIndexHtml(html string) {
	indexHtml.Store(&html)
}

// manageHtml holds the legacy "@manage" shell, with the same concurrency rules.
var manageHtml atomic.Pointer[string]

func init() {
	empty := ""
	manageHtml.Store(&empty)
}

// ManageHtml returns the current legacy admin shell.
func ManageHtml() string {
	if p := manageHtml.Load(); p != nil {
		return *p
	}
	return ""
}

// SetManageHtml publishes a new legacy admin shell.
func SetManageHtml(html string) {
	manageHtml.Store(&html)
}

// rawIndexHtml holds the shell read from disk before any substitution.
var rawIndexHtml atomic.Pointer[string]

func init() {
	empty := ""
	rawIndexHtml.Store(&empty)
}

// RawIndexHtml returns the shell as it was read from disk.
func RawIndexHtml() string {
	if p := rawIndexHtml.Load(); p != nil {
		return *p
	}
	return ""
}

// SetRawIndexHtml publishes the shell read from disk.
func SetRawIndexHtml(html string) {
	rawIndexHtml.Store(&html)
}
