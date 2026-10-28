package localstate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	stdpath "path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"gorm.io/gorm"
)

var (
	ErrConflict             = errors.New("revision conflict")
	ErrInvalidTextExtension = errors.New("file extension not in text editor allowlist")
	ErrTextTooLarge         = errors.New("content exceeds 2MiB limit")
	ErrInvalidUTF8          = errors.New("content is not valid UTF-8 text")
	ErrNulByte              = errors.New("content contains NUL byte")
)

const MaxTextFileSize = 2 * 1024 * 1024 // 2 MiB

// TextAllowedExtensions specifies the exact allowlist of editable text file extensions.
var TextAllowedExtensions = map[string]bool{
	".txt":  true,
	".md":   true,
	".csv":  true,
	".json": true,
	".log":  true,
	".yaml": true,
	".yml":  true,
	".xml":  true,
	".ini":  true,
	".conf": true,
	".toml": true,
	".css":  true,
	".js":   true,
	".ts":   true,
	".html": true,
	".svg":  true,
}

// IsAllowedTextExtension checks if a filename extension is supported by the text editor.
func IsAllowedTextExtension(filename string) bool {
	ext := strings.ToLower(stdpath.Ext(filename))
	return TextAllowedExtensions[ext]
}

// ValidateTextContent verifies that content is valid UTF-8, contains no NUL bytes, and does not exceed 2MiB.
func ValidateTextContent(content []byte) error {
	if len(content) > MaxTextFileSize {
		return ErrTextTooLarge
	}
	if !utf8.Valid(content) {
		return ErrInvalidUTF8
	}
	if bytes.IndexByte(content, 0) != -1 {
		return ErrNulByte
	}
	return nil
}

