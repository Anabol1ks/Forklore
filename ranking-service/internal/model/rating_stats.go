package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRatingStat struct {
	UserID              uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	Username            string    `gorm:"column:username;type:varchar(64);not null;default:''"`
	DisplayName         string    `gorm:"column:display_name;type:varchar(128);not null;default:''"`
	AvatarURL           string    `gorm:"column:avatar_url;type:text;not null;default:''"`
	TitleLabel          string    `gorm:"column:title_label;type:varchar(128);not null;default:''"`
	FollowersCount      int64     `gorm:"column:followers_count;not null;default:0"`
	FollowersGained30d  int64     `gorm:"column:followers_gained_30d;not null;default:0"`
	StarsReceivedTotal  int64     `gorm:"column:stars_received_total;not null;default:0"`
	StarsReceived30d    int64     `gorm:"column:stars_received_30d;not null;default:0"`
	ForksReceivedTotal  int64     `gorm:"column:forks_received_total;not null;default:0"`
	ForksReceived30d    int64     `gorm:"column:forks_received_30d;not null;default:0"`
	PublicReposCount    int64     `gorm:"column:public_repositories_count;not null;default:0"`
	ActivityPointsTotal int64     `gorm:"column:activity_points_total;not null;default:0"`
	ActivityPoints30d   int64     `gorm:"column:activity_points_30d;not null;default:0"`
	ActiveWeeksLast8    int64     `gorm:"column:active_weeks_last_8;not null;default:0"`
	ActiveMonthsCount   int64     `gorm:"column:active_months_count;not null;default:0"`
	OverallScore        float64   `gorm:"column:overall_score;not null;default:0"`
	MonthlyScore        float64   `gorm:"column:monthly_score;not null;default:0"`
	UpdatedAt           time.Time `gorm:"column:updated_at;not null"`
	CreatedAt           time.Time `gorm:"column:created_at;not null"`
}

func (UserRatingStat) TableName() string {
	return "user_rating_stats"
}

type UserSubjectRatingStat struct {
	UserID             uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	TagID              uuid.UUID `gorm:"column:tag_id;type:uuid;primaryKey"`
	StarsReceivedTotal int64     `gorm:"column:stars_received_total;not null;default:0"`
	ForksReceivedTotal int64     `gorm:"column:forks_received_total;not null;default:0"`
	PublicReposCount   int64     `gorm:"column:public_repositories_count;not null;default:0"`
	ActivityPoints30d  int64     `gorm:"column:activity_points_30d;not null;default:0"`
	SubjectScore       float64   `gorm:"column:subject_score;not null;default:0"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null"`
	CreatedAt          time.Time `gorm:"column:created_at;not null"`
}

func (UserSubjectRatingStat) TableName() string {
	return "user_subject_rating_stats"
}

type UserDailyActivity struct {
	UserID          uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	TagID           uuid.UUID `gorm:"column:tag_id;type:uuid;primaryKey"`
	Date            time.Time `gorm:"column:date;type:date;primaryKey"`
	ActivityPoints  int64     `gorm:"column:activity_points;not null;default:0"`
	StarsReceived   int64     `gorm:"column:stars_received;not null;default:0"`
	ForksReceived   int64     `gorm:"column:forks_received;not null;default:0"`
	FollowersGained int64     `gorm:"column:followers_gained;not null;default:0"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
}

func (UserDailyActivity) TableName() string {
	return "user_daily_activity"
}
