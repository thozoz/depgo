package tool

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ErrArchiveIllegalPath indicates an archive entry path is unsafe for extraction.
var ErrArchiveIllegalPath = errors.New("archive entry has illegal path")

// looksLikeWindowsAbs reports whether a slash-normalised entry name is a Windows
// absolute path: a drive letter followed by a colon ("C:/x"), or a UNC prefix.
//
// The check is deliberately independent of the host platform. Both the Windows and
// the Unix builds can read the same archive, so an entry that is absolute on one
// platform has to be rejected on both.
func looksLikeWindowsAbs(p string) bool {
	if strings.HasPrefix(p, "//") {
		return true // UNC, already slash-normalised
	}
	if len(p) < 2 || p[1] != ':' {
		return false
	}
	c := p[0]
	if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
		return false
	}
	// "C:evil.txt" is a drive-relative path, which also escapes the base when
	// interpreted by the Windows API, so require a separator after the colon too
	// OR treat any drive-qualified name as unsafe.
	return true
}

// SecureJoin returns a safe extraction path for an archive entry.
// It rejects absolute paths, traversal, Windows drive/UNC paths, and NUL bytes.
func SecureJoin(baseDir, entryName string) (string, error) {
	if strings.Contains(entryName, "\x00") {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}

	normalized := strings.ReplaceAll(entryName, "\\", "/")
	if strings.HasPrefix(normalized, "//") {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}
	cleaned := path.Clean(normalized)

	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}

	// Reject Windows drive-letter and UNC forms explicitly rather than relying on
	// filepath.VolumeName, which is platform-dependent: on Linux VolumeName("C:/x")
	// is empty, so a "C:\evil.txt" entry extracted on Linux lands safely inside the
	// base directory - but the same archive extracted by the Windows build would
	// interpret the drive letter and escape. An archive entry is attacker-controlled
	// data that must be judged the same way on every platform.
	if looksLikeWindowsAbs(cleaned) {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}

	rel := filepath.FromSlash(cleaned)
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}
	if strings.HasPrefix(rel, `\\`) {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}

	base := filepath.Clean(baseDir)
	dst := filepath.Join(base, rel)

	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("%w: %s (%v)", ErrArchiveIllegalPath, entryName, err)
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return "", fmt.Errorf("%w: %s (%v)", ErrArchiveIllegalPath, entryName, err)
	}

	relCheck, err := filepath.Rel(baseAbs, dstAbs)
	if err != nil {
		return "", fmt.Errorf("%w: %s (%v)", ErrArchiveIllegalPath, entryName, err)
	}
	if relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", ErrArchiveIllegalPath, entryName)
	}
	return dst, nil
}
