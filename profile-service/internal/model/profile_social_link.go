package model

import (
	"time"

	"github.com/google/uuid"
)

type SocialPlatform string

const (
	SocialPlatformTelegram SocialPlatform = "telegram"
	SocialPlatformGitHub   SocialPlatform = "github"
	SocialPlatformVK       SocialPlatform = "vk"
	SocialPlatformLinkedIn SocialPlatform = "linkedin"
	SocialPlatformX        SocialPlatform = "x"
	SocialPlatformYouTube  SocialPlatform = "youtube"
	SocialPlatformWebsite  SocialPlatform = "website"
	SocialPlatformOther    SocialPlatform = "other"
)

type ProfileSocialLink struct {
	ID        uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID      `gorm:"column:user_id;type:uuid;not null;index"`
	Platform  SocialPlatform `gorm:"column:platform;type:varchar(32);not null"`
	URL       string         `gorm:"column:url;type:text;not null"`
	Label     *string        `gorm:"column:label;type:varchar(64)"`
	Position  int32          `gorm:"column:position;not null;default:0"`
	IsVisible bool           `gorm:"column:is_visible;not null;default:true"`
	CreatedAt time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`

	Profile *Profile `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (ProfileSocialLink) TableName() string {
	return "profile_social_links"
}
