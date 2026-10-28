package localstate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	cp "github.com/otiai10/copy"
	"gorm.io/gorm"
)

// SnapshotFile archives the current content of an existing file into the versions directory
// and records a FileVersion in the database.
func (s *Service) SnapshotFile(ctx context.Context, tx *gorm.DB, rec *model.FileRecord, canonicalRel string) (*model.FileVersion, error) {
	fi, err := s.osRoot.Stat(canonicalRel)
	if err != nil {
		if errorsIsNotExist(err) {
			return nil, nil // No existing content to archive
		}
		return nil, fmt.Errorf("stat for snapshot [%s]: %w", canonicalRel, err)
	}
	if fi.IsDir() {
		return nil, nil // Snapshot only regular files
	}

	diskPath, err := s.RelToDiskPath(canonicalRel)
	if err != nil {
		return nil, err
	}

	versionID := "ver_" + random.String(24)
	blobName := fmt.Sprintf("%d_%s_%s", s.storageID, rec.FileID, versionID)
	blobPath := filepath.Join(s.versionPath, blobName)

	if err := cp.Copy(diskPath, blobPath, cp.Options{
		Sync:          true,
		PreserveTimes: true,
	}); err != nil {
		return nil, fmt.Errorf("snapshot copy: %w", err)
	}

	fv := &model.FileVersion{
		VersionID:    versionID,
		StorageID:    s.storageID,
		FileID:       rec.FileID,
		RelativePath: rec.RelativePath,
		Revision:     rec.Revision,
		BlobLocation: blobPath,
		Size:         fi.Size(),
		ModTime:      fi.ModTime(),
		CreatedAt:    time.Now(),
	}

	database := s.db
	if tx != nil {
		database = tx
	}
	if database != nil {
		if err := database.WithContext(ctx).Create(fv).Error; err != nil {
			_ = os.Remove(blobPath)
			return nil, fmt.Errorf("record version: %w", err)
		}
	}

	return fv, nil
}

// ListFileVersions retrieves the version history for a given file ID.
func (s *Service) ListFileVersions(ctx context.Context, fileID string) ([]model.FileVersion, error) {
	if s.db == nil {
		return nil, nil
	}
	var versions []model.FileVersion
	err := s.db.WithContext(ctx).
		Where("storage_id = ? AND file_id = ?", s.storageID, fileID).
		Order("revision desc").
		Find(&versions).Error
	return versions, err
}

// GetFileVersion retrieves a single version record by its version ID.
func (s *Service) GetFileVersion(ctx context.Context, versionID string) (*model.FileVersion, error) {
	if s.db == nil {
		return nil, ErrNotFound
	}
	var fv model.FileVersion
	err := s.db.WithContext(ctx).
		Where("storage_id = ? AND version_id = ?", s.storageID, versionID).
		Take(&fv).Error
	if err != nil {
		return nil, err
	}
	return &fv, nil
}

// OpenVersionBlob opens the stored file blob for a version.
func (s *Service) OpenVersionBlob(fv *model.FileVersion) (io.ReadCloser, error) {
	return os.Open(fv.BlobLocation)
}

// RestoreVersion restores an older version of a file, preserving the current live content
// as a new snapshot in version history first.
// If expectedRevision > 0 and the current revision does not match, ErrConflict is returned.
func (s *Service) RestoreVersion(ctx context.Context, fileID, versionID string, expectedRevision int64) (*model.FileRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := s.GetFileRecordByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("file record not found: %w", err)
	}
	if rec.IsDeleted {
		return nil, fmt.Errorf("cannot restore version for deleted file")
	}
	if expectedRevision > 0 && rec.Revision != expectedRevision {
		return nil, ErrConflict
	}

	fv, err := s.GetFileVersion(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}
	if fv.FileID != fileID {
		return nil, fmt.Errorf("version [%s] does not belong to file [%s]", versionID, fileID)
	}

	liveDiskPath, err := s.RelToDiskPath(rec.RelativePath)
	if err != nil {
		return nil, err
	}

	// 1. Snapshot current live content first
	if _, snapErr := s.SnapshotFile(ctx, nil, rec, rec.RelativePath); snapErr != nil {
		return nil, fmt.Errorf("failed snapshotting current live content before version restore: %w", snapErr)
	}

	// 2. Stage-copy the version blob into live file location
	tempFileName := fmt.Sprintf("stg_ver_%s", random.String(12))
	tempDiskPath := filepath.Join(s.stagingPath, tempFileName)

	if err := cp.Copy(fv.BlobLocation, tempDiskPath, cp.Options{
		Sync:          true,
		PreserveTimes: true,
	}); err != nil {
		return nil, fmt.Errorf("copy version blob to staging: %w", err)
	}

	// 3. Atomic replace
	if err := s.CrossRootRenameReplace(tempDiskPath, liveDiskPath); err != nil {
		_ = os.Remove(tempDiskPath)
		return nil, fmt.Errorf("atomic swap version to live: %w", err)
	}

	// 4. Touch record
	fi, statErr := s.osRoot.Stat(rec.RelativePath)
	size := fv.Size
	modTime := time.Now()
	if statErr == nil {
		size = fi.Size()
		modTime = fi.ModTime()
	}

	updatedRec, err := s.TouchFileRecord(ctx, nil, rec.RelativePath, size, modTime)
	if err != nil {
		return nil, fmt.Errorf("update record after version restore: %w", err)
	}

	return updatedRec, nil
}
