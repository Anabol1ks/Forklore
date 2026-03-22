package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type LeaderboardType string

const (
	LeaderboardTypeOverall LeaderboardType = "overall"
	LeaderboardTypeMonthly LeaderboardType = "monthly"
	LeaderboardTypeSubject LeaderboardType = "subject"
)

type LeaderboardEntry struct {
	UserID                  uuid.UUID
	TagID                   *uuid.UUID
	Username                string
	DisplayName             string
	AvatarURL               string
	TitleLabel              string
	Score                   float64
	FollowersCount          int64
	FollowersGained30d      int64
	StarsReceivedTotal      int64
	StarsReceived30d        int64
	ForksReceivedTotal      int64
	ForksReceived30d        int64
	PublicRepositoriesCount int64
	ActivityPoints30d       int64
	ActivityPointsTotal     int64
	ActiveWeeksLast8        int64
	ActiveMonthsCount       int64
	SubjectScore            float64
}

type ListParams struct {
	Limit  int
	Offset int
}

type ListSubjectParams struct {
	TagID  uuid.UUID
	Limit  int
	Offset int
}

type UserEvent struct {
	Type       string
	UserID     uuid.UUID
	OwnerID    uuid.UUID
	TagID      *uuid.UUID
	RepoID     *uuid.UUID
	Username   string
	IsPublic   bool
	Delta      int64
	Points     int64
	OccurredAt time.Time
}

type Service interface {
	ListOverall(ctx context.Context, params ListParams) ([]LeaderboardEntry, int64, error)
	ListMonthly(ctx context.Context, params ListParams) ([]LeaderboardEntry, int64, error)
	ListSubject(ctx context.Context, params ListSubjectParams) ([]LeaderboardEntry, int64, error)

	ApplyEvent(ctx context.Context, event UserEvent) error
	EnsureUser(ctx context.Context, userID uuid.UUID, username string) error
}
