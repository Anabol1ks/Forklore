package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"ranking-service/internal/domain"
	"ranking-service/internal/model"
	"ranking-service/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxDailyActivityPoints = int64(40)
)

var globalTagID = uuid.Nil

type rankingService struct {
	repos   *repository.Repository
	logger  *zap.Logger
	nowFunc func() time.Time
}

func NewRankingService(repos *repository.Repository, logger *zap.Logger) Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &rankingService{
		repos:   repos,
		logger:  logger,
		nowFunc: time.Now,
	}
}

func (s *rankingService) ListOverall(ctx context.Context, params ListParams) ([]LeaderboardEntry, int64, error) {
	limit, offset, err := normalizeListParams(params)
	if err != nil {
		return nil, 0, err
	}

	entries := make([]model.UserRatingStat, 0, limit)
	var total int64

	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserRatingStat{}).Count(&total).Error; err != nil {
			return err
		}

		return tx.Order("overall_score DESC").Order("updated_at DESC").Limit(limit).Offset(offset).Find(&entries).Error
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]LeaderboardEntry, 0, len(entries))
	for _, item := range entries {
		result = append(result, mapUserRatingToEntry(item, LeaderboardTypeOverall, nil))
	}
	return result, total, nil
}

func (s *rankingService) ListMonthly(ctx context.Context, params ListParams) ([]LeaderboardEntry, int64, error) {
	limit, offset, err := normalizeListParams(params)
	if err != nil {
		return nil, 0, err
	}

	entries := make([]model.UserRatingStat, 0, limit)
	var total int64

	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserRatingStat{}).Count(&total).Error; err != nil {
			return err
		}

		return tx.Order("monthly_score DESC").Order("updated_at DESC").Limit(limit).Offset(offset).Find(&entries).Error
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]LeaderboardEntry, 0, len(entries))
	for _, item := range entries {
		result = append(result, mapUserRatingToEntry(item, LeaderboardTypeMonthly, nil))
	}
	return result, total, nil
}

func (s *rankingService) ListSubject(ctx context.Context, params ListSubjectParams) ([]LeaderboardEntry, int64, error) {
	limit, offset, err := normalizeListParams(ListParams{Limit: params.Limit, Offset: params.Offset})
	if err != nil {
		return nil, 0, err
	}
	if params.TagID == uuid.Nil {
		return nil, 0, domain.ErrInvalidTagID
	}

	type joined struct {
		model.UserSubjectRatingStat
		Username    string
		DisplayName string
		AvatarURL   string
		TitleLabel  string
	}

	rows := make([]joined, 0, limit)
	var total int64

	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserSubjectRatingStat{}).Where("tag_id = ?", params.TagID).Count(&total).Error; err != nil {
			return err
		}

		return tx.Table("user_subject_rating_stats as s").
			Select("s.*, u.username, u.display_name, u.avatar_url, u.title_label").
			Joins("left join user_rating_stats u on u.user_id = s.user_id").
			Where("s.tag_id = ?", params.TagID).
			Order("s.subject_score DESC").
			Order("s.updated_at DESC").
			Limit(limit).
			Offset(offset).
			Scan(&rows).Error
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]LeaderboardEntry, 0, len(rows))
	for _, row := range rows {
		tagID := row.TagID
		result = append(result, LeaderboardEntry{
			UserID:                  row.UserID,
			TagID:                   &tagID,
			Username:                row.Username,
			DisplayName:             row.DisplayName,
			AvatarURL:               row.AvatarURL,
			TitleLabel:              row.TitleLabel,
			Score:                   row.SubjectScore,
			SubjectScore:            row.SubjectScore,
			StarsReceivedTotal:      row.StarsReceivedTotal,
			ForksReceivedTotal:      row.ForksReceivedTotal,
			PublicRepositoriesCount: row.PublicReposCount,
			ActivityPoints30d:       row.ActivityPoints30d,
		})
	}

	return result, total, nil
}

