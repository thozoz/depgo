package utils

import (
	"net/url"
)

// InjectQuery appends query to the URL raw.
//
// The parameters are merged into the parsed URL rather than concatenated onto
// the string. Concatenation looked equivalent but broke two shapes:
//
//   - a URL with a fragment (".../p#frag") received ".../p#frag?a=b", which puts
//     the parameters *inside* the fragment, so the server never sees them;
//   - a URL already ending in "?" received ".../p??a=b", which is malformed.
//
// Merging also means a parameter the destination already defines is kept: the
// upstream knows better than a client-supplied forward.
func InjectQuery(raw string, query url.Values) (string, error) {
	if len(query) == 0 {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	merged := u.Query()
	for key, values := range query {
		if merged.Has(key) {
			continue
		}
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	u.RawQuery = merged.Encode()
	return u.String(), nil
}
