//go:build !linux

package localstate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFallbackNoReplaceFailsClosed(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	dst := filepath.Join(root, "target")
	if err := os.WriteFile(src, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, exists := range []bool{false, true} {
		if exists {
			if err := os.WriteFile(dst, []byte("precious"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		err := renameBetweenDirs(ownedDir{path: root}, ownedDir{path: root}, "source", "target", true)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Fatalf("expected unsupported atomic rename, got %v", err)
		}
		if _, err := os.Stat(src); err != nil {
			t.Fatal("source changed:", err)
		}
		if exists {
			body, err := os.ReadFile(dst)
			if err != nil || string(body) != "precious" {
				t.Fatalf("destination changed: %q, %v", body, err)
			}
		} else if _, err := os.Stat(dst); !os.IsNotExist(err) {
			t.Fatalf("unexpected destination: %v", err)
		}
	}
}