func (s *rankingService) EnsureUser(ctx context.Context, userID uuid.UUID, username string) error {
	if userID == uuid.Nil {
		return nil
	}

	now := s.nowFunc().UTC()
	entry := model.UserRatingStat{
		UserID:    userID,
		Username:  strings.TrimSpace(username),
		UpdatedAt: now,
		CreatedAt: now,
	}

	return s.repos.DB().WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"username", "updated_at"}),
	}).Create(&entry).Error
}

func (s *rankingService) ApplyEvent(ctx context.Context, event UserEvent) error {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.nowFunc().UTC()
	}
	if event.Delta == 0 {
		event.Delta = 1
	}
	if event.Points == 0 {
		event.Points = 1
	}

	effectiveUserID := event.UserID
	if effectiveUserID == uuid.Nil {
		effectiveUserID = event.OwnerID
	}
	if effectiveUserID == uuid.Nil {
		return nil
	}

	return s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ensureUserTx(ctx, tx, effectiveUserID, event.Username); err != nil {
			return err
		}

		switch strings.ToLower(strings.TrimSpace(event.Type)) {
		case "user.followed":
			if err := s.bumpFollowersTx(ctx, tx, effectiveUserID, event.Delta); err != nil {
				return err
			}
			if err := s.bumpDailyTx(ctx, tx, effectiveUserID, nil, event.OccurredAt, 0, 0, event.Delta); err != nil {
				return err
			}
		case "user.unfollowed":
			if err := s.bumpFollowersTx(ctx, tx, effectiveUserID, -absInt64(event.Delta)); err != nil {
				return err
			}
		case "repo.created":
			if event.IsPublic {
				if err := s.bumpPublicReposTx(ctx, tx, effectiveUserID, event.TagID, event.Delta); err != nil {
					return err
				}
			}
			if err := s.bumpActivityTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, maxInt64(1, event.Points)); err != nil {
				return err
			}
		case "repo.visibility.changed":
			if event.IsPublic {
				if err := s.bumpPublicReposTx(ctx, tx, effectiveUserID, event.TagID, absInt64(event.Delta)); err != nil {
					return err
				}
			} else {
				if err := s.bumpPublicReposTx(ctx, tx, effectiveUserID, event.TagID, -absInt64(event.Delta)); err != nil {
					return err
				}
			}
		case "repo.forked":
			if err := s.bumpForksTx(ctx, tx, effectiveUserID, event.TagID, event.Delta); err != nil {
				return err
			}
			if err := s.bumpDailyTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, 0, event.Delta, 0); err != nil {
				return err
			}
		case "repo.starred":
			if err := s.bumpStarsTx(ctx, tx, effectiveUserID, event.TagID, event.Delta); err != nil {
				return err
			}
			if err := s.bumpDailyTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, event.Delta, 0, 0); err != nil {
				return err
			}
		case "repo.unstarred":
			if err := s.bumpStarsTx(ctx, tx, effectiveUserID, event.TagID, -absInt64(event.Delta)); err != nil {
				return err
			}
		case "document.created":
			if err := s.bumpActivityTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, maxInt64(4, event.Points)); err != nil {
				return err
			}
		case "document.version.created":
			if err := s.bumpActivityTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, maxInt64(2, event.Points)); err != nil {
				return err
			}
		case "file.created":
			if err := s.bumpActivityTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, maxInt64(2, event.Points)); err != nil {
				return err
			}
		case "file.version.created":
			if err := s.bumpActivityTx(ctx, tx, effectiveUserID, event.TagID, event.OccurredAt, maxInt64(1, event.Points)); err != nil {
				return err
			}
		default:
			s.logger.Debug("ranking event ignored", zap.String("type", event.Type))
			return nil
		}

		if err := s.recomputeUserScoresTx(ctx, tx, effectiveUserID); err != nil {
			return err
		}

		if event.TagID != nil && *event.TagID != uuid.Nil {
			if err := s.recomputeSubjectScoreTx(ctx, tx, effectiveUserID, *event.TagID); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *rankingService) ensureUserTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, username string) error {
	if userID == uuid.Nil {
		return nil
	}

	now := s.nowFunc().UTC()
	entry := model.UserRatingStat{
		UserID:    userID,
		Username:  strings.TrimSpace(username),
		UpdatedAt: now,
		CreatedAt: now,
	}

	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"username":   gorm.Expr("CASE WHEN user_rating_stats.username = '' THEN ? ELSE user_rating_stats.username END", entry.Username),
			"updated_at": now,
		}),
	}).Create(&entry).Error
}

