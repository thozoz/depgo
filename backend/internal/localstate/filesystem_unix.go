//go:build !windows

package localstate

import (
	"fmt"
	"os"
	"syscall"
)

// CheckSameFilesystem checks that pathA and pathB reside on the same filesystem (same st_dev).
func CheckSameFilesystem(pathA, pathB string) error {
	fiA, err := os.Stat(pathA)
	if err != nil {
		return fmt.Errorf("stat %s: %w", pathA, err)
	}
	fiB, err := os.Stat(pathB)
	if err != nil {
		return fmt.Errorf("stat %s: %w", pathB, err)
	}
	statA, okA := fiA.Sys().(*syscall.Stat_t)
	statB, okB := fiB.Sys().(*syscall.Stat_t)
	if !okA || !okB {
		return fmt.Errorf("unable to retrieve filesystem device IDs for %s and %s", pathA, pathB)
	}
	if statA.Dev != statB.Dev {
		return fmt.Errorf("cross-device storage state not allowed: root [%s] (dev %d) and state [%s] (dev %d) must reside on the same filesystem", pathA, statA.Dev, pathB, statB.Dev)
	}
	return nil
}
