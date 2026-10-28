//go:build windows

package localstate

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CheckSameFilesystem checks that pathA and pathB reside on the same drive/volume.
func CheckSameFilesystem(pathA, pathB string) error {
	absA, err := filepath.Abs(pathA)
	if err != nil {
		return fmt.Errorf("abs %s: %w", pathA, err)
	}
	absB, err := filepath.Abs(pathB)
	if err != nil {
		return fmt.Errorf("abs %s: %w", pathB, err)
	}

	volA := strings.ToUpper(filepath.VolumeName(absA))
	volB := strings.ToUpper(filepath.VolumeName(absB))

	if volA == "" || volB == "" {
		return fmt.Errorf("unable to determine volume names for %s and %s", pathA, pathB)
	}
	if volA != volB {
		return fmt.Errorf("cross-device storage state not allowed: root [%s] (volume %s) and state [%s] (volume %s) must reside on the same filesystem/drive", pathA, volA, pathB, volB)
	}
	return nil
}
