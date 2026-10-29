package diskusage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirSizeSumsFilesAndIgnoresMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, n := range map[string]int{"x": 10, "a/y": 20, "a/b/z": 30} {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := DirSize(dir); got != 60 {
		t.Fatalf("DirSize = %d, want 60", got)
	}
	if got := DirSize(filepath.Join(dir, "does-not-exist")); got != 0 {
		t.Fatalf("DirSize of a missing path = %d, want 0", got)
	}
}

func TestSizerCachesWithinTTL(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), make([]byte, 5), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Sizer{TTL: time.Hour}
	if got := s.Size(dir); got != 5 {
		t.Fatalf("first = %d, want 5", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "g"), make([]byte, 7), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := s.Size(dir); got != 5 {
		t.Fatalf("cached = %d, want 5 (still inside the TTL)", got)
	}
	s.TTL = 0
	if got := s.Size(dir); got != 12 {
		t.Fatalf("after TTL = %d, want 12", got)
	}
}

func TestFilesystemReportsSaneNumbers(t *testing.T) {
	total, free, err := Filesystem(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 || free > total {
		t.Fatalf("total=%d free=%d", total, free)
	}
}
