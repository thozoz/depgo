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

// TestSafeMoveRefusesExistingDestination is the baseline: moving onto an existing
// name must not destroy it.
func TestSafeMoveRefusesExistingDestination(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dst"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dst", "src.txt"), []byte("PRECIOUS"), 0o644))

	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}
	dstObj := &model.Object{Path: filepath.Join(root, "dst"), Name: "dst", IsFolder: true}

	err := svc.SafeMove(context.Background(), srcObj, dstObj, "/")
	require.ErrorIs(t, err, ErrDestinationExists)

	got, readErr := os.ReadFile(filepath.Join(root, "dst", "src.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "PRECIOUS", string(got), "the destination must be byte-for-byte untouched")

	// The source must still be where it was: a refused move moves nothing.
	src, srcErr := os.ReadFile(filepath.Join(root, "src.txt"))
	require.NoError(t, srcErr)
	assert.Equal(t, "SRC", string(src))
}

// TestSafeMoveSucceedsWhenDestinationFree covers the happy path.
func TestSafeMoveSucceedsWhenDestinationFree(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dst"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src.txt"), []byte("SRC-CONTENT"), 0o644))

	srcObj := &model.Object{Path: filepath.Join(root, "src.txt"), Name: "src.txt"}
	dstObj := &model.Object{Path: filepath.Join(root, "dst"), Name: "dst", IsFolder: true}

	require.NoError(t, svc.SafeMove(context.Background(), srcObj, dstObj, "/"))

	moved, err := os.ReadFile(filepath.Join(root, "dst", "src.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC-CONTENT", string(moved))

	// A move leaves no copy at the source.
	_, err = os.Stat(filepath.Join(root, "src.txt"))
	assert.True(t, os.IsNotExist(err), "the source must be gone after a move")
}
