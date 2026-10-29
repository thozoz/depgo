package handles_test

import (
	"testing"

	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// TestSearchFilterUsesPathContainment is the regression test for the search
// filter admitting sibling paths: the check was strings.HasPrefix, so a user
// confined to "/public" also saw results under "/public-secret".
func TestSearchFilterUsesPathContainment(t *testing.T) {
	const basePath = "/public"

	// Paths a naive prefix test would wrongly admit.
	for _, parent := range []string{"/public-secret", "/public2", "/publicity"} {
		assert.False(t, utils.IsSubPath(basePath, parent),
			"search must not admit results under %q for a user confined to %q", parent, basePath)
	}

	// Paths it must admit.
	for _, parent := range []string{"/public", "/public/docs", "/public/a/b"} {
		assert.True(t, utils.IsSubPath(basePath, parent),
			"search must admit results under %q", parent)
	}
}
