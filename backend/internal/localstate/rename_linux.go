//go:build linux

package localstate

import "golang.org/x/sys/unix"

// renameBetweenDirs renames srcName inside src to dstName inside dst using
// only the opened directory descriptors, so no path component can be
// redirected outside the root-anchored directories. When noReplace is set the
// destination is never overwritten.
func renameBetweenDirs(src, dst ownedDir, srcName, dstName string, noReplace bool) error {
	srcFD := int(src.file.Fd())
	dstFD := int(dst.file.Fd())

	if !noReplace {
		return unix.Renameat(srcFD, srcName, dstFD, dstName)
	}

	// Fail closed when the filesystem cannot guarantee atomic no-overwrite.
	return unix.Renameat2(srcFD, srcName, dstFD, dstName, unix.RENAME_NOREPLACE)
}
