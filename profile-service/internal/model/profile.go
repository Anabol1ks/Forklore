package model

import (
	"time"

	"github.com/google/uuid"
)

type ProfileTitleSource string

const (
	ProfileTitleSourceSystem      ProfileTitleSource = "system"
	ProfileTitleSourceManual      ProfileTitleSource = "manual"
	ProfileTitleSourceAchievement ProfileTitleSource = "achievement"
)

type Profile struct {
	UserID         uuid.UUID          `gorm:"column:user_id;type:uuid;primaryKey"`
	Username       string             `gorm:"column:username;type:varchar(32);not null;uniqueIndex"`
	DisplayName    string             `gorm:"column:display_name;type:varchar(100);not null"`
	Bio            *string            `gorm:"column:bio;type:text"`
	AvatarURL      *string            `gorm:"column:avatar_url;type:text"`
	CoverURL       *string            `gorm:"column:cover_url;type:text"`
	Location       *string            `gorm:"column:location;type:varchar(100)"`
	WebsiteURL     *string            `gorm:"column:website_url;type:text"`
	ReadmeMarkdown *string            `gorm:"column:readme_markdown;type:text"`
	TitleCode      *string            `gorm:"column:title_code;type:varchar(64);index"`
	TitleSource    ProfileTitleSource `gorm:"column:title_source;type:varchar(32);not null;default:'system'"`
	IsPublic       bool               `gorm:"column:is_public;not null;default:true"`
	CreatedAt      time.Time          `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt      time.Time          `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`

	Title       *ProfileTitle        `gorm:"foreignKey:TitleCode;references:Code;constraint:OnDelete:SET NULL"`
	SocialLinks []*ProfileSocialLink `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (Profile) TableName() string {
	return "profiles"
}
