package utils

import (
	"testing"

	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJoinBasePathRejectsTraversal covers the guard that every fs handler relies
// on before touching storage. The interesting cases are the variants that a
// literal string check would miss: Windows-style separators and the forms where
// ".." is not the whole component.
func TestJoinBasePathRejectsTraversal(t *testing.T) {
	rejected := []string{
		"..",
		"../x",
		"/..",
		"/../x",
		"a/../b",
		"a/..",
		`..\x`,
		`a\..\b`,
		`a/..\b`,
	}
	for _, req := range rejected {
		_, err := JoinBasePath("/base", req)
		assert.ErrorIsf(t, err, errs.RelativePath,
			"JoinBasePath(/base, %q) must be refused as a relative path", req)
	}
}

// TestJoinBasePathAcceptsOrdinaryPaths records the inputs that must keep working,
// so the guard above cannot be tightened into a usability regression.
func TestJoinBasePathAcceptsOrdinaryPaths(t *testing.T) {
	cases := map[string]string{
		"/a/b":        "/base/a/b",
		"a/b":         "/base/a/b",
		"....//x":     "/base/..../x",
		".../x":       "/base/.../x",
		"/base2/x":    "/base/base2/x",
		"a b/c d.txt": "/base/a b/c d.txt",
	}
	for req, want := range cases {
		got, err := JoinBasePath("/base", req)
		require.NoErrorf(t, err, "JoinBasePath(/base, %q)", req)
		assert.Equalf(t, want, got, "JoinBasePath(/base, %q)", req)
	}

	// A root base path maps through unchanged.
	got, err := JoinBasePath("/", "/a/b")
	require.NoError(t, err)
	assert.Equal(t, "/a/b", got)
}

// TestJoinBasePathStaysInsideBase asserts the result never escapes the base
// directory for an accepted input.
func TestJoinBasePathStaysInsideBase(t *testing.T) {
	inputs := []string{"a", "a/b", "....//x", ".../x", "/base2/x", "a//b", "./a"}
	for _, req := range inputs {
		got, err := JoinBasePath("/base", req)
		require.NoErrorf(t, err, "JoinBasePath(/base, %q)", req)
		assert.Truef(t, IsSubPath("/base", got),
			"JoinBasePath(/base, %q) = %q escaped the base directory", req, got)
	}
}
