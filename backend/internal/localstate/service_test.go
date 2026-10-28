package localstate_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/localstate"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&model.FileRecord{},
		&model.TrashItem{},
		&model.FileVersion{},
		&model.PendingOperation{},
		&model.Share{},
	)
	require.NoError(t, err)
	return db
}

func setupTestEnv(t *testing.T) (*localstate.Service, string, string, *gorm.DB) {
	baseDir := t.TempDir()
	rootDir := filepath.Join(baseDir, "root")
	stateDir := filepath.Join(baseDir, "state")
	require.NoError(t, os.MkdirAll(rootDir, 0755))
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	db := setupTestDB(t)

	svc, err := localstate.New(localstate.Config{
		StorageID: 1,
		RootPath:  rootDir,
		StatePath: stateDir,
		DB:        db,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = svc.Close()
	})

	return svc, rootDir, stateDir, db
}

func TestCanonicalRelPathAndLinuxPaths(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)

	// 1. Storage-relative paths with leading slashes must not be treated as disk absolute
	p, err := svc.CanonicalRelPath("/foo/bar.txt")
	require.NoError(t, err)
	assert.Equal(t, "foo/bar.txt", p)

	pRoot, err := svc.CanonicalRelPath("/")
	require.NoError(t, err)
	assert.Equal(t, "", pRoot)

	// Legitimate dot-prefixed filenames must not be rejected
	pDot, err := svc.CanonicalRelPath("/..name.txt")
	require.NoError(t, err)
	assert.Equal(t, "..name.txt", pDot)

	pDoubleDotFile, err := svc.CanonicalRelPath("/dir/..custom")
	require.NoError(t, err)
	assert.Equal(t, "dir/..custom", pDoubleDotFile)

	// Escapes must be rejected
	_, err = svc.CanonicalRelPath("/../etc/passwd")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrPathEscapesRoot))

	_, err = svc.CanonicalRelPath("dir/../../outside")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrPathEscapesRoot))

	// Invalid component names
	_, err = svc.CanonicalRelPath("/foo/bad\x00name")
	assert.Error(t, err)

	// 2. DiskToRelative on absolute paths
	childDisk := filepath.Join(rootDir, "sub", "doc.txt")
	rel, err := svc.DiskToRelative(childDisk)
	require.NoError(t, err)
	assert.Equal(t, "sub/doc.txt", rel)

	// Outside disk path must fail
	outsideDisk := filepath.Join(t.TempDir(), "outside.txt")
	_, err = svc.DiskToRelative(outsideDisk)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrPathEscapesRoot))
}

func TestDisjointValidationWithSymlinks(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "served")
	state := filepath.Join(base, "state")
	require.NoError(t, os.MkdirAll(root, 0755))
	require.NoError(t, os.MkdirAll(state, 0755))

	// Direct nested must fail
	nested := filepath.Join(root, "nested_state")
	require.NoError(t, os.MkdirAll(nested, 0755))
	err := localstate.ValidateDisjoint(root, nested)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrNestedStateRoot))

	// Sibling succeeds
	assert.NoError(t, localstate.ValidateDisjoint(root, state))

	// Symlink aliasing: symlink inside root pointing to state directory must fail
	symlinkInRoot := filepath.Join(root, "symlink_to_state")
	if errSym := os.Symlink(state, symlinkInRoot); errSym == nil {
		err = localstate.ValidateDisjoint(root, symlinkInRoot)
		assert.Error(t, err, "symlink aliasing between root and state must be rejected")
	}
}

func TestSymlinkForbiddenOnMutationsAndReads(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)
	ctx := context.Background()

	// Create an external secret file outside root
	externalDir := t.TempDir()
	secretFile := filepath.Join(externalDir, "secret.key")
	require.NoError(t, os.WriteFile(secretFile, []byte("SUPER_SECRET"), 0644))

	// Create a symlink inside root pointing outside
	symlinkPath := filepath.Join(rootDir, "escape_link")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Skip("Symlinks not supported in current environment; skipping symlink test")
	}

	// 1. CheckSymlink on the symlink directly
	err := svc.CheckSymlink("escape_link")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrSymlinkForbidden))

	// 2. SafePut targeting a path under symlink must fail
	parentObj := &model.Object{Path: symlinkPath, IsFolder: true}
	s := &stream.FileStream{
		Obj:    &model.Object{Name: "data.txt", Size: 4, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("test")),
	}
	err = svc.SafePut(ctx, parentObj, s, nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrSymlinkForbidden))
}

