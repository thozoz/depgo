//go:build !linux

package localstate

import (
	"errors"
	"os"
	"path/filepath"
)

// renameBetweenDirs performs a path-based rename. Platforms without
// renameat(2) (notably Windows) cannot pin a directory across the rename, so
// the caller's ownership and root-anchored parent checks narrow, but do not
// fully eliminate, a concurrent symlink swap in the parent path. The final
// component itself is never followed.
func renameBetweenDirs(src, dst ownedDir, srcName, dstName string, noReplace bool) error {
	srcPath := filepath.Join(src.path, srcName)
	dstPath := filepath.Join(dst.path, dstName)

	if noReplace {
		// A check followed by Rename cannot guarantee no replacement.
		return &os.LinkError{Op: "rename-noreplace", Old: srcPath, New: dstPath, Err: errors.ErrUnsupported}
	}
	return os.Rename(srcPath, dstPath)
}
