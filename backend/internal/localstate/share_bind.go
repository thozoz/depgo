package localstate

import (
	"context"
	"errors"
	stdpath "path"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"gorm.io/gorm"
)

func cleanCanonicalRel(p string) string {
	cleaned := stdpath.Clean("/" + strings.ReplaceAll(p, "\\", "/"))
	return strings.TrimPrefix(cleaned, "/")
}

// BindShareToFileRecord binds a share to a stable FileRecord for the given storage and relative path.
// If no active FileRecord exists, a lazy one is created and bound.
func BindShareToFileRecord(ctx context.Context, db *gorm.DB, share *model.Share, storageID uint, relPath string, isDir bool) error {
	if db == nil {
		return nil
	}
	normPath := cleanCanonicalRel(relPath)

	var rec model.FileRecord
	err := db.WithContext(ctx).
		Where("storage_id = ? AND relative_path = ? AND is_deleted = ?", storageID, normPath, false).
		Take(&rec).Error

	if err == nil {
		share.FileRecordID = rec.FileID
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create lazy record
	rec = model.FileRecord{
		FileID:       "fil_" + random.String(16),
		StorageID:    storageID,
		RelativePath: normPath,
		Revision:     1,
		IsDir:        isDir,
		IsDeleted:    false,
		ModTime:      time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.WithContext(ctx).Create(&rec).Error; err != nil {
		return err
	}
	share.FileRecordID = rec.FileID
	return nil
}

// ValidateShareBoundFileRecord verifies that if a share is bound to a FileRecord,
// the record exists and is not deleted or trashed.
// For existing unbound shares on a local storage, it binds them if active, or returns false (fail-closed).
func ValidateShareBoundFileRecord(ctx context.Context, db *gorm.DB, share *model.Share, storageID uint, relPath string) (bool, error) {
	if db == nil {
		return true, nil
	}

	if share.FileRecordID != "" {
		var rec model.FileRecord
		err := db.WithContext(ctx).
			Where("file_id = ?", share.FileRecordID).
			Take(&rec).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil // bound record missing -> fail closed
			}
			return false, err
		}
		if rec.IsDeleted {
			return false, nil // trashed or deleted -> fail closed
		}
		return true, nil
	}

	// Unbound share on a local storage: check if active file record exists
	normPath := cleanCanonicalRel(relPath)
	var rec model.FileRecord
	err := db.WithContext(ctx).
		Where("storage_id = ? AND relative_path = ? AND is_deleted = ?", storageID, normPath, false).
		Take(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Unbound and file does not exist -> fail closed
			return false, nil
		}
		return false, err
	}

	// Bind safely for future requests
	share.FileRecordID = rec.FileID
	_ = db.WithContext(ctx).Model(&model.Share{}).
		Where("id = ?", share.ID).
		Update("file_record_id", rec.FileID).Error

	return true, nil
}
