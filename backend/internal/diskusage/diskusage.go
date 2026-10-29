// Package diskusage reports how much space a directory tree takes and how much
// the filesystem under it has left.
package diskusage

import (
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// DirSize sums the sizes of regular files below root. Symlinks are not
// followed, and entries that vanish or cannot be read while walking are skipped:
// the number is a status display, not an accounting record.
func DirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

type cached struct {
	at   time.Time
	size int64
}

// Sizer memoises DirSize per path so that a page polling for the number does
// not walk a large tree on every request.
type Sizer struct {
	TTL time.Duration

	mu    sync.Mutex
	cache map[string]cached
}

func (s *Sizer) Size(root string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hit, ok := s.cache[root]; ok && time.Since(hit.at) < s.TTL {
		return hit.size
	}
	// shortcut: the walk runs under the lock, so concurrent callers wait for one
	// walk instead of starting several; fine for one admin, revisit for many.
	size := DirSize(root)
	if s.cache == nil {
		s.cache = make(map[string]cached)
	}
	s.cache[root] = cached{at: time.Now(), size: size}
	return size
}
