package localstate

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestServiceWithDB builds a Service over a temporary root backed by an
// in-memory journal, so tests can hook the journal write.
func newTestServiceWithDB(t *testing.T) (*Service, string, *gorm.DB) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "root")
	state := filepath.Join(base, "state")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.MkdirAll(state, 0o755))

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.FileRecord{}, &model.FileVersion{}, &model.PendingOperation{}))

	svc, err := New(Config{RootPath: root, StatePath: state, DB: db})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })
	return svc, root, db
}

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

// racingReader creates the destination on its first Read, simulating another
// writer landing at the target name while the body is still being staged.
type racingReader struct {
	r    io.Reader
	dst  string
	body string
	once sync.Once
	err  error
}

func (r *racingReader) Read(p []byte) (int, error) {
	r.once.Do(func() { r.err = os.WriteFile(r.dst, []byte(r.body), 0o644) })
	if r.err != nil {
		return 0, r.err
	}
	return r.r.Read(p)
}

// TestSafePutNoOverwriteRefusesDestinationCreatedDuringStaging is deterministic:
// the pre-check runs against a free name, the destination appears mid-upload, and
// the write must still fail closed instead of replacing it.
func TestSafePutNoOverwriteRefusesDestinationCreatedDuringStaging(t *testing.T) {
	svc, root := newTestService(t)
	dst := filepath.Join(root, "target.txt")

	racer := &racingReader{r: strings.NewReader("NEW"), dst: dst, body: "RACED"}
	s := newUploadStream("target.txt", "NEW", true)
	s.Reader = racer

	dstDir := &model.Object{Path: root, Name: "root", IsFolder: true}
	err := svc.SafePut(context.Background(), dstDir, s, func(float64) {})
	require.NoError(t, racer.err, "the raced-in destination must actually be created")
	require.ErrorIs(t, err, ErrDestinationExists)

	got, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, "RACED", string(got), "the raced-in file must be untouched")

	entries, readDirErr := os.ReadDir(svc.stagingPath)
	require.NoError(t, readDirErr)
	assert.Empty(t, entries, "a refused no-overwrite upload must leave no staging behind")
}

// TestSafePutNoOverwritePublishRefusesLateDestination targets the window the
// second Stat cannot see: the destination is created *after* that check and
// *before* the final publish, so only an atomic no-replace rename can refuse it.
// A GORM create callback on the journal write lands exactly in that window,
// deterministically and without sleeps.
func TestSafePutNoOverwritePublishRefusesLateDestination(t *testing.T) {
	svc, root, db := newTestServiceWithDB(t)
	dst := filepath.Join(root, "target.txt")

	var (
		mu          sync.Mutex
		fired       bool
		raceWriteEr error
	)
	require.NoError(t, db.Callback().Create().After("gorm:create").Register("test:race_late_destination", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil || tx.Statement.Schema.Table != "pending_operations" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if fired {
			return
		}
		fired = true
		raceWriteEr = os.WriteFile(dst, []byte("RACED"), 0o644)
	}))

	dstDir := &model.Object{Path: root, Name: "root", IsFolder: true}
	err := svc.SafePut(context.Background(), dstDir, newUploadStream("target.txt", "NEW", true), func(float64) {})
	require.ErrorIs(t, err, ErrDestinationExists)

	mu.Lock()
	didFire, writeErr := fired, raceWriteEr
	mu.Unlock()
	require.True(t, didFire, "the callback must have created the destination before publish")
	require.NoError(t, writeErr)

	got, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, "RACED", string(got), "the late destination must be untouched by publish")

	entries, readDirErr := os.ReadDir(svc.stagingPath)
	require.NoError(t, readDirErr)
	assert.Empty(t, entries, "a refused publish must leave no staging behind")
}
