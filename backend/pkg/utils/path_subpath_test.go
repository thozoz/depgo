package utils

import (
	"strings"
	"testing"
)

// TestIsSubPathBoundaries documents the containment contract the share and
// archive handlers rely on. The interesting cases are the ones where a naive
// HasPrefix would say yes: "/a" must not contain "/ab", and any traversal
// segment must be normalised away before the comparison.
func TestIsSubPathBoundaries(t *testing.T) {
	cases := []struct {
		base string
		sub  string
		want bool
	}{
		{"/a", "/a", true},
		{"/a", "/a/b", true},
		{"/a/b", "/a", false},
		{"/a", "/ab", false},
		{"/a", "/a-b", false},
		{"/a", "/a/../b", false},
		{"/a/b", "/a/b/../../etc", false},
		{"/a", "/A/b", false},
		{"/", "/anything", true},
		{"/", "/", true},
		{"/a", "/a//b", true},
		// An empty base normalises to "/", i.e. the filesystem root, which
		// contains every absolute path.
		{"", "/x", true},
		{"", "", true},
	}
	for _, tc := range cases {
		if got := IsSubPath(tc.base, tc.sub); got != tc.want {
			t.Errorf("IsSubPath(%q, %q) = %v, want %v", tc.base, tc.sub, got, tc.want)
		}
	}
}

// TestValidateNameComponentRejectsTraversal locks the stricter rule used for a
// single path component such as an upload name.
func TestValidateNameComponentRejectsTraversal(t *testing.T) {
	// The check is deliberately blunt: any ".." substring is rejected, so
	// legitimate names like "..hidden" are refused too. That is the safe side of
	// the trade and the reason a name such as "notes..txt" needs renaming.
	rejected := []string{"", ".", "..", "a/b", `a\b`, "..x", "x..", "..hidden", "a\x00b"}
	for _, name := range rejected {
		if err := ValidateNameComponent(name); err == nil {
			t.Errorf("ValidateNameComponent(%q) should have failed", name)
		}
	}
	accepted := []string{"a", "file.txt", "a b", "a.b", "%2e%2e"}
	for _, name := range accepted {
		if err := ValidateNameComponent(name); err != nil {
			t.Errorf("ValidateNameComponent(%q) should have been accepted, got %v", name, err)
		}
	}
}

// TestJoinUnderBaseStaysInside proves the join helper refuses to leave the base
// directory no matter what the caller passes.
func TestJoinUnderBaseStaysInside(t *testing.T) {
	joined, err := JoinUnderBase("/base", "file.txt")
	if err != nil {
		t.Fatalf("expected a valid join, got %v", err)
	}
	if joined != "/base/file.txt" {
		t.Fatalf("joined path = %q, want /base/file.txt", joined)
	}

	for _, name := range []string{"..", "../etc/passwd", "a/../../etc", "", "."} {
		if _, err := JoinUnderBase("/base", name); err == nil {
			t.Errorf("JoinUnderBase(/base, %q) should have failed", name)
		}
	}
}

// TestValidateNameComponentRejectsControlCharacters covers the characters that
// break line-oriented output: a newline in a file name is legal on Linux, and it
// can forge extra access-log lines or corrupt any header the name is put into.
func TestValidateNameComponentRejectsControlCharacters(t *testing.T) {
	rejected := []string{
		"evil\nname.txt",
		"evil\rname.txt",
		"evil\r\nname.txt",
		"tab\there.txt",
		"nul\x00byte.txt",
		"del\x7fchar.txt",
		"bell\aname.txt",
	}
	for _, name := range rejected {
		if err := ValidateNameComponent(name); err == nil {
			t.Errorf("ValidateNameComponent(%q) should have been rejected", name)
		}
	}
}

// TestSanitizeForHeader keeps a quoted header parameter from being broken out of.
func TestSanitizeForHeader(t *testing.T) {
	cases := map[string]string{
		"plain.txt":          "plain.txt",
		`quote".txt`:         "quote.txt",
		`back\slash.txt`:     "backslash.txt",
		"new\nline.txt":      "newline.txt",
		"cr\rreturn.txt":     "crreturn.txt",
		"tab\there.txt":      "tabhere.txt",
		"日本語.txt":            "日本語.txt",
		`a"b\c` + "\n" + `d`: "abcd",
	}
	for input, want := range cases {
		if got := SanitizeForHeader(input); got != want {
			t.Errorf("SanitizeForHeader(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestIsSubPathSiblingPrefixes documents why the search filter must use IsSubPath
// rather than strings.HasPrefix: a base path of "/public" must not admit
// "/public-secret", which a raw prefix test does admit.
func TestIsSubPathSiblingPrefixes(t *testing.T) {
	const base = "/public"
	siblings := []string{"/public-secret/x", "/public2/y", "/publicity", "/public.bak/z"}
	for _, candidate := range siblings {
		if strings.HasPrefix(candidate, base) != true {
			t.Fatalf("fixture wrong: %q should share the string prefix", candidate)
		}
		if IsSubPath(base, candidate) {
			t.Errorf("IsSubPath(%q, %q) = true; a sibling prefix must not be contained", base, candidate)
		}
	}
	// The real children must still be contained.
	for _, child := range []string{"/public", "/public/", "/public/a/b"} {
		if !IsSubPath(base, child) {
			t.Errorf("IsSubPath(%q, %q) = false, want true", base, child)
		}
	}
}
