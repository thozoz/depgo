package conf

import (
	"regexp"
	"sync"
)

// Settings that are derived from the database are read on hot request paths
// (file listing, download, proxy, preview, error masking) and rewritten whenever
// an administrator saves settings. Plain package-level maps and slices cannot
// serve that pattern: Go treats a concurrent map read and map write as
// unrecoverable and aborts the process, so a settings save racing with a
// download would kill the server outright - no panic recovery can catch it.
//
// The values are therefore published as immutable snapshots behind an RWMutex.
// Readers get a copy they own for the duration of the call; writers replace a
// whole value instead of mutating one in place.

var derivedSettings struct {
	mu sync.RWMutex
	// slices holds the comma-separated settings (file type lists, ignored
	// headers, ignore paths, ...) keyed by setting name.
	slices map[string][]string
	// filenameCharMap maps a character that cannot appear in a filename onto
	// the replacement used when displaying it.
	filenameCharMap map[string]string
	// privacyReg holds the compiled expressions whose matches are masked in
	// error messages and logs.
	privacyReg []*regexp.Regexp
}

func init() {
	derivedSettings.slices = make(map[string][]string)
	derivedSettings.filenameCharMap = make(map[string]string)
}

// SetSlice publishes a snapshot of a comma-separated setting. The caller may
// reuse or modify the passed slice afterwards.
func SetSlice(key string, values []string) {
	copied := make([]string, len(values))
	copy(copied, values)

	derivedSettings.mu.Lock()
	defer derivedSettings.mu.Unlock()
	derivedSettings.slices[key] = copied
}

// GetSlice returns the snapshot for a comma-separated setting. The returned
// slice must not be modified; it is shared with other readers.
func GetSlice(key string) []string {
	derivedSettings.mu.RLock()
	defer derivedSettings.mu.RUnlock()
	return derivedSettings.slices[key]
}

// SetFilenameCharMap publishes a snapshot of the filename character mapping.
func SetFilenameCharMap(m map[string]string) {
	copied := make(map[string]string, len(m))
	for k, v := range m {
		copied[k] = v
	}

	derivedSettings.mu.Lock()
	defer derivedSettings.mu.Unlock()
	derivedSettings.filenameCharMap = copied
}

// GetFilenameCharMap returns the current filename character mapping. The
// returned map must not be modified.
func GetFilenameCharMap() map[string]string {
	derivedSettings.mu.RLock()
	defer derivedSettings.mu.RUnlock()
	return derivedSettings.filenameCharMap
}

// SetPrivacyReg publishes a snapshot of the compiled privacy expressions.
func SetPrivacyReg(regs []*regexp.Regexp) {
	copied := make([]*regexp.Regexp, len(regs))
	copy(copied, regs)

	derivedSettings.mu.Lock()
	defer derivedSettings.mu.Unlock()
	derivedSettings.privacyReg = copied
}

// GetPrivacyReg returns the current compiled privacy expressions. The returned
// slice must not be modified.
func GetPrivacyReg() []*regexp.Regexp {
	derivedSettings.mu.RLock()
	defer derivedSettings.mu.RUnlock()
	return derivedSettings.privacyReg
}

// ResetDerivedSettings clears every snapshot. It exists for tests that need a
// clean slate between cases.
func ResetDerivedSettings() {
	derivedSettings.mu.Lock()
	defer derivedSettings.mu.Unlock()
	derivedSettings.slices = make(map[string][]string)
	derivedSettings.filenameCharMap = make(map[string]string)
	derivedSettings.privacyReg = nil
}