func TestPutSafeStageAndOldContentSurvivesError(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)
	ctx := context.Background()

	parentDir := &model.Object{Path: rootDir, IsFolder: true}

	initialContent := []byte("hello original content")
	s1 := &stream.FileStream{
		Obj:    &model.Object{Name: "doc.txt", Size: int64(len(initialContent)), Modified: time.Now()},
		Reader: bytes.NewReader(initialContent),
	}
	err := svc.SafePut(ctx, parentDir, s1, nil)
	require.NoError(t, err)

	docPath := filepath.Join(rootDir, "doc.txt")
	data, err := os.ReadFile(docPath)
	require.NoError(t, err)
	assert.Equal(t, initialContent, data)

	// Stream copy error leaves original intact
	errReader := &failingReader{data: []byte("partial"), failAt: 3}
	sFail := &stream.FileStream{
		Obj:    &model.Object{Name: "doc.txt", Size: 100, Modified: time.Now()},
		Reader: errReader,
	}
	err = svc.SafePut(ctx, parentDir, sFail, nil)
	assert.Error(t, err)

	dataAfterErr, err := os.ReadFile(docPath)
	require.NoError(t, err)
	assert.Equal(t, initialContent, dataAfterErr)
}

func TestPutSnapshotBeforeOverwriteAndRestoreVersion(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)
	ctx := context.Background()

	parentDir := &model.Object{Path: rootDir, IsFolder: true}

	// Version 1
	v1Content := []byte("version 1 content")
	s1 := &stream.FileStream{
		Obj:    &model.Object{Name: "file.txt", Size: int64(len(v1Content)), Modified: time.Now()},
		Reader: bytes.NewReader(v1Content),
	}
	require.NoError(t, svc.SafePut(ctx, parentDir, s1, nil))

	rec1, err := svc.GetActiveFileRecord(ctx, "/file.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rec1.Revision)

	// Overwrite upload -> Version 2
	v2Content := []byte("version 2 new content")
	s2 := &stream.FileStream{
		Obj:    &model.Object{Name: "file.txt", Size: int64(len(v2Content)), Modified: time.Now()},
		Reader: bytes.NewReader(v2Content),
	}
	require.NoError(t, svc.SafePut(ctx, parentDir, s2, nil))

	rec2, err := svc.GetActiveFileRecord(ctx, "/file.txt")
	require.NoError(t, err)
	assert.Equal(t, rec1.FileID, rec2.FileID)
	assert.Equal(t, int64(2), rec2.Revision)

	versions, err := svc.ListFileVersions(ctx, rec1.FileID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, int64(1), versions[0].Revision)

	// Restore Version 1: current version 2 must be preserved as snapshot in history!
	restoredRec, err := svc.RestoreVersion(ctx, rec1.FileID, versions[0].VersionID, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), restoredRec.Revision)

	restoredData, err := os.ReadFile(filepath.Join(rootDir, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, v1Content, restoredData)

	// History now has 2 snapshots (v1 before v2 overwrite, and v2 before v1 restore)
	allVersions, err := svc.ListFileVersions(ctx, rec1.FileID)
	require.NoError(t, err)
	assert.Len(t, allVersions, 2)
}

