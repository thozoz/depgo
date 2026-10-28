package localstate

import (
	"context"
	"fmt"
	"os"
	stdpath "path"
	"path/filepath"
	"time"

	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	cp "github.com/otiai10/copy"
	"gorm.io/gorm"
)

// SafePut performs stage-write-sync-close, snapshotting any existing destination before atomic replacement.
func (s *Service) SafePut(ctx context.Context, dstDir model.Obj, stream model.FileStreamer, up driver.UpdateProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ValidateNameComponent(stream.GetName()); err != nil {
		return err
	}

	parentRel, err := s.DiskToRelative(dstDir.GetPath())
	if err != nil {
		return err
	}

	dstRel := stream.GetName()
	if parentRel != "" {
		dstRel = stdpath.Join(parentRel, stream.GetName())
	}

	dstDiskPath, err := s.RelToDiskPath(dstRel)
	if err != nil {
		return err
	}
	if err := s.CheckSymlink(parentRel); err != nil {
		return err
	}

	// 1. Stage: write into temporary file inside private staging directory on the same filesystem
	tempFileName := fmt.Sprintf("stg_%s_%s", random.String(12), stream.GetName())
	tempDiskPath := filepath.Join(s.stagingPath, tempFileName)

	out, err := os.OpenFile(tempDiskPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create staging file: %w", err)
	}

	var copyErr error
	defer func() {
		_ = out.Close()
		if copyErr != nil {
			_ = os.Remove(tempDiskPath)
		}
	}()

	progress := up
	if progress == nil {
		progress = func(float64) {}
	}
	copyErr = utils.CopyWithCtx(ctx, out, stream, stream.GetSize(), progress)
	if copyErr != nil {
		return fmt.Errorf("stream copy to staging: %w", copyErr)
	}

	if err := out.Sync(); err != nil {
		copyErr = err
		return fmt.Errorf("sync staging file: %w", err)
	}
	if err := out.Close(); err != nil {
		copyErr = err
		return fmt.Errorf("close staging file: %w", err)
	}

	// 2. Check existing destination: snapshot to version history before overwrite
	var existingRec *model.FileRecord
	var existingRevision int64
	if s.db != nil {
		existingRec, _ = s.GetActiveFileRecord(ctx, dstRel)
	}

	if fi, err := s.osRoot.Stat(dstRel); err == nil {
		if fi.IsDir() {
			copyErr = fmt.Errorf("cannot overwrite existing directory with file: %s", dstRel)
			return copyErr
		}
		if existingRec == nil && s.db != nil {
			existingRec, err = s.GetOrCreateFileRecord(ctx, dstRel, false, fi.Size(), fi.ModTime())
			if err != nil {
				copyErr = err
				return fmt.Errorf("failed creating file record before snapshot: %w", err)
			}
		}
		if existingRec != nil {
			existingRevision = existingRec.Revision
			if _, snapErr := s.SnapshotFile(ctx, nil, existingRec, dstRel); snapErr != nil {
				copyErr = snapErr
				// Fail closed before mutation if snapshot fails
				return fmt.Errorf("snapshot before overwrite failed: %w", snapErr)
			}
		}
	} else if !errorsIsNotExist(err) {
		copyErr = err
		return fmt.Errorf("stat destination: %w", err)
	}

	// 3. Begin pending operation journal BEFORE disk mutation
	op := &model.PendingOperation{
		OpType:       "put",
		Stage:        "staged",
		SrcPath:      tempDiskPath,
		DstPath:      dstRel,
		TempLocation: tempDiskPath,
		Revision:     existingRevision,
		IsDir:        false,
	}
	if existingRec != nil {
		op.FileRecordID = existingRec.FileID
	}
	if err := s.BeginOperation(ctx, op); err != nil {
		copyErr = err
		return fmt.Errorf("failed recording journal before swap: %w", err)
	}

	// 4. Atomic replace using cross-root rename
	if err := s.CrossRootRenameReplace(tempDiskPath, dstDiskPath); err != nil {
		_ = os.Remove(tempDiskPath)
		_ = s.EndOperation(ctx, op.OpID, false)
		return fmt.Errorf("atomic replace failed: %w", err)
	}

	_ = s.UpdateOperationStage(ctx, op.OpID, "swapped")

	// 5. Preserve timestamps
	modTime := stream.ModTime()
	if modTime.IsZero() {
		modTime = time.Now()
	}
	_ = s.osRoot.Chtimes(dstRel, modTime, modTime)

	// 6. Update FileRecord in DB
	if s.db != nil {
		fi, statErr := s.osRoot.Stat(dstRel)
		size := stream.GetSize()
		if statErr == nil {
			size = fi.Size()
		}
		_, touchErr := s.TouchFileRecord(ctx, nil, dstRel, size, modTime)
		if touchErr != nil {
			// Leave operation in swapped state for recovery and return error
			return fmt.Errorf("destination written but file record update failed: %w", touchErr)
		}
	}

	if err := s.EndOperation(ctx, op.OpID, true); err != nil {
		return fmt.Errorf("destination written but journal completion failed: %w", err)
	}

	return nil
}

