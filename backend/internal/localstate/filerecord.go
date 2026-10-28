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

// GetOrCreateFileRecord retrieves the active FileRecord for a canonical path or creates a fresh one.
func (s *Service) GetOrCreateFileRecord(ctx context.Context, relPath string, isDir bool, size int64, modTime time.Time) (*model.FileRecord, error) {
	normPath, err := s.CanonicalRelPath(relPath)
	if err != nil {
		return nil, err
	}
	if s.db == nil {
		return &model.FileRecord{
			FileID:       "fil_" + random.String(16),
			StorageID:    s.storageID,
			RelativePath: normPath,
			Revision:     1,
			Size:         size,
			ModTime:      modTime,
			IsDir:        isDir,
		}, nil
	}

	var rec model.FileRecord
	err = s.db.WithContext(ctx).
		Where("storage_id = ? AND relative_path = ? AND is_deleted = ?", s.storageID, normPath, false).
		Take(&rec).Error

	if err == nil {
		return &rec, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create a new lazy FileRecord
	rec = model.FileRecord{
		FileID:       "fil_" + random.String(16),
		StorageID:    s.storageID,
		RelativePath: normPath,
		Revision:     1,
		Size:         size,
		ModTime:      modTime,
		IsDir:        isDir,
		IsDeleted:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// GetActiveFileRecord retrieves the active (non-deleted) FileRecord for a canonical path.
func (s *Service) GetActiveFileRecord(ctx context.Context, relPath string) (*model.FileRecord, error) {
	normPath, err := s.CanonicalRelPath(relPath)
	if err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, ErrNotFound
	}
	var rec model.FileRecord
	err = s.db.WithContext(ctx).
		Where("storage_id = ? AND relative_path = ? AND is_deleted = ?", s.storageID, normPath, false).
		Take(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

// GetCurrentRecord exposes the active FileRecord for a canonical path (used by API handlers).
func (s *Service) GetCurrentRecord(ctx context.Context, relPath string) (*model.FileRecord, error) {
	return s.GetActiveFileRecord(ctx, relPath)
}

// GetFileRecordByID retrieves a FileRecord by its unique FileID.
func (s *Service) GetFileRecordByID(ctx context.Context, fileID string) (*model.FileRecord, error) {
	if s.db == nil {
		return nil, ErrNotFound
	}
	var rec model.FileRecord
	err := s.db.WithContext(ctx).
		Where("storage_id = ? AND file_id = ?", s.storageID, fileID).
		Take(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

// MarkFileRecordDeleted marks the file record at relPath as deleted.
// A subsequent file created at the same path will receive a distinct new FileID.
func (s *Service) MarkFileRecordDeleted(ctx context.Context, tx *gorm.DB, relPath string) error {
	normPath, err := s.CanonicalRelPath(relPath)
	if err != nil {
		return err
	}
	database := s.db
	if tx != nil {
		database = tx
	}
	if database == nil {
		return nil
	}

	now := time.Now()
	prefix := normPath + "/"
	if normPath == "" {
		prefix = ""
	}

	var errDB error
	if prefix == "" {
		errDB = database.WithContext(ctx).Model(&model.FileRecord{}).
			Where("storage_id = ? AND is_deleted = ?", s.storageID, false).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"deleted_at": now,
			}).Error
	} else {
		errDB = database.WithContext(ctx).Model(&model.FileRecord{}).
			Where("storage_id = ? AND (relative_path = ? OR relative_path LIKE ?) AND is_deleted = ?", s.storageID, normPath, prefix+"%", false).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"deleted_at": now,
			}).Error
	}
	return errDB
}

// UpdateFileRecordPath updates the path of a record and all descendants within a transaction,
// also updating any bound Shares so share links remain valid across renames/moves.
func (s *Service) UpdateFileRecordPath(ctx context.Context, tx *gorm.DB, oldRelPath, newRelPath string, storageMountPath string) error {
	oldNorm, err := s.CanonicalRelPath(oldRelPath)
	if err != nil {
		return err
	}
	newNorm, err := s.CanonicalRelPath(newRelPath)
	if err != nil {
		return err
	}
	database := s.db
	if tx != nil {
		database = tx
	}
	if database == nil {
		return nil
	}

	// 1. Update the direct target record
	err = database.WithContext(ctx).Model(&model.FileRecord{}).
		Where("storage_id = ? AND relative_path = ? AND is_deleted = ?", s.storageID, oldNorm, false).
		Updates(map[string]interface{}{
			"relative_path": newNorm,
			"revision":      gorm.Expr("revision + 1"),
			"updated_at":    time.Now(),
		}).Error
	if err != nil {
		return err
	}

	// 2. Update descendant records if directory
	oldPrefix := oldNorm + "/"
	var descendants []model.FileRecord
	err = database.WithContext(ctx).
		Where("storage_id = ? AND relative_path LIKE ? AND is_deleted = ?", s.storageID, oldPrefix+"%", false).
		Find(&descendants).Error
	if err != nil {
		return err
	}

	for _, d := range descendants {
		suffix := strings.TrimPrefix(d.RelativePath, oldPrefix)
		updatedChildPath := stdpath.Join(newNorm, suffix)
		if err := database.WithContext(ctx).Model(&model.FileRecord{}).
			Where("id = ?", d.ID).
			Updates(map[string]interface{}{
				"relative_path": updatedChildPath,
				"revision":      gorm.Expr("revision + 1"),
				"updated_at":    time.Now(),
			}).Error; err != nil {
			return err
		}
	}

	// 3. Update Shares matching the moved path
	if storageMountPath != "" {
		mountPrefix := stdpath.Clean("/" + storageMountPath)
		if mountPrefix == "/" {
			mountPrefix = ""
		}
		oldShareRoot := stdpath.Clean(mountPrefix + "/" + oldNorm)
		newShareRoot := stdpath.Clean(mountPrefix + "/" + newNorm)

		// Direct match on Share.RootPath
		_ = database.WithContext(ctx).Model(&model.Share{}).
			Where("root_path = ?", oldShareRoot).
			Update("root_path", newShareRoot).Error

		// Descendant match on Share.RootPath
		var subShares []model.Share
		if err := database.WithContext(ctx).
			Where("root_path LIKE ?", oldShareRoot+"/%").
			Find(&subShares).Error; err == nil {
			for _, sh := range subShares {
				subSuffix := strings.TrimPrefix(sh.RootPath, oldShareRoot+"/")
				updatedSharePath := stdpath.Join(newShareRoot, subSuffix)
				_ = database.WithContext(ctx).Model(&model.Share{}).
					Where("id = ?", sh.ID).
					Update("root_path", updatedSharePath).Error
			}
		}
	}

	return nil
}

// TouchFileRecord increments revision and updates size/modtime for an existing active record.
func (s *Service) TouchFileRecord(ctx context.Context, tx *gorm.DB, relPath string, size int64, modTime time.Time) (*model.FileRecord, error) {
	normPath, err := s.CanonicalRelPath(relPath)
	if err != nil {
		return nil, err
	}
	database := s.db
	if tx != nil {
		database = tx
	}

	rec, err := s.GetActiveFileRecord(ctx, normPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return s.GetOrCreateFileRecord(ctx, normPath, false, size, modTime)
		}
		return nil, err
	}

	if database != nil {
		err = database.WithContext(ctx).Model(&model.FileRecord{}).
			Where("id = ?", rec.ID).
			Updates(map[string]interface{}{
				"revision":   rec.Revision + 1,
				"size":       size,
				"mod_time":   modTime,
				"updated_at": time.Now(),
			}).Error
		if err != nil {
			return nil, err
		}
		rec.Revision++
		rec.Size = size
		rec.ModTime = modTime
	}
	return rec, nil
}
