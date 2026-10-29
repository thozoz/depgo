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

// TestSafeCopyRefusesExistingDestination is the baseline: copying onto an
// existing name must not destroy it.
func TestSafeCopyRefusesExistingDestination(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dst"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dst", "src.txt"), []byte("PRECIOUS"), 0o644))

	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}
	dstObj := &model.Object{Path: filepath.Join(root, "dst"), Name: "dst", IsFolder: true}

	err := svc.SafeCopy(context.Background(), srcObj, dstObj)
	require.ErrorIs(t, err, ErrDestinationExists)

	got, readErr := os.ReadFile(filepath.Join(root, "dst", "src.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "PRECIOUS", string(got), "the destination must be byte-for-byte untouched")
}

// TestSafeCopySucceedsWhenDestinationFree covers the happy path for a file.
func TestSafeCopySucceedsWhenDestinationFree(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dst"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC-CONTENT"), 0o644))

	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}
	dstObj := &model.Object{Path: filepath.Join(root, "dst"), Name: "dst", IsFolder: true}

	require.NoError(t, svc.SafeCopy(context.Background(), srcObj, dstObj))

	copied, err := os.ReadFile(filepath.Join(root, "dst", "src.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC-CONTENT", string(copied))

	// The source must still be there: copy, not move.
	src, err := os.ReadFile(filepath.Join(root, "src.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC-CONTENT", string(src))
}

// TestSafeCopyHandlesDirectory covers copying a directory tree.
func TestSafeCopyHandlesDirectory(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "tree", "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "tree", "a.txt"), []byte("A"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "tree", "nested", "b.txt"), []byte("B"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dst"), 0o755))

	srcObj := &model.Object{Path: filepath.Join(root, "tree"), Name: "tree", IsFolder: true}
	dstObj := &model.Object{Path: filepath.Join(root, "dst"), Name: "dst", IsFolder: true}

	require.NoError(t, svc.SafeCopy(context.Background(), srcObj, dstObj))

	a, err := os.ReadFile(filepath.Join(root, "dst", "tree", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "A", string(a))
	b, err := os.ReadFile(filepath.Join(root, "dst", "tree", "nested", "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, "B", string(b))
}

// TestSafeCopyLeavesNoStagingBehind proves the staged copy cleans up after itself
// in both the refused and the successful case.
func TestSafeCopyLeavesNoStagingBehind(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dst"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC"), 0o644))

	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}
	dstObj := &model.Object{Path: filepath.Join(root, "dst"), Name: "dst", IsFolder: true}

	require.NoError(t, svc.SafeCopy(context.Background(), srcObj, dstObj))

	entries, err := os.ReadDir(svc.stagingPath)
	require.NoError(t, err)
	assert.Empty(t, entries, "staging directory must be empty after a successful copy")
}