func (s *rankingService) bumpFollowersTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, delta int64) error {
	return tx.WithContext(ctx).Model(&model.UserRatingStat{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"followers_count": gorm.Expr("GREATEST(0, followers_count + ?)", delta),
			"updated_at":      s.nowFunc().UTC(),
		}).Error
}

func (s *rankingService) bumpPublicReposTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tagID *uuid.UUID, delta int64) error {
	now := s.nowFunc().UTC()
	if err := tx.WithContext(ctx).Model(&model.UserRatingStat{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"public_repositories_count": gorm.Expr("GREATEST(0, public_repositories_count + ?)", delta),
			"updated_at":                now,
		}).Error; err != nil {
		return err
	}

	if tagID == nil || *tagID == uuid.Nil {
		return nil
	}

	subject := model.UserSubjectRatingStat{
		UserID:    userID,
		TagID:     *tagID,
		UpdatedAt: now,
		CreatedAt: now,
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "tag_id"}}, DoNothing: true}).Create(&subject).Error; err != nil {
		return err
	}

	return tx.WithContext(ctx).Model(&model.UserSubjectRatingStat{}).
		Where("user_id = ? AND tag_id = ?", userID, *tagID).
		Updates(map[string]any{
			"public_repositories_count": gorm.Expr("GREATEST(0, public_repositories_count + ?)", delta),
			"updated_at":                now,
		}).Error
}

func (s *rankingService) bumpStarsTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tagID *uuid.UUID, delta int64) error {
	now := s.nowFunc().UTC()
	if err := tx.WithContext(ctx).Model(&model.UserRatingStat{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"stars_received_total": gorm.Expr("GREATEST(0, stars_received_total + ?)", delta),
			"updated_at":           now,
		}).Error; err != nil {
		return err
	}

	if tagID == nil || *tagID == uuid.Nil {
		return nil
	}

	return s.bumpSubjectCounterTx(ctx, tx, userID, *tagID, "stars_received_total", delta)
}

func (s *rankingService) bumpForksTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tagID *uuid.UUID, delta int64) error {
	now := s.nowFunc().UTC()
	if err := tx.WithContext(ctx).Model(&model.UserRatingStat{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"forks_received_total": gorm.Expr("GREATEST(0, forks_received_total + ?)", delta),
			"updated_at":           now,
		}).Error; err != nil {
		return err
	}

	if tagID == nil || *tagID == uuid.Nil {
		return nil
	}

	return s.bumpSubjectCounterTx(ctx, tx, userID, *tagID, "forks_received_total", delta)
}

func (s *rankingService) bumpSubjectCounterTx(ctx context.Context, tx *gorm.DB, userID, tagID uuid.UUID, field string, delta int64) error {
	now := s.nowFunc().UTC()
	subject := model.UserSubjectRatingStat{UserID: userID, TagID: tagID, UpdatedAt: now, CreatedAt: now}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "tag_id"}}, DoNothing: true}).Create(&subject).Error; err != nil {
		return err
	}

	updates := map[string]any{
		field:        gorm.Expr("GREATEST(0, "+field+" + ?)", delta),
		"updated_at": now,
	}
	return tx.WithContext(ctx).Model(&model.UserSubjectRatingStat{}).Where("user_id = ? AND tag_id = ?", userID, tagID).Updates(updates).Error
}

func (s *rankingService) bumpActivityTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tagID *uuid.UUID, occurredAt time.Time, points int64) error {
	if points <= 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Model(&model.UserRatingStat{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"activity_points_total": gorm.Expr("GREATEST(0, activity_points_total + ?)", points),
			"updated_at":            s.nowFunc().UTC(),
		}).Error; err != nil {
		return err
	}

	if err := s.bumpDailyActivityPointsTx(ctx, tx, userID, globalTagID, occurredAt, points); err != nil {
		return err
	}

	if tagID != nil && *tagID != uuid.Nil {
		if err := s.bumpDailyActivityPointsTx(ctx, tx, userID, *tagID, occurredAt, points); err != nil {
			return err
		}
	}

	return nil
}

