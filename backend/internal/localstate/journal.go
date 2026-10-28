package localstate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BeginOperation records an operation in the durable pending operation journal.
func (s *Service) BeginOperation(ctx context.Context, op *model.PendingOperation) error {
	if s.db == nil {
		if op.OpID == "" {
			op.OpID = "op_" + random.String(24)
		}
		op.StorageID = s.storageID
		op.CreatedAt = time.Now()
		op.UpdatedAt = time.Now()
		return nil
	}

	if op.OpID == "" {
		op.OpID = "op_" + random.String(24)
	}
	op.StorageID = s.storageID
	op.CreatedAt = time.Now()
	op.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Create(op).Error; err != nil {
		return fmt.Errorf("record pending operation: %w", err)
	}
	return nil
}

// UpdateOperationStage updates the progress stage of an operation.
func (s *Service) UpdateOperationStage(ctx context.Context, opID, stage string) error {
	if s.db == nil {
		return nil
	}
	return s.db.WithContext(ctx).Model(&model.PendingOperation{}).
		Where("storage_id = ? AND op_id = ?", s.storageID, opID).
		Updates(map[string]interface{}{
			"stage":      stage,
			"updated_at": time.Now(),
		}).Error
}

// EndOperation marks an operation as committed or failed, cleaning up the journal entry on success.
func (s *Service) EndOperation(ctx context.Context, opID string, success bool) error {
	if s.db == nil {
		return nil
	}
	if success {
		return s.db.WithContext(ctx).
			Where("storage_id = ? AND op_id = ?", s.storageID, opID).
			Delete(&model.PendingOperation{}).Error
	}
	return s.db.WithContext(ctx).Model(&model.PendingOperation{}).
		Where("storage_id = ? AND op_id = ?", s.storageID, opID).
		Updates(map[string]interface{}{
			"stage":      "failed",
			"updated_at": time.Now(),
		}).Error
}