// ReadText reads text under the storage lock, opening the file via osRoot, validating UTF-8 and size,
// and returning matching content and revision atomically without TOCTOU races.
func (s *Service) ReadText(ctx context.Context, storageRelPath string) (string, int64, error) {
	normRel, err := s.CanonicalRelPath(storageRelPath)
	if err != nil {
		return "", 0, err
	}
	if normRel == "" {
		return "", 0, fmt.Errorf("cannot read text from root directory")
	}

	if !IsAllowedTextExtension(normRel) {
		return "", 0, ErrInvalidTextExtension
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.CheckSymlink(normRel); err != nil {
		return "", 0, err
	}

	file, err := s.osRoot.Open(normRel)
	if err != nil {
		if errorsIsNotExist(err) {
			return "", 0, ErrNotFound
		}
		return "", 0, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	if fi.IsDir() {
		return "", 0, fmt.Errorf("path is a directory")
	}
	if fi.Size() > MaxTextFileSize {
		return "", 0, ErrTextTooLarge
	}

	// Bounded read: read up to MaxTextFileSize + 1 to detect growing files
	data, err := io.ReadAll(io.LimitReader(file, MaxTextFileSize+1))
	if err != nil {
		return "", 0, err
	}
	if len(data) > MaxTextFileSize {
		return "", 0, ErrTextTooLarge
	}

	if err := ValidateTextContent(data); err != nil {
		return "", 0, err
	}

	var rec *model.FileRecord
	if s.db != nil {
		rec, err = s.GetOrCreateFileRecord(ctx, normRel, false, int64(len(data)), fi.ModTime())
		if err != nil {
			return "", 0, err
		}
	} else {
		rec = &model.FileRecord{Revision: 1}
	}

	return string(data), rec.Revision, nil
}

// SaveText saves UTF-8 text with an in-lock revision check, snapshotting any existing content first.
// If expectedRevision > 0 and the current revision does not match, ErrConflict is returned.
func (s *Service) SaveText(ctx context.Context, storageRelPath string, content []byte, expectedRevision int64) (int64, error) {
	if err := ValidateTextContent(content); err != nil {
		return 0, err
	}

	normRel, err := s.CanonicalRelPath(storageRelPath)
	if err != nil {
		return 0, err
	}
	if normRel == "" {
		return 0, fmt.Errorf("cannot write text to root directory")
	}

	if !IsAllowedTextExtension(normRel) {
		return 0, ErrInvalidTextExtension
	}

	dstDiskPath, err := s.RelToDiskPath(normRel)
	if err != nil {
		return 0, err
	}

	parentRel := stdpath.Dir(normRel)
	if parentRel == "." {
		parentRel = ""
	}
	if err := s.CheckSymlink(parentRel); err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// In-lock revision check
	var currentRec *model.FileRecord
	if s.db != nil {
		currentRec, _ = s.GetActiveFileRecord(ctx, normRel)
	}

	var existingRevision int64
	if fi, err := s.osRoot.Stat(normRel); err == nil {
		if fi.IsDir() {
			return 0, fmt.Errorf("cannot overwrite directory with text file: %s", normRel)
		}
		if currentRec == nil && s.db != nil {
			currentRec, err = s.GetOrCreateFileRecord(ctx, normRel, false, fi.Size(), fi.ModTime())
			if err != nil {
				return 0, fmt.Errorf("failed creating file record: %w", err)
			}
		}
		if currentRec != nil {
			existingRevision = currentRec.Revision
			if expectedRevision > 0 && currentRec.Revision != expectedRevision {
				return 0, ErrConflict
			}
			// Snapshot current content into FileVersion
			if _, snapErr := s.SnapshotFile(ctx, nil, currentRec, normRel); snapErr != nil {
				return 0, fmt.Errorf("snapshot before text save failed: %w", snapErr)
			}
		}
	} else if errorsIsNotExist(err) {
		// New file creation: if client expected an existing revision, conflict
		if expectedRevision > 0 {
			return 0, ErrConflict
		}
	} else {
		return 0, err
	}

	// Stage: write into temporary file inside private staging directory
	tempFileName := fmt.Sprintf("stg_txt_%s", random.String(12))
	tempDiskPath := filepath.Join(s.stagingPath, tempFileName)

	tempFile, err := os.OpenFile(tempDiskPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return 0, fmt.Errorf("create staging file: %w", err)
	}

	var writeErr error
	defer func() {
		if writeErr != nil {
			_ = os.Remove(tempDiskPath)
		}
	}()

	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		writeErr = err
		return 0, fmt.Errorf("write staging file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		writeErr = err
		return 0, fmt.Errorf("sync staging file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		writeErr = err
		return 0, fmt.Errorf("close staging file: %w", err)
	}

	// Sync staging directory
	if stagingDir, err := os.Open(s.stagingPath); err == nil {
		_ = stagingDir.Sync()
		_ = stagingDir.Close()
	}

	op := &model.PendingOperation{
		OpType:       "put",
		Stage:        "staged",
		SrcPath:      tempDiskPath,
		DstPath:      normRel,
		TempLocation: tempDiskPath,
		Revision:     existingRevision,
		IsDir:        false,
	}
	if currentRec != nil {
		op.FileRecordID = currentRec.FileID
	}
	if err := s.BeginOperation(ctx, op); err != nil {
		writeErr = err
		return 0, fmt.Errorf("failed recording journal before text swap: %w", err)
	}

	// Atomic replace
	if err := s.CrossRootRenameReplace(tempDiskPath, dstDiskPath); err != nil {
		writeErr = err
		_ = s.EndOperation(ctx, op.OpID, false)
		return 0, fmt.Errorf("atomic replace failed for text save: %w", err)
	}

	// Sync destination directory
	if dstParentDir, err := os.Open(filepath.Dir(dstDiskPath)); err == nil {
		_ = dstParentDir.Sync()
		_ = dstParentDir.Close()
	}

	now := time.Now()
	_ = s.osRoot.Chtimes(normRel, now, now)

	newRevision := int64(1)
	if currentRec != nil {
		newRevision = currentRec.Revision + 1
	}

	// Commit DB updates and clear journal in the same transaction
	if s.db != nil {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if currentRec != nil {
				if err := tx.Model(&model.FileRecord{}).
					Where("id = ?", currentRec.ID).
					Updates(map[string]interface{}{
						"revision":   newRevision,
						"size":       int64(len(content)),
						"mod_time":   now,
						"updated_at": now,
					}).Error; err != nil {
					return err
				}
			} else {
				newRec := model.FileRecord{
					FileID:       "fil_" + random.String(16),
					StorageID:    s.storageID,
					RelativePath: normRel,
					Revision:     newRevision,
					Size:         int64(len(content)),
					ModTime:      now,
					IsDir:        false,
					IsDeleted:    false,
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				if err := tx.Create(&newRec).Error; err != nil {
					return err
				}
			}
			return tx.Where("storage_id = ? AND op_id = ?", s.storageID, op.OpID).
				Delete(&model.PendingOperation{}).Error
		})
		if err != nil {
			return 0, fmt.Errorf("text file written but DB commit failed: %w", err)
		}
	}

	return newRevision, nil
}