// SafeMove moves a file or folder, enforcing no silent overwrite, updating descendant mappings and shares.
func (s *Service) SafeMove(ctx context.Context, srcObj, dstDir model.Obj, storageMountPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	srcRel, err := s.DiskToRelative(srcObj.GetPath())
	if err != nil {
		return err
	}
	dstDirRel, err := s.DiskToRelative(dstDir.GetPath())
	if err != nil {
		return err
	}

	dstRel := srcObj.GetName()
	if dstDirRel != "" {
		dstRel = stdpath.Join(dstDirRel, srcObj.GetName())
	}

	// Same path: no-op
	if srcRel == dstRel {
		return nil
	}

	srcDiskPath, err := s.RelToDiskPath(srcRel)
	if err != nil {
		return err
	}
	dstDiskPath, err := s.RelToDiskPath(dstRel)
	if err != nil {
		return err
	}

	if utils.IsSubPath(srcDiskPath, dstDiskPath) {
		return fmt.Errorf("destination folder is a subfolder of source folder")
	}

	// Enforce no silent destination overwrite
	if _, err := s.osRoot.Stat(dstRel); err == nil {
		return fmt.Errorf("%w: [%s]", ErrDestinationExists, dstRel)
	} else if !errorsIsNotExist(err) {
		return err
	}

	if err := s.CheckSymlink(srcRel); err != nil {
		return err
	}
	if err := s.CheckSymlink(dstDirRel); err != nil {
		return err
	}

	op := &model.PendingOperation{
		OpType:    "move",
		Stage:     "staged",
		SrcPath:   srcRel,
		DstPath:   dstRel,
		MountPath: storageMountPath,
	}
	if err := s.BeginOperation(ctx, op); err != nil {
		return fmt.Errorf("journal record failed before move: %w", err)
	}

	// Anchored intra-root move
	if err := s.osRoot.Rename(srcRel, dstRel); err != nil {
		_ = s.EndOperation(ctx, op.OpID, false)
		return fmt.Errorf("disk rename %s -> %s: %w", srcRel, dstRel, err)
	}

	_ = s.UpdateOperationStage(ctx, op.OpID, "swapped")

	// Update DB mappings & shares
	if s.db != nil {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return s.UpdateFileRecordPath(ctx, tx, srcRel, dstRel, storageMountPath)
		})
		if err != nil {
			// Try to roll back disk move
			if rbErr := s.osRoot.Rename(dstRel, srcRel); rbErr == nil {
				_ = s.EndOperation(ctx, op.OpID, false)
				return fmt.Errorf("db update failed for move, rolled back: %w", err)
			}
			// Rollback failed; preserve swapped stage for startup recovery
			return fmt.Errorf("db update failed for move, rollback failed: %w", err)
		}
	}

	_ = s.EndOperation(ctx, op.OpID, true)
	return nil
}