func TestMoveAndRenameNoOverwriteAndSameName(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)
	ctx := context.Background()

	parentDir := &model.Object{Path: rootDir, IsFolder: true}
	sA := &stream.FileStream{
		Obj:    &model.Object{Name: "A.txt", Size: 4, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("fileA")),
	}
	sB := &stream.FileStream{
		Obj:    &model.Object{Name: "B.txt", Size: 4, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("fileB")),
	}
	require.NoError(t, svc.SafePut(ctx, parentDir, sA, nil))
	require.NoError(t, svc.SafePut(ctx, parentDir, sB, nil))

	// Same-name rename is no-op
	objA := &model.Object{Path: filepath.Join(rootDir, "A.txt"), Name: "A.txt"}
	err := svc.SafeRename(ctx, objA, "A.txt", "")
	assert.NoError(t, err)

	// Renaming A to existing B must fail without overwrite
	err = svc.SafeRename(ctx, objA, "B.txt", "")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrDestinationExists))

	// Content of B remains intact
	dataB, err := os.ReadFile(filepath.Join(rootDir, "B.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("fileB"), dataB)
}

func TestMoveUpdatesDescendantFileRecordsAndShares(t *testing.T) {
	svc, rootDir, _, db := setupTestEnv(t)
	ctx := context.Background()

	subDirDisk := filepath.Join(rootDir, "sub")
	require.NoError(t, os.MkdirAll(subDirDisk, 0755))
	subObj := &model.Object{Path: subDirDisk, IsFolder: true}

	sNote := &stream.FileStream{
		Obj:    &model.Object{Name: "note.txt", Size: 5, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("notes")),
	}
	require.NoError(t, svc.SafePut(ctx, subObj, sNote, nil))

	recNote, err := svc.GetActiveFileRecord(ctx, "/sub/note.txt")
	require.NoError(t, err)

	// Create bound Share for /sub/note.txt
	share := &model.Share{
		ShareID:      "sh123",
		Name:         "note.txt",
		RootPath:     "/local/sub/note.txt",
		FileRecordID: recNote.FileID,
	}
	require.NoError(t, db.Create(share).Error)

	destDirDisk := filepath.Join(rootDir, "dest")
	require.NoError(t, os.MkdirAll(destDirDisk, 0755))
	destDirObj := &model.Object{Path: destDirDisk, IsFolder: true}

	// Move "sub" folder to "dest/sub"
	folderSubObj := &model.Object{Path: subDirDisk, Name: "sub", IsFolder: true}
	err = svc.SafeMove(ctx, folderSubObj, destDirObj, "local")
	require.NoError(t, err)

	// Verify descendant FileRecord updated to "dest/sub/note.txt"
	recMoved, err := svc.GetFileRecordByID(ctx, recNote.FileID)
	require.NoError(t, err)
	assert.Equal(t, "dest/sub/note.txt", recMoved.RelativePath)

	// Verify Share.RootPath was automatically updated
	var updatedShare model.Share
	require.NoError(t, db.Where("share_id = ?", "sh123").Take(&updatedShare).Error)
	assert.Equal(t, "/local/dest/sub/note.txt", updatedShare.RootPath)
}

func TestTrashAndRestorePreventsOverwrite(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)
	ctx := context.Background()

	parentDir := &model.Object{Path: rootDir, IsFolder: true}
	s := &stream.FileStream{
		Obj:    &model.Object{Name: "secret.txt", Size: 6, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("secret")),
	}
	require.NoError(t, svc.SafePut(ctx, parentDir, s, nil))

	recBefore, err := svc.GetActiveFileRecord(ctx, "secret.txt")
	require.NoError(t, err)

	// Move to trash
	trashItem, err := svc.MoveToTrash(ctx, "secret.txt")
	require.NoError(t, err)
	assert.NotEmpty(t, trashItem.TrashID)

	// File removed from active served root
	_, err = os.Stat(filepath.Join(rootDir, "secret.txt"))
	assert.True(t, os.IsNotExist(err))

	// Active file record marked deleted
	_, err = svc.GetActiveFileRecord(ctx, "secret.txt")
	assert.Error(t, err)

	// Simulate conflicting file at original path
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "secret.txt"), []byte("newfile"), 0644))

	// Restoring must fail without overwrite
	_, err = svc.RestoreFromTrash(ctx, trashItem.TrashID, "")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrDestinationExists))

	// Remove conflicting file and restore successfully
	require.NoError(t, os.Remove(filepath.Join(rootDir, "secret.txt")))
	restoredRec, err := svc.RestoreFromTrash(ctx, trashItem.TrashID, "")
	require.NoError(t, err)
	assert.Equal(t, recBefore.FileID, restoredRec.FileID)

	restoredData, err := os.ReadFile(filepath.Join(rootDir, "secret.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("secret"), restoredData)
}

func TestAppDeletionNewFileDoesNotInheritOldShare(t *testing.T) {
	svc, rootDir, _, db := setupTestEnv(t)
	ctx := context.Background()

	parentDir := &model.Object{Path: rootDir, IsFolder: true}
	s1 := &stream.FileStream{
		Obj:    &model.Object{Name: "contract.pdf", Size: 4, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("old1")),
	}
	require.NoError(t, svc.SafePut(ctx, parentDir, s1, nil))

	share := &model.Share{
		ShareID:  "share_contract",
		Name:     "contract.pdf",
		RootPath: "/contract.pdf",
	}
	require.NoError(t, localstate.BindShareToFileRecord(ctx, db, share, 1, "contract.pdf", false))
	require.NotEmpty(t, share.FileRecordID)
	require.NoError(t, db.Create(share).Error)

	ok, err := localstate.ValidateShareBoundFileRecord(ctx, db, share, 1, "contract.pdf")
	require.NoError(t, err)
	assert.True(t, ok)

	// Move file to trash (deletion)
	_, err = svc.MoveToTrash(ctx, "contract.pdf")
	require.NoError(t, err)

	// Share access fails closed
	ok, err = localstate.ValidateShareBoundFileRecord(ctx, db, share, 1, "contract.pdf")
	require.NoError(t, err)
	assert.False(t, ok)

	// New distinct file created at the SAME path
	s2 := &stream.FileStream{
		Obj:    &model.Object{Name: "contract.pdf", Size: 4, Modified: time.Now()},
		Reader: bytes.NewReader([]byte("new2")),
	}
	require.NoError(t, svc.SafePut(ctx, parentDir, s2, nil))

	recNew, err := svc.GetActiveFileRecord(ctx, "contract.pdf")
	require.NoError(t, err)
	assert.NotEqual(t, share.FileRecordID, recNew.FileID)

	// The old share bound to old FileID MUST STILL FAIL
	ok, err = localstate.ValidateShareBoundFileRecord(ctx, db, share, 1, "contract.pdf")
	require.NoError(t, err)
	assert.False(t, ok, "old share must not be inherited by newly created file at same path")
}

func TestDeterministicReconcileAfterSwapBeforeDBCommit(t *testing.T) {
	svc, rootDir, stateDir, db := setupTestEnv(t)
	ctx := context.Background()

	// 1. Simulate Put interrupted in "swapped" stage (file in place on disk, but DB commit didn't happen)
	dstRel := "swapped_doc.txt"
	dstDiskPath := filepath.Join(rootDir, dstRel)
	require.NoError(t, os.WriteFile(dstDiskPath, []byte("recovered content"), 0644))

	putOp := &model.PendingOperation{
		OpType:       "put",
		Stage:        "swapped",
		DstPath:      dstRel,
		TempLocation: filepath.Join(stateDir, "staging", "stg_dummy"),
		Revision:     0,
	}
	require.NoError(t, svc.BeginOperation(ctx, putOp))

	// Reconcile must forward-complete the DB record!
	require.NoError(t, svc.Reconcile(ctx))

	rec, err := svc.GetActiveFileRecord(ctx, dstRel)
	require.NoError(t, err)
	assert.Equal(t, dstRel, rec.RelativePath)
	assert.Equal(t, int64(len("recovered content")), rec.Size)

	// Pending operation cleared after successful reconciliation
	var count int64
	db.Model(&model.PendingOperation{}).Where("op_id = ?", putOp.OpID).Count(&count)
	assert.Equal(t, int64(0), count)

	// 2. Simulate Move interrupted in "swapped" stage
	srcRel := "old_name.txt"
	movedRel := "moved_name.txt"
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, movedRel), []byte("moved"), 0644))

	moveOp := &model.PendingOperation{
		OpType:  "move",
		Stage:   "swapped",
		SrcPath: srcRel,
		DstPath: movedRel,
	}
	require.NoError(t, svc.BeginOperation(ctx, moveOp))

	// Initial file record was at srcRel
	recBefore := model.FileRecord{
		FileID:       "fil_move_recover",
		StorageID:    1,
		RelativePath: srcRel,
		Revision:     1,
		Size:         5,
	}
	require.NoError(t, db.Create(&recBefore).Error)

	require.NoError(t, svc.Reconcile(ctx))

	// DB record must be forward-completed to movedRel
	var recAfter model.FileRecord
	require.NoError(t, db.Where("file_id = ?", "fil_move_recover").Take(&recAfter).Error)
	assert.Equal(t, movedRel, recAfter.RelativePath)

	// 3. Indeterminate state (e.g. neither src nor dst exist) must FAIL reconcile and retain op
	badOp := &model.PendingOperation{
		OpType:  "move",
		Stage:   "swapped",
		SrcPath: "ghost_src.txt",
		DstPath: "ghost_dst.txt",
	}
	require.NoError(t, svc.BeginOperation(ctx, badOp))
	err = svc.Reconcile(ctx)
	assert.Error(t, err, "indeterminate pending operation must fail reconcile to avoid silent corruption")
}

