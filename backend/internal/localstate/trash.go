package localstate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"gorm.io/gorm"
)

func errorsIsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || os.IsNotExist(err)
}

// MoveToTrash safely moves an active file or directory to private trash storage
// and records it in the TrashItem database table.
func (s *Service) MoveToTrash(ctx context.Context, storageRelPath string) (*model.TrashItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normPath, err := s.CanonicalRelPath(storageRelPath)
	if err != nil {
		return nil, err
	}
	if normPath == "" {
		return nil, fmt.Errorf("deleting storage root is not permitted")
	}

	diskPath, err := s.RelToDiskPath(normPath)
	if err != nil {
		return nil, err
	}
	if err := s.CheckSymlink(normPath); err != nil {
		return nil, err
	}

	fi, err := s.osRoot.Stat(normPath)
	if err != nil {
		return nil, fmt.Errorf("stat target for trash [%s]: %w", normPath, err)
	}

	trashID := "tsh_" + random.String(24)
	blobPath := filepath.Join(s.trashPath, trashID)

	rec, _ := s.GetActiveFileRecord(ctx, normPath)
	fileID := ""
	if rec != nil {
		fileID = rec.FileID
	} else {
		fileID = "fil_" + random.String(16)
	}

	// 1. Begin durable journal operation BEFORE mutation
	op := &model.PendingOperation{
		OpType:       "remove",
		Stage:        "staged",
		SrcPath:      normPath,
		DstPath:      blobPath,
		TempLocation: blobPath,
		FileRecordID: fileID,
		TrashID:      trashID,
		IsDir:        fi.IsDir(),
		OriginalSize: fi.Size(),
		Revision:     0,
	}
	if rec != nil {
		op.Revision = rec.Revision
	}
	if err := s.BeginOperation(ctx, op); err != nil {
		return nil, fmt.Errorf("journal record failed before trash move: %w", err)
	}

	// 2. Move disk contents to unique trash blob
	if err := s.CrossRootRename(diskPath, blobPath); err != nil {
		_ = s.EndOperation(ctx, op.OpID, false)
		return nil, fmt.Errorf("move to trash blob: %w", err)
	}

	_ = s.UpdateOperationStage(ctx, op.OpID, "swapped")

	item := &model.TrashItem{
		TrashID:      trashID,
		StorageID:    s.storageID,
		FileID:       fileID,
		OriginalPath: "/" + normPath,
		BlobLocation: blobPath,
		Size:         fi.Size(),
		IsDir:        fi.IsDir(),
		DeletedAt:    time.Now(),
		CreatedAt:    time.Now(),
	}

	// 3. Commit DB records
	if s.db != nil {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(item).Error; err != nil {
				return err
			}
			return s.MarkFileRecordDeleted(ctx, tx, normPath)
		})
		if err != nil {
			// Roll back disk move if DB write failed
			_ = s.CrossRootRename(blobPath, diskPath)
			_ = s.EndOperation(ctx, op.OpID, false)
			return nil, fmt.Errorf("record trash transaction failed, rolled back: %w", err)
		}
	}

	_ = s.EndOperation(ctx, op.OpID, true)
	return item, nil
}

