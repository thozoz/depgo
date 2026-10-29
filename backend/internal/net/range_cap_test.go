package net

import (
	"testing"

	"github.com/alist-org/alist/v3/pkg/http_range"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRangeSpec returns a Range header value requesting n single-byte ranges.
func buildRangeSpec(n int) string {
	spec := "bytes="
	for i := 0; i < n; i++ {
		if i > 0 {
			spec += ","
		}
		spec += "0-0"
	}
	return spec
}

// TestRangeCountAmplification documents why the cap exists: a tiny request header
// expands into a much larger multipart response.
func TestRangeCountAmplification(t *testing.T) {
	const fileSize = 1024
	spec := buildRangeSpec(1000)
	ranges, err := http_range.ParseRange(spec, fileSize)
	require.NoError(t, err)
	require.Len(t, ranges, 1000)

	encoded, err := rangesMIMESize(ranges, "application/octet-stream", fileSize)
	require.NoError(t, err)

	// The per-range multipart overhead is what makes this asymmetric.
	assert.Greater(t, encoded, int64(10*len(spec)),
		"1000 ranges should expand the response far beyond the request size")
}

// TestMaxRangesPerRequestIsReasonable keeps the cap in a range that real media
// players never exceed but that still bounds the amplification.
func TestMaxRangesPerRequestIsReasonable(t *testing.T) {
	assert.GreaterOrEqual(t, maxRangesPerRequest, 16, "the cap must not break ordinary seeking")
	assert.LessOrEqual(t, maxRangesPerRequest, 256, "the cap must keep the response bounded")

	// At the cap, the worst case expansion of a 1 KB file stays small.
	const fileSize = 1024
	ranges, err := http_range.ParseRange(buildRangeSpec(maxRangesPerRequest), fileSize)
	require.NoError(t, err)
	encoded, err := rangesMIMESize(ranges, "application/octet-stream", fileSize)
	require.NoError(t, err)
	assert.Less(t, encoded, int64(64*1024),
		"a capped range request must not produce a response larger than a few tens of KB")
}

// TestSumRangesSizeStillGuardsOversizedRequests keeps the pre-existing guard
// meaningful alongside the new count cap.
func TestSumRangesSizeStillGuardsOversizedRequests(t *testing.T) {
	ranges, err := http_range.ParseRange("bytes=0-1023", 1024)
	require.NoError(t, err)
	assert.Equal(t, int64(1024), sumRangesSize(ranges))

	// Requesting more bytes than the file has must be treated as "serve it all".
	over, err := http_range.ParseRange(buildRangeSpec(64), 64)
	require.NoError(t, err)
	assert.LessOrEqual(t, sumRangesSize(over), int64(128))
}
