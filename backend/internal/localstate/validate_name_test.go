package localstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateNameComponentRejectsControlCharacters keeps invisible bytes out of
// names, so one file cannot masquerade as several lines in logs or listings.
func TestValidateNameComponentRejectsControlCharacters(t *testing.T) {
	for _, bad := range []string{"a\nb", "a\rb", "a\tb", "a\x7fb", "a\x01b"} {
		assert.ErrorIs(t, ValidateNameComponent(bad), ErrInvalidName, "name %q must be rejected", bad)
	}
	for _, ok := range []string{"a b", "rapor-2026.txt", "üşü", "a.b"} {
		assert.NoError(t, ValidateNameComponent(ok), "name %q must stay valid", ok)
	}
}

// TestSafeMakeDirRejectsControlCharacters checks the validator is actually wired
// into the mutation path rather than only existing.
func TestSafeMakeDirRejectsControlCharacters(t *testing.T) {
	svc, root := newTestService(t)
	parent := &model.Object{Path: root, Name: "root", IsFolder: true}

	err := svc.SafeMakeDir(context.Background(), parent, "evil\nname", 0o755)
	require.ErrorIs(t, err, ErrInvalidName)

	entries, readErr := os.ReadDir(root)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "nothing may be created for a rejected name")

	_, statErr := os.Stat(filepath.Join(root, "evil\nname"))
	assert.True(t, os.IsNotExist(statErr))
}

// TestSafeRenameRejectsControlCharacters covers the rename path too.
func TestSafeRenameRejectsControlCharacters(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC"), 0o644))
	obj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}

	err := svc.SafeRename(context.Background(), obj, "dst\nname", "/")
	require.ErrorIs(t, err, ErrInvalidName)
	require.FileExists(t, filepath.Join(root, "src.txt"))
}
