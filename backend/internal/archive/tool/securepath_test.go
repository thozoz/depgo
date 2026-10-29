package tool

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSecureJoin(t *testing.T) {
	baseDir := t.TempDir()
	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{name: "ok", entry: "a/b/c.txt", wantErr: false},
		{name: "parent", entry: "../evil.txt", wantErr: true},
		{name: "parent-backslash", entry: "..\\evil.txt", wantErr: true},
		{name: "abs", entry: "/tmp/evil.txt", wantErr: true},
		{name: "drive", entry: "C:\\evil.txt", wantErr: true},
		{name: "unc", entry: "\\\\server\\share\\evil.txt", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst, err := SecureJoin(baseDir, tc.entry)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.entry)
				}
				if !strings.Contains(err.Error(), tc.entry) {
					t.Fatalf("error should include entry name %q, got %q", tc.entry, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.entry, err)
			}
			rel, err := filepath.Rel(baseDir, dst)
			if err != nil {
				t.Fatalf("Rel failed: %v", err)
			}
			if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				t.Fatalf("path escaped baseDir: %q", dst)
			}
		})
	}
}

// TestSecureJoinRejectsWindowsPathsOnAnyHost is the regression test for a
// platform-dependent check: filepath.VolumeName reports "" for "C:/x" on Linux, so
// a drive-qualified archive entry passed the old guard. The same archive opened by
// the Windows build would treat it as absolute and escape the extraction root.
func TestSecureJoinRejectsWindowsPathsOnAnyHost(t *testing.T) {
	baseDir := t.TempDir()

	hostile := []string{
		`C:\evil.txt`,
		`C:/evil.txt`,
		`c:\evil.txt`,
		`Z:\Windows\System32\evil.dll`,
		`C:evil.txt`, // drive-relative
		`\\server\share\evil.txt`,
		`\\?\C:\evil.txt`, // extended-length prefix
	}
	for _, entry := range hostile {
		t.Run(entry, func(t *testing.T) {
			if _, err := SecureJoin(baseDir, entry); err == nil {
				t.Fatalf("SecureJoin(%q) must be rejected on every platform", entry)
			}
		})
	}

	// Legitimate names that merely contain a colon are unaffected.
	for _, entry := range []string{"notes:2024.txt", "dir/a:b.txt"} {
		if _, err := SecureJoin(baseDir, entry); err != nil {
			t.Errorf("SecureJoin(%q) should be accepted, got %v", entry, err)
		}
	}
}