func (s *rankingService) bumpDailyActivityPointsTx(ctx context.Context, tx *gorm.DB, userID, tagID uuid.UUID, occurredAt time.Time, points int64) error {
	date := normalizeDate(occurredAt)
	now := s.nowFunc().UTC()

	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO user_daily_activity (user_id, tag_id, date, activity_points, stars_received, forks_received, followers_gained, updated_at, created_at)
		VALUES (?, ?, ?, 0, 0, 0, 0, ?, ?)
		ON CONFLICT (user_id, tag_id, date) DO NOTHING
	`, userID, tagID, date, now, now).Error; err != nil {
		return err
	}

	return tx.WithContext(ctx).Exec(`
		UPDATE user_daily_activity
		SET activity_points = LEAST(?, activity_points + ?), updated_at = ?
		WHERE user_id = ? AND tag_id = ? AND date = ?
	`, maxDailyActivityPoints, points, now, userID, tagID, date).Error
}

func (s *rankingService) bumpDailyTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tagID *uuid.UUID, occurredAt time.Time, stars, forks, followers int64) error {
	effectiveTagID := globalTagID
	if tagID != nil && *tagID != uuid.Nil {
		effectiveTagID = *tagID
	}

	date := normalizeDate(occurredAt)
	now := s.nowFunc().UTC()

	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO user_daily_activity (user_id, tag_id, date, activity_points, stars_received, forks_received, followers_gained, updated_at, created_at)
		VALUES (?, ?, ?, 0, 0, 0, 0, ?, ?)
		ON CONFLICT (user_id, tag_id, date) DO NOTHING
	`, userID, effectiveTagID, date, now, now).Error; err != nil {
		return err
	}

	return tx.WithContext(ctx).Exec(`
		UPDATE user_daily_activity
		SET stars_received = stars_received + ?, forks_received = forks_received + ?, followers_gained = followers_gained + ?, updated_at = ?
		WHERE user_id = ? AND tag_id = ? AND date = ?
	`, stars, forks, followers, now, userID, effectiveTagID, date).Error
}

