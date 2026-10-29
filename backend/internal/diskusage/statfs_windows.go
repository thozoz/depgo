//go:build windows

package diskusage

import "golang.org/x/sys/windows"

// Filesystem returns the total and the available bytes of the volume that holds
// path. Available means what the calling user may still write (quota aware).
func Filesystem(path string) (total, free uint64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var avail, tot, totalFree uint64
	if err = windows.GetDiskFreeSpaceEx(p, &avail, &tot, &totalFree); err != nil {
		return 0, 0, err
	}
	return tot, avail, nil
}