// SafeRename renames a file or directory, treating same-name as a no-op and preventing silent overwrites.
func (s *Service) SafeRename(ctx context.Context, srcObj model.Obj, newName string, storageMountPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ValidateNameComponent(newName); err != nil {
		return err
	}

	// Same-name rename is a no-op
	if srcObj.GetName() == newName {
		return nil
	}

	srcRel, err := s.DiskToRelative(srcObj.GetPath())
	if err != nil {
		return err
	}
	parentRel := stdpath.Dir(srcRel)
	if parentRel == "." {
		parentRel = ""
	}

	dstRel := newName
	if parentRel != "" {
		dstRel = stdpath.Join(parentRel, newName)
	}

	if srcRel == dstRel {
		return nil
	}

	// Enforce no silent destination overwrite
	if _, err := s.osRoot.Stat(dstRel); err == nil {
		return fmt.Errorf("%w: [%s]", ErrDestinationExists, dstRel)
	} else if !errorsIsNotExist(err) {
		return err
	}

	if err := s.CheckSymlink(srcRel); err != nil {
		return err
	}

	op := &model.PendingOperation{
		OpType:    "rename",
		Stage:     "staged",
		SrcPath:   srcRel,
		DstPath:   dstRel,
		MountPath: storageMountPath,
	}
	if err := s.BeginOperation(ctx, op); err != nil {
		return fmt.Errorf("journal record failed before rename: %w", err)
	}

	if err := s.osRoot.Rename(srcRel, dstRel); err != nil {
		_ = s.EndOperation(ctx, op.OpID, false)
		return fmt.Errorf("disk rename: %w", err)
	}

	_ = s.UpdateOperationStage(ctx, op.OpID, "swapped")

	if s.db != nil {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return s.UpdateFileRecordPath(ctx, tx, srcRel, dstRel, storageMountPath)
		})
		if err != nil {
			if rbErr := s.osRoot.Rename(dstRel, srcRel); rbErr == nil {
				_ = s.EndOperation(ctx, op.OpID, false)
				return fmt.Errorf("db update failed for rename, rolled back: %w", err)
			}
			return fmt.Errorf("db update failed for rename, rollback failed: %w", err)
		}
	}

	_ = s.EndOperation(ctx, op.OpID, true)
	return nil
}

// SafeCopy copies a file or directory, creating fresh FileRecords and preserving timestamps without symlink escapes.
func (s *Service) SafeCopy(ctx context.Context, srcObj, dstDir model.Obj) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	srcRel, err := s.DiskToRelative(srcObj.GetPath())
	if err != nil {
		return err
	}
	dstDirRel, err := s.DiskToRelative(dstDir.GetPath())
	if err != nil {
		return err
	}

	dstRel := srcObj.GetName()
	if dstDirRel != "" {
		dstRel = stdpath.Join(dstDirRel, srcObj.GetName())
	}

	srcDiskPath, err := s.RelToDiskPath(srcRel)
	if err != nil {
		return err
	}
	dstDiskPath, err := s.RelToDiskPath(dstRel)
	if err != nil {
		return err
	}

	if utils.IsSubPath(srcDiskPath, dstDiskPath) {
		return fmt.Errorf("destination folder is a subfolder of source folder")
	}

	// Enforce no silent destination overwrite
	if _, err := s.osRoot.Stat(dstRel); err == nil {
		return fmt.Errorf("%w: [%s]", ErrDestinationExists, dstRel)
	} else if !errorsIsNotExist(err) {
		return err
	}

	if err := s.CheckSymlink(srcRel); err != nil {
		return err
	}
	if err := s.CheckSymlink(dstDirRel); err != nil {
		return err
	}

	if err := cp.Copy(srcDiskPath, dstDiskPath, cp.Options{
		Sync:          true,
		PreserveTimes: true,
		OnSymlink: func(src string) cp.SymlinkAction {
			return cp.Skip
		},
	}); err != nil {
		return fmt.Errorf("secure copy %s -> %s: %w", srcDiskPath, dstDiskPath, err)
	}

	if s.db != nil {
		fi, statErr := s.osRoot.Stat(dstRel)
		if statErr == nil {
			_, _ = s.GetOrCreateFileRecord(ctx, dstRel, fi.IsDir(), fi.Size(), fi.ModTime())
		}
	}

	return nil
}

// SafeMakeDir creates a directory within root, validating against symlink escapes.
func (s *Service) SafeMakeDir(ctx context.Context, parentDir model.Obj, dirName string, perm os.FileMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ValidateNameComponent(dirName); err != nil {
		return err
	}

	parentRel, err := s.DiskToRelative(parentDir.GetPath())
	if err != nil {
		return err
	}

	dstRel := dirName
	if parentRel != "" {
		dstRel = stdpath.Join(parentRel, dirName)
	}

	if err := s.CheckSymlink(parentRel); err != nil {
		return err
	}

	if err := s.osRoot.MkdirAll(dstRel, perm); err != nil {
		return err
	}

	if s.db != nil {
		fi, err := s.osRoot.Stat(dstRel)
		if err == nil {
			_, _ = s.GetOrCreateFileRecord(ctx, dstRel, true, 0, fi.ModTime())
		}
	}

	return nil
}
