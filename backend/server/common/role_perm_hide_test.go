package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHidePatternMatchesBasic covers the happy path of a hide expression.
func TestHidePatternMatchesBasic(t *testing.T) {
	assert.True(t, hidePatternMatches(`^secret`, "secret.txt"))
	assert.False(t, hidePatternMatches(`^secret`, "public.txt"))
}

// TestHidePatternMatchesSkipsEmptyLines makes sure a trailing newline in the
// stored expression does not become an empty pattern that matches everything.
func TestHidePatternMatchesSkipsEmptyLines(t *testing.T) {
	assert.False(t, hidePatternMatches("^secret\n\n", "public.txt"))
	assert.True(t, hidePatternMatches("^secret\n\n", "secret.txt"))
}

// TestHidePatternMatchesInvalidPatternFailsClosed is the regression test for a
// panic: the matcher used regexp2.MustCompile, so one invalid line in a meta's
// hide expression crashed every listing request under that meta. A rule that
// cannot be evaluated must deny rather than expose what it was meant to hide.
func TestHidePatternMatchesInvalidPatternFailsClosed(t *testing.T) {
	// An unclosed group is invalid for regexp2.
	assert.True(t, hidePatternMatches("(unclosed", "anything.txt"),
		"an unparsable hide pattern must deny access")
	assert.True(t, hidePatternMatches("^ok\n(unclosed", "anything.txt"),
		"one bad line must deny even when earlier lines are valid")
}
