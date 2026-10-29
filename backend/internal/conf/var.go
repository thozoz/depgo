package conf

import (
	"net/url"
)

var (
	BuiltAt    string
	GitAuthor  string
	GitCommit  string
	Version    string = "dev"
	WebVersion string
)

var (
	Conf *Config
	URL  *url.URL
)

// Settings derived from the database (file type lists, filename character
// mapping, privacy expressions) live in derived_settings.go behind an RWMutex:
// they are rewritten on every settings save while request handlers read them.

var (
	// StoragesLoaded loaded success if empty
	StoragesLoaded = false
)

// The rendered SPA shells live in index_html.go behind atomic snapshots,
// because they are rebuilt from settings while requests are being served.