// RestoreFromTrash restores a previously trashed item to its original path or custom target.
// Fails without overwrite if destination already exists.
func (s *Service) RestoreFromTrash(ctx context.Context, trashID string, customTargetRelPath string) (*model.FileRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil, fmt.Errorf("database required for trash restore")
	}

	var item model.TrashItem
	if err := s.db.WithContext(ctx).Where("storage_id = ? AND trash_id = ?", s.storageID, trashID).Take(&item).Error; err != nil {
		return nil, fmt.Errorf("trash item not found: %w", err)
	}

	targetPath := item.OriginalPath
	if customTargetRelPath != "" {
		targetPath = customTargetRelPath
	}
	normTarget, err := s.CanonicalRelPath(targetPath)
	if err != nil {
		return nil, err
	}

	dstDiskPath, err := s.RelToDiskPath(normTarget)
	if err != nil {
		return nil, err
	}

	// Check if destination already exists; fail without overwrite
	if _, err := s.osRoot.Stat(normTarget); err == nil {
		return nil, fmt.Errorf("%w: cannot restore over existing path [%s]", ErrDestinationExists, normTarget)
	} else if !errorsIsNotExist(err) {
		return nil, fmt.Errorf("check destination for restore: %w", err)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(dstDiskPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, fmt.Errorf("create restore parent dir: %w", err)
	}

	// Journal before mutation
	op := &model.PendingOperation{
		OpType:       "restore",
		Stage:        "staged",
		SrcPath:      item.BlobLocation,
		DstPath:      normTarget,
		TempLocation: item.BlobLocation,
		FileRecordID: item.FileID,
		TrashID:      item.TrashID,
		IsDir:        item.IsDir,
	}
	if err := s.BeginOperation(ctx, op); err != nil {
		return nil, fmt.Errorf("journal record failed before restore: %w", err)
	}

	// Move blob back from trash to destination.
	//
	// No-replace, not plain move: the existence check above only observed one
	// instant, and a plain rename would silently destroy whatever was created at
	// that name in between. Restoring a file must never cost someone else's data.
	if err := s.CrossRootRenameNoReplace(item.BlobLocation, dstDiskPath); err != nil {
		_ = s.EndOperation(ctx, op.OpID, false)
		if errors.Is(err, os.ErrExist) || errors.Is(err, syscall.EEXIST) {
			return nil, fmt.Errorf("%w: cannot restore over existing path [%s]", ErrDestinationExists, normTarget)
		}
		return nil, fmt.Errorf("restore blob: %w", err)
	}

	_ = s.UpdateOperationStage(ctx, op.OpID, "swapped")

	var restoredRec *model.FileRecord
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create or reactivate FileRecord
		var rec model.FileRecord
		err := tx.Where("storage_id = ? AND file_id = ?", s.storageID, item.FileID).Take(&rec).Error
		if err == nil {
			rec.RelativePath = normTarget
			rec.IsDeleted = false
			rec.DeletedAt = nil
			rec.Revision++
			rec.UpdatedAt = time.Now()
			if err := tx.Save(&rec).Error; err != nil {
				return err
			}
			restoredRec = &rec
		} else {
			newRec := model.FileRecord{
				FileID:       item.FileID,
				StorageID:    s.storageID,
				RelativePath: normTarget,
				Revision:     1,
				Size:         item.Size,
				ModTime:      time.Now(),
				IsDir:        item.IsDir,
				IsDeleted:    false,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := tx.Create(&newRec).Error; err != nil {
				return err
			}
			restoredRec = &newRec
		}

		// Delete the trash item record
		return tx.Delete(&item).Error
	})

	if err != nil {
		// Roll back disk move if DB transaction failed
		_ = s.CrossRootRename(dstDiskPath, item.BlobLocation)
		_ = s.EndOperation(ctx, op.OpID, false)
		return nil, fmt.Errorf("restore transaction failed, rolled back: %w", err)
	}

	_ = s.EndOperation(ctx, op.OpID, true)
	return restoredRec, nil
}

// PurgeTrash permanently removes a trashed item from disk and DB.
func (s *Service) PurgeTrash(ctx context.Context, trashID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return fmt.Errorf("database required for purge")
	}

	var item model.TrashItem
	if err := s.db.WithContext(ctx).Where("storage_id = ? AND trash_id = ?", s.storageID, trashID).Take(&item).Error; err != nil {
		return fmt.Errorf("trash item not found: %w", err)
	}

	// Safety check: ensure blob is strictly inside private trashPath
	rel, err := filepath.Rel(s.trashPath, item.BlobLocation)
	if err != nil || isEscapingRel(rel) {
		return fmt.Errorf("security violation: trash blob [%s] escapes trash directory", item.BlobLocation)
	}

	_ = os.RemoveAll(item.BlobLocation)

	return s.db.WithContext(ctx).Delete(&item).Error
}

// ListTrash returns all trash items for this storage.
func (s *Service) ListTrash(ctx context.Context) ([]model.TrashItem, error) {
	if s.db == nil {
		return nil, nil
	}
	var items []model.TrashItem
	err := s.db.WithContext(ctx).
		Where("storage_id = ?", s.storageID).
		Order("deleted_at desc").
		Find(&items).Error
	return items, err
}

// GetTrashItem retrieves a single trash item by its trash ID.
func (s *Service) GetTrashItem(ctx context.Context, trashID string) (*model.TrashItem, error) {
	if s.db == nil {
		return nil, ErrNotFound
	}
	var item model.TrashItem
	err := s.db.WithContext(ctx).
		Where("storage_id = ? AND trash_id = ?", s.storageID, trashID).
		Take(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}
