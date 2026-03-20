package model

import (
	"time"

	"github.com/google/uuid"
)

type FileVersion struct {
	ID             uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	FileID         uuid.UUID `gorm:"column:file_id;type:uuid;not null;index"`
	UploadedBy     uuid.UUID `gorm:"column:uploaded_by;type:uuid;not null"`
	VersionNumber  uint32    `gorm:"column:version_number;not null"`
	StorageKey     string    `gorm:"column:storage_key;type:text;not null"`
	MimeType       string    `gorm:"column:mime_type;type:varchar(255);not null"`
	SizeBytes      uint64    `gorm:"column:size_bytes;not null"`
	ChecksumSHA256 *string   `gorm:"column:checksum_sha256;type:varchar(64)"`
	ChangeSummary  *string   `gorm:"column:change_summary;type:varchar(255)"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()"`

	File *File `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE"`
}

func (FileVersion) TableName() string {
	return "file_versions"
}
