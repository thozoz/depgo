package local_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/drivers/local"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/localstate"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLocalDriverLifecycle(t *testing.T) {
	baseDir := t.TempDir()
	rootDir := filepath.Join(baseDir, "storage_root")
	stateDir := filepath.Join(baseDir, "storage_state")
	require.NoError(t, os.MkdirAll(rootDir, 0755))
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	db.Init(gormDB)

	d := &local.Local{
		Storage: model.Storage{
			ID:        42,
			MountPath: "/local",
		},
		Addition: local.Addition{
			RootPath:    driver.RootPath{RootFolderPath: rootDir},
			StateFolder: stateDir,
			MkdirPerm:   "755",
		},
	}
	ctx := context.Background()
	require.NoError(t, d.Init(ctx))
	defer func() { _ = d.Drop(ctx) }()

	require.NotNil(t, d.State())

	// 1. Put a file
	rootObj := &model.Object{Path: rootDir, IsFolder: true}
	content1 := []byte("first content")
	s1 := &stream.FileStream{
		Obj:    &model.Object{Name: "sample.txt", Size: int64(len(content1)), Modified: time.Now()},
		Reader: bytes.NewReader(content1),
	}
	require.NoError(t, d.Put(ctx, rootObj, s1, nil))

	// Verify file on disk
	samplePath := filepath.Join(rootDir, "sample.txt")
	got, err := os.ReadFile(samplePath)
	require.NoError(t, err)
	assert.Equal(t, content1, got)

	// 2. Put overwrite creates version
	content2 := []byte("second revised content")
	s2 := &stream.FileStream{
		Obj:    &model.Object{Name: "sample.txt", Size: int64(len(content2)), Modified: time.Now()},
		Reader: bytes.NewReader(content2),
	}
	require.NoError(t, d.Put(ctx, rootObj, s2, nil))

	got2, err := os.ReadFile(samplePath)
	require.NoError(t, err)
	assert.Equal(t, content2, got2)

	versions, err := d.ListVersions(ctx, "/sample.txt")
	require.NoError(t, err)
	require.Len(t, versions, 1)

	// 3. Rename same-name is no-op
	sampleObj := &model.Object{Path: samplePath, Name: "sample.txt"}
	require.NoError(t, d.Rename(ctx, sampleObj, "sample.txt"))

	// 4. Rename to new name
	require.NoError(t, d.Rename(ctx, sampleObj, "renamed.txt"))
	renamedPath := filepath.Join(rootDir, "renamed.txt")
	assert.FileExists(t, renamedPath)
	assert.NoFileExists(t, samplePath)

	// 5. Remove sends to trash
	renamedObj := &model.Object{Path: renamedPath, Name: "renamed.txt"}
	require.NoError(t, d.Remove(ctx, renamedObj))
	assert.NoFileExists(t, renamedPath)

	trashItems, err := d.ListTrash(ctx)
	require.NoError(t, err)
	require.Len(t, trashItems, 1)
	assert.Equal(t, "/renamed.txt", trashItems[0].OriginalPath)

	// 6. Restore from trash
	restoredRec, err := d.RestoreTrash(ctx, trashItems[0].TrashID, "")
	require.NoError(t, err)
	assert.Equal(t, "renamed.txt", restoredRec.RelativePath)
	assert.FileExists(t, renamedPath)

	// 7. Disjoint validation check
	nestedAddition := local.Addition{
		RootPath:    driver.RootPath{RootFolderPath: rootDir},
		StateFolder: filepath.Join(rootDir, "state_nested"),
	}
	dNested := &local.Local{
		Storage:  model.Storage{ID: 99},
		Addition: nestedAddition,
	}
	err = dNested.Init(ctx)
	assert.Error(t, err, "nested state folder must be rejected")
}

var _ = localstate.ErrDestinationExists
