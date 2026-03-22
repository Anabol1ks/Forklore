package grpcserver

import (
	"ranking-service/internal/service"
	"strings"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	rankingv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &commonv1.UUID{Value: id.String()}
}

func parseRequiredUUID(id *commonv1.UUID, fieldName string) (uuid.UUID, error) {
	if id == nil {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	value := strings.TrimSpace(id.GetValue())
	if value == "" {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s must be a valid uuid", fieldName)
	}

	return parsed, nil
}

func toProtoEntry(entry service.LeaderboardEntry) *rankingv1.LeaderboardEntry {
	var tagID *commonv1.UUID
	if entry.TagID != nil {
		tagID = toProtoUUID(*entry.TagID)
	}

	return &rankingv1.LeaderboardEntry{
		UserId:                  toProtoUUID(entry.UserID),
		TagId:                   tagID,
		Username:                entry.Username,
		DisplayName:             entry.DisplayName,
		AvatarUrl:               entry.AvatarURL,
		TitleLabel:              entry.TitleLabel,
		Score:                   entry.Score,
		FollowersCount:          entry.FollowersCount,
		FollowersGained_30D:     entry.FollowersGained30d,
		StarsReceivedTotal:      entry.StarsReceivedTotal,
		StarsReceived_30D:       entry.StarsReceived30d,
		ForksReceivedTotal:      entry.ForksReceivedTotal,
		ForksReceived_30D:       entry.ForksReceived30d,
		PublicRepositoriesCount: entry.PublicRepositoriesCount,
		ActivityPointsTotal:     entry.ActivityPointsTotal,
		ActivityPoints_30D:      entry.ActivityPoints30d,
		ActiveWeeksLast_8:       entry.ActiveWeeksLast8,
		ActiveMonthsCount:       entry.ActiveMonthsCount,
		SubjectScore:            entry.SubjectScore,
	}
}

func toProtoEntries(items []service.LeaderboardEntry) []*rankingv1.LeaderboardEntry {
	result := make([]*rankingv1.LeaderboardEntry, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoEntry(item))
	}
	return result
}
