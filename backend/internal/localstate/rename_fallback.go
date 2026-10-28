//go:build !linux

package localstate

import (
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
		if _, err := os.Lstat(dstPath); err == nil {
			return os.ErrExist
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(srcPath, dstPath)
}
