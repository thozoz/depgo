package model

import (
	"time"
)

// FileRecord represents a stable random ID keyed storageID+relativePath with revision.
type FileRecord struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	FileID       string     `json:"file_id" gorm:"uniqueIndex;size:64;not null"`
	StorageID    uint       `json:"storage_id" gorm:"index:idx_storage_relpath;not null"`
	RelativePath string     `json:"relative_path" gorm:"index:idx_storage_relpath;size:4096;not null"`
	Revision     int64      `json:"revision" gorm:"default:1;not null"`
	Size         int64      `json:"size" gorm:"not null"`
	ModTime      time.Time  `json:"mod_time" gorm:"not null"`
	IsDir        bool       `json:"is_dir" gorm:"not null"`
	IsDeleted    bool       `json:"is_deleted" gorm:"default:false;index;not null"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TrashItem represents an item moved to trash with original path, deleted time, and private blob location.
type TrashItem struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	TrashID      string    `json:"trash_id" gorm:"uniqueIndex;size:64;not null"`
	StorageID    uint      `json:"storage_id" gorm:"index;not null"`
	FileID       string    `json:"file_id" gorm:"index;size:64;not null"`
	OriginalPath string    `json:"original_path" gorm:"size:4096;not null"`
	BlobLocation string    `json:"blob_location" gorm:"size:4096;not null"`
	Size         int64     `json:"size" gorm:"not null"`
	IsDir        bool      `json:"is_dir" gorm:"not null"`
	DeletedAt    time.Time `json:"deleted_at" gorm:"not null;index"`
	CreatedAt    time.Time `json:"created_at"`
}

// FileVersion represents a historical snapshot of a file prior to overwrite or restore.
type FileVersion struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	VersionID    string    `json:"version_id" gorm:"uniqueIndex;size:64;not null"`
	StorageID    uint      `json:"storage_id" gorm:"index;not null"`
	FileID       string    `json:"file_id" gorm:"index;size:64;not null"`
	RelativePath string    `json:"relative_path" gorm:"size:4096;not null"`
	Revision     int64     `json:"revision" gorm:"not null"`
	BlobLocation string    `json:"blob_location" gorm:"size:4096;not null"`
	Size         int64     `json:"size" gorm:"not null"`
	ModTime      time.Time `json:"mod_time" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;index"`
}

// PendingOperation tracks crash recovery state for two-phase / stage-write operations.
type PendingOperation struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	OpID           string    `json:"op_id" gorm:"uniqueIndex;size:64;not null"`
	StorageID      uint      `json:"storage_id" gorm:"index;not null"`
	OpType         string    `json:"op_type" gorm:"size:32;not null"` // "put", "move", "rename", "remove", "restore"
	Stage          string    `json:"stage" gorm:"size:32;not null"`   // "staged", "swapped", "in_progress", "failed"
	SrcPath        string    `json:"src_path" gorm:"size:4096"`
	DstPath        string    `json:"dst_path" gorm:"size:4096"`
	TempLocation   string    `json:"temp_location" gorm:"size:4096"`
	BackupLocation string    `json:"backup_location" gorm:"size:4096"`
	FileRecordID   string    `json:"file_record_id" gorm:"size:64"`
	Revision       int64     `json:"revision"`
	OriginalSize   int64     `json:"original_size"`
	OriginalModTime time.Time `json:"original_mod_time"`
	IsDir          bool      `json:"is_dir"`
	TrashID        string    `json:"trash_id" gorm:"size:64"`
	MountPath      string    `json:"mount_path" gorm:"size:1024"`
	ErrorMsg       string    `json:"error_msg,omitempty" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
