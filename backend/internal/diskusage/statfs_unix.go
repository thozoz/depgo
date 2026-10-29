//go:build !windows

package diskusage

import "syscall"

// Filesystem returns the total and the available bytes of the filesystem that
// holds path. Available means what an unprivileged process may still write.
func Filesystem(path string) (total, free uint64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	bsize := uint64(st.Bsize)
	return uint64(st.Blocks) * bsize, uint64(st.Bavail) * bsize, nil
}
