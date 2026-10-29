package utils

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustValues(t *testing.T, pairs ...string) url.Values {
	t.Helper()
	if len(pairs)%2 != 0 {
		t.Fatal("pairs must be key/value")
	}
	v := url.Values{}
	for i := 0; i < len(pairs); i += 2 {
		v.Add(pairs[i], pairs[i+1])
	}
	return v
}

func TestInjectQueryPlainURL(t *testing.T) {
	got, err := InjectQuery("https://example.com/path", mustValues(t, "a", "1"))
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/path?a=1", got)
}

func TestInjectQueryKeepsExistingQuery(t *testing.T) {
	got, err := InjectQuery("https://example.com/path?z=9", mustValues(t, "a", "1"))
	require.NoError(t, err)
	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "9", u.Query().Get("z"))
	assert.Equal(t, "1", u.Query().Get("a"))
}

func TestInjectQueryEscapesValues(t *testing.T) {
	got, err := InjectQuery("https://example.com/p", mustValues(t, "weird", "a b&c=d#frag"))
	require.NoError(t, err)
	u, err := url.Parse(got)
	require.NoError(t, err)
	// The value must round-trip intact and must not have introduced extra keys.
	assert.Equal(t, "a b&c=d#frag", u.Query().Get("weird"))
	assert.Len(t, u.Query(), 1)
}

// TestInjectQueryPreservesFragment is the regression test for parameters being
// appended after the "#": they would land inside the fragment and never reach the
// server.
func TestInjectQueryPreservesFragment(t *testing.T) {
	got, err := InjectQuery("https://example.com/p#frag", mustValues(t, "a", "1"))
	require.NoError(t, err)

	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "frag", u.Fragment, "the fragment must stay a fragment")
	assert.Equal(t, "1", u.Query().Get("a"), "the parameter must be query, not fragment text")
}

// TestInjectQueryDoesNotDoubleQuestionMark covers a URL that already ends in "?".
func TestInjectQueryDoesNotDoubleQuestionMark(t *testing.T) {
	got, err := InjectQuery("https://example.com/p?", mustValues(t, "a", "1"))
	require.NoError(t, err)
	assert.NotContains(t, got, "??")

	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "1", u.Query().Get("a"))
}

// TestInjectQueryDestinationParamWins documents the collision rule: a parameter
// the upstream link already sets is not overridden by the forwarded one.
func TestInjectQueryDestinationParamWins(t *testing.T) {
	got, err := InjectQuery("https://example.com/p?token=upstream", mustValues(t, "token", "attacker"))
	require.NoError(t, err)
	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "upstream", u.Query().Get("token"))
}

func TestInjectQueryEmptyInputs(t *testing.T) {
	got, err := InjectQuery("https://example.com/p?a=1", url.Values{})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/p?a=1", got)

	_, err = InjectQuery("://not a url", mustValues(t, "a", "1"))
	assert.Error(t, err)
}

func TestInjectQueryMultipleValues(t *testing.T) {
	got, err := InjectQuery("https://example.com/p", mustValues(t, "a", "1", "a", "2"))
	require.NoError(t, err)
	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, u.Query()["a"])
}
