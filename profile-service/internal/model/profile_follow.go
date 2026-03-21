package model

import (
	"time"

	"github.com/google/uuid"
)

type ProfileFollow struct {
	FollowerID uuid.UUID `gorm:"column:follower_id;type:uuid;primaryKey"`
	FolloweeID uuid.UUID `gorm:"column:followee_id;type:uuid;primaryKey"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
}

func (ProfileFollow) TableName() string {
	return "profile_follows"
}