func (s *rankingService) recomputeUserScoresTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error {
	var row model.UserRatingStat
	if err := tx.WithContext(ctx).First(&row, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	since30 := normalizeDate(s.nowFunc().UTC()).AddDate(0, 0, -29)
	since8w := normalizeDate(s.nowFunc().UTC()).AddDate(0, 0, -55)

	type metrics struct {
		Followers int64
		Stars     int64
		Forks     int64
		Activity  int64
	}
	var month metrics
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(followers_gained),0) AS followers,
		       COALESCE(SUM(stars_received),0) AS stars,
		       COALESCE(SUM(forks_received),0) AS forks,
		       COALESCE(SUM(activity_points),0) AS activity
		FROM user_daily_activity
		WHERE user_id = ? AND tag_id = ? AND date >= ?
	`, userID, globalTagID, since30).Scan(&month).Error; err != nil {
		return err
	}

	var activeWeeks int64
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(COUNT(DISTINCT DATE_TRUNC('week', date::timestamp)),0)
		FROM user_daily_activity
		WHERE user_id = ? AND tag_id = ? AND date >= ? AND activity_points > 0
	`, userID, globalTagID, since8w).Scan(&activeWeeks).Error; err != nil {
		return err
	}

	var activeMonths int64
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(COUNT(DISTINCT DATE_TRUNC('month', date::timestamp)),0)
		FROM user_daily_activity
		WHERE user_id = ? AND tag_id = ? AND activity_points > 0
	`, userID, globalTagID).Scan(&activeMonths).Error; err != nil {
		return err
	}

	monthlyScore :=
		12*math.Log1p(float64(maxInt64(0, month.Followers))) +
			14*math.Log1p(float64(maxInt64(0, month.Stars))) +
			12*math.Log1p(float64(maxInt64(0, month.Forks))) +
			float64(minInt64(maxInt64(0, month.Activity), 120)) +
			3*float64(activeWeeks)

	overallScore :=
		20*math.Log1p(float64(maxInt64(0, row.FollowersCount))) +
			10*math.Log1p(float64(maxInt64(0, row.StarsReceivedTotal))) +
			12*math.Log1p(float64(maxInt64(0, row.ForksReceivedTotal))) +
			6*math.Log1p(float64(maxInt64(0, row.PublicReposCount))) +
			0.35*float64(maxInt64(0, row.ActivityPointsTotal)) +
			2*float64(activeMonths) +
			0.15*monthlyScore

	return tx.WithContext(ctx).Model(&model.UserRatingStat{}).Where("user_id = ?", userID).Updates(map[string]any{
		"followers_gained_30d": month.Followers,
		"stars_received_30d":   month.Stars,
		"forks_received_30d":   month.Forks,
		"activity_points_30d":  month.Activity,
		"active_weeks_last_8":  activeWeeks,
		"active_months_count":  activeMonths,
		"monthly_score":        monthlyScore,
		"overall_score":        overallScore,
		"updated_at":           s.nowFunc().UTC(),
	}).Error
}

func (s *rankingService) recomputeSubjectScoreTx(ctx context.Context, tx *gorm.DB, userID, tagID uuid.UUID) error {
	since30 := normalizeDate(s.nowFunc().UTC()).AddDate(0, 0, -29)

	var activity30 int64
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(activity_points),0)
		FROM user_daily_activity
		WHERE user_id = ? AND tag_id = ? AND date >= ?
	`, userID, tagID, since30).Scan(&activity30).Error; err != nil {
		return err
	}

	var row model.UserSubjectRatingStat
	if err := tx.WithContext(ctx).First(&row, "user_id = ? AND tag_id = ?", userID, tagID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	subjectScore :=
		12*math.Log1p(float64(maxInt64(0, row.StarsReceivedTotal))) +
			12*math.Log1p(float64(maxInt64(0, row.ForksReceivedTotal))) +
			8*math.Log1p(float64(maxInt64(0, row.PublicReposCount))) +
			float64(minInt64(maxInt64(0, activity30), 100))

	return tx.WithContext(ctx).Model(&model.UserSubjectRatingStat{}).
		Where("user_id = ? AND tag_id = ?", userID, tagID).
		Updates(map[string]any{
			"activity_points_30d": activity30,
			"subject_score":       subjectScore,
			"updated_at":          s.nowFunc().UTC(),
		}).Error
}

func normalizeListParams(params ListParams) (int, int, error) {
	if params.Limit == 0 {
		params.Limit = 50
	}
	if params.Limit < 1 || params.Limit > 100 {
		return 0, 0, domain.ErrInvalidLimit
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	return params.Limit, params.Offset, nil
}

func normalizeDate(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func mapUserRatingToEntry(item model.UserRatingStat, boardType LeaderboardType, tagID *uuid.UUID) LeaderboardEntry {
	score := item.OverallScore
	if boardType == LeaderboardTypeMonthly {
		score = item.MonthlyScore
	}

	return LeaderboardEntry{
		UserID:                  item.UserID,
		TagID:                   tagID,
		Username:                item.Username,
		DisplayName:             item.DisplayName,
		AvatarURL:               item.AvatarURL,
		TitleLabel:              item.TitleLabel,
		Score:                   score,
		FollowersCount:          item.FollowersCount,
		FollowersGained30d:      item.FollowersGained30d,
		StarsReceivedTotal:      item.StarsReceivedTotal,
		StarsReceived30d:        item.StarsReceived30d,
		ForksReceivedTotal:      item.ForksReceivedTotal,
		ForksReceived30d:        item.ForksReceived30d,
		PublicRepositoriesCount: item.PublicReposCount,
		ActivityPoints30d:       item.ActivityPoints30d,
		ActivityPointsTotal:     item.ActivityPointsTotal,
		ActiveWeeksLast8:        item.ActiveWeeksLast8,
		ActiveMonthsCount:       item.ActiveMonthsCount,
	}
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