// Reconcile scans the pending operation journal on startup and deterministically
// forward-completes or rolls back interrupted operations. Unresolvable states fail startup.
func (s *Service) Reconcile(ctx context.Context) error {
	if s.db == nil {
		return nil
	}

	var pending []model.PendingOperation
	err := s.db.WithContext(ctx).
		Where("storage_id = ? AND stage IN (?)", s.storageID, []string{"staged", "swapped", "in_progress"}).
		Find(&pending).Error
	if err != nil {
		return fmt.Errorf("query pending operations for reconcile: %w", err)
	}

	for _, op := range pending {
		log.Infof("[localstate reconcile] inspecting pending op %s type=%s stage=%s src=%s dst=%s",
			op.OpID, op.OpType, op.Stage, op.SrcPath, op.DstPath)

		switch op.OpType {
		case "put":
			if op.Stage == "staged" {
				// Disk destination not yet swapped: remove staging file and remove journal entry
				if op.TempLocation != "" {
					rel, err := filepath.Rel(s.stagingPath, op.TempLocation)
					if err == nil && !isEscapingRel(rel) {
						_ = os.Remove(op.TempLocation)
					}
				}
				_ = s.EndOperation(ctx, op.OpID, true)
			} else if op.Stage == "swapped" {
				// Atomic rename to destination succeeded on disk, but DB commit was interrupted
				fi, statErr := s.osRoot.Stat(op.DstPath)
				if statErr == nil {
					// Forward-complete: record in DB
					_, touchErr := s.TouchFileRecord(ctx, nil, op.DstPath, fi.Size(), fi.ModTime())
					if touchErr != nil {
						return fmt.Errorf("reconcile put op %s failed updating file record: %w", op.OpID, touchErr)
					}
					_ = s.EndOperation(ctx, op.OpID, true)
				} else {
					return fmt.Errorf("reconcile put op %s: swapped destination [%s] missing on disk: %w", op.OpID, op.DstPath, statErr)
				}
			}

		case "move", "rename":
			if op.Stage == "staged" {
				// Disk not yet renamed
				_ = s.EndOperation(ctx, op.OpID, true)
			} else if op.Stage == "swapped" || op.Stage == "in_progress" {
				_, dstErr := s.osRoot.Stat(op.DstPath)
				_, srcErr := s.osRoot.Stat(op.SrcPath)

				if dstErr == nil && errors.Is(srcErr, os.ErrNotExist) {
					// Disk move succeeded: forward-complete DB records
					err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
						return s.UpdateFileRecordPath(ctx, tx, op.SrcPath, op.DstPath, op.MountPath)
					})
					if err != nil {
						return fmt.Errorf("reconcile %s op %s failed updating DB: %w", op.OpType, op.OpID, err)
					}
					_ = s.EndOperation(ctx, op.OpID, true)
				} else if srcErr == nil && errors.Is(dstErr, os.ErrNotExist) {
					// Disk move did not happen
					_ = s.EndOperation(ctx, op.OpID, true)
				} else {
					return fmt.Errorf("reconcile %s op %s indeterminate state: srcErr=%v, dstErr=%v", op.OpType, op.OpID, srcErr, dstErr)
				}
			}

		case "remove":
			if op.Stage == "staged" {
				_ = s.EndOperation(ctx, op.OpID, true)
			} else if op.Stage == "swapped" {
				// File was moved to trash blob on disk
				blobPath := op.TempLocation
				_, blobErr := os.Stat(blobPath)
				_, srcErr := s.osRoot.Stat(op.SrcPath)

				if blobErr == nil && errors.Is(srcErr, os.ErrNotExist) {
					// Forward-complete DB: record TrashItem and mark FileRecord deleted
					fi, _ := os.Stat(blobPath)
					size := int64(0)
					if fi != nil {
						size = fi.Size()
					}
					item := model.TrashItem{
						TrashID:      op.TrashID,
						StorageID:    s.storageID,
						FileID:       op.FileRecordID,
						OriginalPath: op.SrcPath,
						BlobLocation: blobPath,
						Size:         size,
						IsDir:        op.IsDir,
						DeletedAt:    op.CreatedAt,
						CreatedAt:    time.Now(),
					}
					err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
						if err := tx.Create(&item).Error; err != nil {
							return err
						}
						return s.MarkFileRecordDeleted(ctx, tx, op.SrcPath)
					})
					if err != nil {
						return fmt.Errorf("reconcile remove op %s failed updating DB: %w", op.OpID, err)
					}
					_ = s.EndOperation(ctx, op.OpID, true)
				} else if srcErr == nil && errors.Is(blobErr, os.ErrNotExist) {
					_ = s.EndOperation(ctx, op.OpID, true)
				} else {
					return fmt.Errorf("reconcile remove op %s indeterminate state: srcErr=%v, blobErr=%v", op.OpID, srcErr, blobErr)
				}
			}

		case "restore":
			if op.Stage == "staged" {
				_ = s.EndOperation(ctx, op.OpID, true)
			} else if op.Stage == "swapped" {
				// Restored to destination on disk, blob removed from trash
				_, dstErr := s.osRoot.Stat(op.DstPath)
				_, blobErr := os.Stat(op.TempLocation)

				if dstErr == nil && errors.Is(blobErr, os.ErrNotExist) {
					// Forward-complete DB
					err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
						var rec model.FileRecord
						if err := tx.Where("storage_id = ? AND file_id = ?", s.storageID, op.FileRecordID).Take(&rec).Error; err == nil {
							rec.RelativePath = op.DstPath
							rec.IsDeleted = false
							rec.DeletedAt = nil
							rec.Revision++
							rec.UpdatedAt = time.Now()
							if err := tx.Save(&rec).Error; err != nil {
								return err
							}
						}
						return tx.Where("storage_id = ? AND trash_id = ?", s.storageID, op.TrashID).Delete(&model.TrashItem{}).Error
					})
					if err != nil {
						return fmt.Errorf("reconcile restore op %s failed updating DB: %w", op.OpID, err)
					}
					_ = s.EndOperation(ctx, op.OpID, true)
				} else {
					return fmt.Errorf("reconcile restore op %s indeterminate state: dstErr=%v, blobErr=%v", op.OpID, dstErr, blobErr)
				}
			}

		default:
			return fmt.Errorf("unknown pending operation type: %s", op.OpType)
		}
	}

	return nil
}

// StagingPath returns the path to the temporary staging directory.
func (s *Service) StagingPath() string {
	return s.stagingPath
}
