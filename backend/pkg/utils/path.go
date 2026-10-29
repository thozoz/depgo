package utils

import (
	"net/url"
	stdpath "path"
	"strings"

	"github.com/alist-org/alist/v3/internal/errs"
)

// FixAndCleanPath
// The upper layer of the root directory is still the root directory.
// So ".." And "." will be cleared
// for example
// 1. ".." or "." => "/"
// 2. "../..." or "./..." => "/..."
// 3. "../.x." or "./.x." => "/.x."
// 4. "x//\\y" = > "/z/x"
func FixAndCleanPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return stdpath.Clean(path)
}

// PathAddSeparatorSuffix Add path '/' suffix
// for example /root => /root/
func PathAddSeparatorSuffix(path string) string {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	return path
}

// PathEqual judge path is equal
func PathEqual(path1, path2 string) bool {
	return FixAndCleanPath(path1) == FixAndCleanPath(path2)
}

func IsSubPath(path string, subPath string) bool {
	path, subPath = FixAndCleanPath(path), FixAndCleanPath(subPath)
	return path == subPath || strings.HasPrefix(subPath, PathAddSeparatorSuffix(path))
}

func Ext(path string) string {
	ext := stdpath.Ext(path)
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}
	return strings.ToLower(ext)
}

func EncodePath(path string, all ...bool) string {
	seg := strings.Split(path, "/")
	toReplace := []struct {
		Src string
		Dst string
	}{
		{Src: "%", Dst: "%25"},
		{"%", "%25"},
		{"?", "%3F"},
		{"#", "%23"},
	}
	for i := range seg {
		if len(all) > 0 && all[0] {
			seg[i] = url.PathEscape(seg[i])
		} else {
			for j := range toReplace {
				seg[i] = strings.ReplaceAll(seg[i], toReplace[j].Src, toReplace[j].Dst)
			}
		}
	}
	return strings.Join(seg, "/")
}

func JoinBasePath(basePath, reqPath string) (string, error) {
	/** relative path:
	 * 1. ..
	 * 2. ../
	 * 3. /..
	 * 4. /../
	 * 5. /a/b/..
	 */
	// Normalise separators first: a Windows-style "..\" would otherwise slip
	// past the checks below and be silently collapsed by Join, quietly resolving
	// a parent-directory request to a name inside the base instead of refusing it.
	normalized := strings.ReplaceAll(reqPath, "\\", "/")
	if normalized == ".." ||
		strings.HasSuffix(normalized, "/..") ||
		strings.HasPrefix(normalized, "../") ||
		strings.Contains(normalized, "/../") {
		return "", errs.RelativePath
	}

	return stdpath.Join(FixAndCleanPath(basePath), FixAndCleanPath(reqPath)), nil
}

func GetFullPath(mountPath, path string) string {
	return stdpath.Join(GetActualMountPath(mountPath), path)
}

// ValidateNameComponent validates a single path component.
// It rejects empty names, dot segments, separators, ".." sequences, and NUL bytes.
func ValidateNameComponent(name string) error {
	if name == "" {
		return errs.InvalidName
	}
	if name == "." || name == ".." {
		return errs.InvalidName
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errs.InvalidName
	}
	if strings.Contains(name, "..") {
		return errs.InvalidName
	}
	if strings.ContainsRune(name, 0) {
		return errs.InvalidName
	}
	// Control characters are rejected outright. A newline is legal in a Linux file
	// name, which makes it usable to forge extra lines in the access log and to
	// break any header the name is interpolated into.
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return errs.InvalidName
		}
	}
	return nil
}

// SanitizeForHeader makes value safe to interpolate into an HTTP header value.
//
// File names are attacker-influenced (an uploader chooses them), and a quoted
// header parameter cannot carry a quote, a backslash or a line break without
// changing the meaning of the header. Those characters are removed rather than
// escaped, because the percent-encoded filename* parameter already carries the
// exact bytes for well-behaved clients.
func SanitizeForHeader(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r == '"' || r == '\\':
			continue
		case r < 0x20 || r == 0x7f:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// JoinUnderBase safely joins baseDir with a single name component and ensures the
// result stays under baseDir after normalization.
func JoinUnderBase(baseDir, name string) (string, error) {
	if err := ValidateNameComponent(name); err != nil {
		return "", err
	}
	base := FixAndCleanPath(baseDir)
	joined := FixAndCleanPath(stdpath.Join(base, name))
	if !IsSubPath(base, joined) {
		return "", errs.InvalidName
	}
	return joined, nil
}
