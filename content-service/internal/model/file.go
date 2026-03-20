package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	RepoID           uuid.UUID  `gorm:"column:repo_id;type:uuid;not null;index"`
	UploadedBy       uuid.UUID  `gorm:"column:uploaded_by;type:uuid;not null;index"`
	FileName         string     `gorm:"column:file_name;type:varchar(255);not null"`
	CurrentVersionID *uuid.UUID `gorm:"column:current_version_id;type:uuid"`

	CreatedAt time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

	CurrentVersion *FileVersion   `gorm:"foreignKey:CurrentVersionID;references:ID;constraint:OnDelete:SET NULL"`
	Versions       []*FileVersion `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE"`
}

func (File) TableName() string {
	return "files"
}
