package migration

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	_ = ctx

	log.Info("ranking-db migration started")

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		log.Error("failed to enable pgcrypto", zap.Error(err))
		return err
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_rating_stats (
			user_id UUID PRIMARY KEY,
			username VARCHAR(64) NOT NULL DEFAULT '',
			display_name VARCHAR(128) NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			title_label VARCHAR(128) NOT NULL DEFAULT '',
			followers_count BIGINT NOT NULL DEFAULT 0,
			followers_gained_30d BIGINT NOT NULL DEFAULT 0,
			stars_received_total BIGINT NOT NULL DEFAULT 0,
			stars_received_30d BIGINT NOT NULL DEFAULT 0,
			forks_received_total BIGINT NOT NULL DEFAULT 0,
			forks_received_30d BIGINT NOT NULL DEFAULT 0,
			public_repositories_count BIGINT NOT NULL DEFAULT 0,
			activity_points_total BIGINT NOT NULL DEFAULT 0,
			activity_points_30d BIGINT NOT NULL DEFAULT 0,
			active_weeks_last_8 BIGINT NOT NULL DEFAULT 0,
			active_months_count BIGINT NOT NULL DEFAULT 0,
			overall_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			monthly_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		log.Error("failed to create user_rating_stats", zap.Error(err))
		return err
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_subject_rating_stats (
			user_id UUID NOT NULL,
			tag_id UUID NOT NULL,
			stars_received_total BIGINT NOT NULL DEFAULT 0,
			forks_received_total BIGINT NOT NULL DEFAULT 0,
			public_repositories_count BIGINT NOT NULL DEFAULT 0,
			activity_points_30d BIGINT NOT NULL DEFAULT 0,
			subject_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY(user_id, tag_id)
		)
	`).Error; err != nil {
		log.Error("failed to create user_subject_rating_stats", zap.Error(err))
		return err
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_daily_activity (
			user_id UUID NOT NULL,
			tag_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
			date DATE NOT NULL,
			activity_points BIGINT NOT NULL DEFAULT 0,
			stars_received BIGINT NOT NULL DEFAULT 0,
			forks_received BIGINT NOT NULL DEFAULT 0,
			followers_gained BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY(user_id, tag_id, date)
		)
	`).Error; err != nil {
		log.Error("failed to create user_daily_activity", zap.Error(err))
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_user_rating_stats_overall ON user_rating_stats(overall_score DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_rating_stats_monthly ON user_rating_stats(monthly_score DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_subject_rating_stats_tag_score ON user_subject_rating_stats(tag_id, subject_score DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_daily_activity_date ON user_daily_activity(date DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_daily_activity_user_date ON user_daily_activity(user_id, date DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_daily_activity_user_tag_date ON user_daily_activity(user_id, tag_id, date DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_daily_activity_tag_date ON user_daily_activity(tag_id, date DESC)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Error("failed to create index", zap.String("sql", idx), zap.Error(err))
			return err
		}
	}

	log.Info("ranking-db migration completed")
	return nil
}
