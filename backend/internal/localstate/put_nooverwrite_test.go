package localstate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUploadStream builds a FileStream for upload tests.
func newUploadStream(name, content string, noOverwrite bool) *stream.FileStream {
	return &stream.FileStream{
		Obj: &model.Object{
			Name:     name,
			Size:     int64(len(content)),
			HashInfo: utils.NewHashInfoByMap(nil),
		},
		Reader:      strings.NewReader(content),
		Mimetype:    "text/plain",
		NoOverwrite: noOverwrite,
	}
}

// TestSafePutRefusesExistingDestination is the regression test for "Overwrite:
// false" being enforced only in the HTTP handler: the storage layer replaced
// whatever was there, so a file created between the handler's check and the write
// was lost.
func TestSafePutRefusesExistingDestination(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "target.txt"), []byte("PRECIOUS"), 0o644))

	dstDir := &model.Object{Path: root, Name: "root", IsFolder: true}
	err := svc.SafePut(context.Background(), dstDir, newUploadStream("target.txt", "NEW", true), func(float64) {})
	require.ErrorIs(t, err, ErrDestinationExists)

	got, readErr := os.ReadFile(filepath.Join(root, "target.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "PRECIOUS", string(got), "the existing file must be untouched")
}

// TestSafePutOverwritesWhenAllowed is the default behaviour: an upload without
// the flag replaces the file (after snapshotting it to version history).
func TestSafePutOverwritesWhenAllowed(t *testing.T) {
	svc, root := newTestService(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "target.txt"), []byte("OLD"), 0o644))

	dstDir := &model.Object{Path: root, Name: "root", IsFolder: true}
	require.NoError(t, svc.SafePut(context.Background(), dstDir, newUploadStream("target.txt", "NEW", false), func(float64) {}))

	got, err := os.ReadFile(filepath.Join(root, "target.txt"))
	require.NoError(t, err)
	assert.Equal(t, "NEW", string(got))
}

// TestSafePutCreatesWhenDestinationFree covers the no-overwrite case against a
// free name: it must succeed, not fail.
func TestSafePutCreatesWhenDestinationFree(t *testing.T) {
	svc, root := newTestService(t)

	dstDir := &model.Object{Path: root, Name: "root", IsFolder: true}
	require.NoError(t, svc.SafePut(context.Background(), dstDir, newUploadStream("fresh.txt", "FRESH", true), func(float64) {}))

	got, err := os.ReadFile(filepath.Join(root, "fresh.txt"))
	require.NoError(t, err)
	assert.Equal(t, "FRESH", string(got))
}

var _ driver.UpdateProgress = func(float64) {}
