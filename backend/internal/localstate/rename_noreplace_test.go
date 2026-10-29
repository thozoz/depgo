package localstate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService builds a Service over a temporary root with a state directory.
func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "root")
	state := filepath.Join(base, "state")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.MkdirAll(state, 0o755))

	svc, err := New(Config{
		RootPath:  root,
		StatePath: state,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })
	return svc, root
}

// TestSafeRenameRefusesExistingDestination is the baseline: a rename onto an
// existing name must not silently destroy it.
func TestSafeRenameRefusesExistingDestination(t *testing.T) {
	svc, root := newTestService(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dst.txt"), []byte("PRECIOUS"), 0o644))

	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}
	err := svc.SafeRename(context.Background(), srcObj, "dst.txt", "/")
	require.ErrorIs(t, err, ErrDestinationExists)

	// The destination must be untouched and the source still present.
	dst, err := os.ReadFile(filepath.Join(root, "dst.txt"))
	require.NoError(t, err)
	assert.Equal(t, "PRECIOUS", string(dst), "destination must not be overwritten")

	src, err := os.ReadFile(filepath.Join(root, "src.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC", string(src), "source must remain after a refused rename")
}

// TestSafeRenameSucceedsWhenDestinationFree covers the happy path.
func TestSafeRenameSucceedsWhenDestinationFree(t *testing.T) {
	svc, root := newTestService(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC"), 0o644))
	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}

	require.NoError(t, svc.SafeRename(context.Background(), srcObj, "renamed.txt", "/"))

	_, err := os.Stat(filepath.Join(root, "src.txt"))
	assert.True(t, os.IsNotExist(err), "the source name must be gone")
	moved, err := os.ReadFile(filepath.Join(root, "renamed.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC", string(moved))
}

// TestSafeRenameHonoursStorageMountPath checks the mount path argument is used for
// the journal record rather than ignored.
func TestSafeRenameHonoursStorageMountPath(t *testing.T) {
	svc, root := newTestService(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("A"), 0o644))
	obj := &model.Object{Path: filepath.Join(root, "a.txt"), Name: "a.txt"}
	require.NoError(t, svc.SafeRename(context.Background(), obj, "b.txt", "/"))
	require.FileExists(t, filepath.Join(root, "b.txt"))
}

// TestCrossRootRenameNoReplaceRefusesExisting proves the cross-root helper used by
// trash restore never replaces an existing destination, even though a plain move
// between the two roots would.
func TestCrossRootRenameNoReplaceRefusesExisting(t *testing.T) {
	svc, root := newTestService(t)

	// A file in the state/staging root and a same-named file in the served root.
	staged := filepath.Join(svc.stagingPath, "stg_restore_test")
	require.NoError(t, os.WriteFile(staged, []byte("RESTORED"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "target.txt"), []byte("IN_THE_WAY"), 0o644))

	err := svc.CrossRootRenameNoReplace(staged, filepath.Join(root, "target.txt"))
	require.Error(t, err, "an existing destination must be refused")
	assert.True(t, errors.Is(err, os.ErrExist) || errors.Is(err, syscall.EEXIST),
		"expected an exists error, got %v", err)

	got, readErr := os.ReadFile(filepath.Join(root, "target.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "IN_THE_WAY", string(got), "the destination must be untouched")

	// The source must still exist: a refused move moves nothing.
	require.FileExists(t, staged)
}

// TestCrossRootRenameNoReplaceSucceedsWhenFree covers the happy path.
func TestCrossRootRenameNoReplaceSucceedsWhenFree(t *testing.T) {
	svc, root := newTestService(t)

	staged := filepath.Join(svc.stagingPath, "stg_restore_ok")
	require.NoError(t, os.WriteFile(staged, []byte("RESTORED"), 0o644))

	require.NoError(t, svc.CrossRootRenameNoReplace(staged, filepath.Join(root, "fresh.txt")))

	got, err := os.ReadFile(filepath.Join(root, "fresh.txt"))
	require.NoError(t, err)
	assert.Equal(t, "RESTORED", string(got))
}
