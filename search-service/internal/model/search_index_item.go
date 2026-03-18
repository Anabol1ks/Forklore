package model

import (
	"time"

	"github.com/google/uuid"
)

type SearchEntityType string

const (
	SearchEntityTypeRepository SearchEntityType = "repository"
	SearchEntityTypeDocument   SearchEntityType = "document"
	SearchEntityTypeFile       SearchEntityType = "file"
)

type SearchIndexItem struct {
	ID         uuid.UUID        `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	EntityType SearchEntityType `gorm:"column:entity_type;type:varchar(32);not null;index;uniqueIndex:uq_search_index_items_entity"`
	EntityID   uuid.UUID        `gorm:"column:entity_id;type:uuid;not null;uniqueIndex:uq_search_index_items_entity"`

	RepoID  *uuid.UUID `gorm:"column:repo_id;type:uuid;index"`
	OwnerID *uuid.UUID `gorm:"column:owner_id;type:uuid;index"`
	TagID   *uuid.UUID `gorm:"column:tag_id;type:uuid;index"`

	Title       string  `gorm:"column:title;type:varchar(255);not null"`
	Description *string `gorm:"column:description;type:text"`
	Content     *string `gorm:"column:content;type:text"`
	TagName     *string `gorm:"column:tag_name;type:varchar(128)"`
	MimeType    *string `gorm:"column:mime_type;type:varchar(255)"`

	IsPublic     bool      `gorm:"column:is_public;not null;default:true;index"`
	SearchVector string    `gorm:"column:search_vector;type:tsvector;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;index"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
}

func (SearchIndexItem) TableName() string {
	return "search_index_items"
}