func TestCrossRootRenameRejectsUnrelatedPaths(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)

	insidePath := filepath.Join(rootDir, "inside.txt")
	require.NoError(t, os.WriteFile(insidePath, []byte("inside"), 0644))

	externalDir := t.TempDir()
	externalPath := filepath.Join(externalDir, "external.txt")
	require.NoError(t, os.WriteFile(externalPath, []byte("external"), 0644))

	stagingDir := filepath.Join(svc.StatePath(), "staging")

	// Source outside both owned roots is rejected and left untouched.
	err := svc.CrossRootRename(externalPath, filepath.Join(stagingDir, "blob1"))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrPathEscapesRoot))
	assert.FileExists(t, externalPath)

	// Destination outside both owned roots is rejected and source is untouched.
	err = svc.CrossRootRename(insidePath, filepath.Join(externalDir, "moved.txt"))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, localstate.ErrPathEscapesRoot))
	assert.FileExists(t, insidePath)
	assert.NoFileExists(t, filepath.Join(externalDir, "moved.txt"))

	// An owned root itself is never renameable.
	err = svc.CrossRootRename(rootDir, filepath.Join(stagingDir, "rootblob"))
	assert.Error(t, err)
}

func TestCrossRootRenameSymlinkParentPivotRejected(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)

	externalDir := t.TempDir()
	victim := filepath.Join(externalDir, "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("victim"), 0644))

	pivot := filepath.Join(rootDir, "pivot")
	if err := os.Symlink(externalDir, pivot); err != nil {
		t.Skip("symlinks unavailable in current environment")
	}

	// Symlinked source parent must not redirect the move out of the served root.
	err := svc.CrossRootRename(filepath.Join(pivot, "victim.txt"), filepath.Join(svc.StatePath(), "staging", "stolen"))
	assert.Error(t, err)
	assert.FileExists(t, victim)
	assert.NoFileExists(t, filepath.Join(svc.StatePath(), "staging", "stolen"))

	// Symlinked destination parent must not accept content from the state root.
	secret := filepath.Join(svc.StatePath(), "staging", "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0644))
	dstPivot := filepath.Join(svc.StatePath(), "trash", "outpivot")
	if err := os.Symlink(externalDir, dstPivot); err == nil {
		err = svc.CrossRootRename(secret, filepath.Join(dstPivot, "leaked.txt"))
		assert.Error(t, err)
		assert.FileExists(t, secret)
		assert.NoFileExists(t, filepath.Join(externalDir, "leaked.txt"))
	}
}

func TestCrossRootRenameLifecycleNoReplaceAndReplace(t *testing.T) {
	svc, rootDir, _, _ := setupTestEnv(t)

	src := filepath.Join(rootDir, "payload.txt")
	require.NoError(t, os.WriteFile(src, []byte("payload"), 0644))
	blob := filepath.Join(svc.StatePath(), "staging", "blob_payload")

	// Ordinary root -> state move (parent is the owned root itself).
	require.NoError(t, svc.CrossRootRename(src, blob))
	_, err := os.Stat(src)
	assert.True(t, os.IsNotExist(err))
	data, err := os.ReadFile(blob)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), data)

	// No-replace must refuse to clobber an existing destination.
	decoy := filepath.Join(rootDir, "decoy.txt")
	require.NoError(t, os.WriteFile(decoy, []byte("decoy"), 0644))
	err = svc.CrossRootRename(decoy, blob)
	assert.Error(t, err)
	data, err = os.ReadFile(blob)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), data)
	assert.FileExists(t, decoy)

	// Explicit replace moves state -> root over the existing file.
	require.NoError(t, svc.CrossRootRenameReplace(blob, decoy))
	data, err = os.ReadFile(decoy)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), data)
	_, err = os.Stat(blob)
	assert.True(t, os.IsNotExist(err))
}

type failingReader struct {
	data   []byte
	pos    int
	failAt int
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.failAt {
		return 0, errors.New("simulated network drop during upload")
	}
	n = copy(p, r.data[r.pos:r.failAt])
	r.pos += n
	return n, nil
}
