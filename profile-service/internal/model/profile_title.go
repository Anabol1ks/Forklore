package model

import "time"

type ProfileTitle struct {
	Code        string    `gorm:"column:code;type:varchar(64);primaryKey"`
	Label       string    `gorm:"column:label;type:varchar(100);not null"`
	Description *string   `gorm:"column:description;type:text"`
	SortOrder   int32     `gorm:"column:sort_order;not null;default:0"`
	IsActive    bool      `gorm:"column:is_active;not null;default:true"`
	IsSystem    bool      `gorm:"column:is_system;not null;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`
}

func (ProfileTitle) TableName() string {
	return "profile_titles_catalog"
}
