package migration

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	_ = ctx

	log.Info("Начало миграции базы данных профилей")

	log.Info("Создание расширений PostgreSQL")
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		log.Error("Не удалось включить расширение pgcrypto", zap.Error(err))
		return err
	}

	log.Info("Создание таблицы profile_titles_catalog")
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS profile_titles_catalog (
			code VARCHAR(64) PRIMARY KEY,
			label VARCHAR(100) NOT NULL,
			description TEXT NULL,
			sort_order INT NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			is_system BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

			CONSTRAINT chk_profile_titles_catalog_code_len
				CHECK (char_length(code) BETWEEN 1 AND 64),

			CONSTRAINT chk_profile_titles_catalog_label_len
				CHECK (char_length(label) BETWEEN 1 AND 100)
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу profile_titles_catalog", zap.Error(err))
		return err
	}

	log.Info("Создание таблицы profiles")
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS profiles (
			user_id UUID PRIMARY KEY,
			username VARCHAR(32) NOT NULL UNIQUE,
			display_name VARCHAR(100) NOT NULL,
			bio TEXT NULL,
			avatar_url TEXT NULL,
			cover_url TEXT NULL,
			location VARCHAR(100) NULL,
			website_url TEXT NULL,
			readme_markdown TEXT NULL,
			title_code VARCHAR(64) NULL REFERENCES profile_titles_catalog(code) ON DELETE SET NULL,
			title_source VARCHAR(32) NOT NULL DEFAULT 'system',
			is_public BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

			CONSTRAINT chk_profiles_username_len
				CHECK (char_length(username) BETWEEN 3 AND 32),

			CONSTRAINT chk_profiles_username_format
				CHECK (username ~ '^[a-z0-9](?:[a-z0-9._-]{1,30}[a-z0-9])?$'),

			CONSTRAINT chk_profiles_display_name_len
				CHECK (char_length(display_name) BETWEEN 1 AND 100),

			CONSTRAINT chk_profiles_bio_len
				CHECK (bio IS NULL OR char_length(bio) <= 1000),

			CONSTRAINT chk_profiles_location_len
				CHECK (location IS NULL OR char_length(location) <= 100),

			CONSTRAINT chk_profiles_title_source
				CHECK (title_source IN ('system', 'manual', 'achievement'))
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу profiles", zap.Error(err))
		return err
	}

	log.Info("Создание таблицы profile_social_links")
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS profile_social_links (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES profiles(user_id) ON DELETE CASCADE,
			platform VARCHAR(32) NOT NULL,
			url TEXT NOT NULL,
			label VARCHAR(64) NULL,
			position INT NOT NULL DEFAULT 0,
			is_visible BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

			CONSTRAINT chk_profile_social_links_platform
				CHECK (platform IN (
					'telegram',
					'github',
					'vk',
					'linkedin',
					'x',
					'youtube',
					'website',
					'other'
				)),

			CONSTRAINT chk_profile_social_links_label_len
				CHECK (label IS NULL OR char_length(label) <= 64)
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу profile_social_links", zap.Error(err))
		return err
	}

	log.Info("Создание таблицы profile_follows")
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS profile_follows (
			follower_id UUID NOT NULL REFERENCES profiles(user_id) ON DELETE CASCADE,
			followee_id UUID NOT NULL REFERENCES profiles(user_id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

			PRIMARY KEY (follower_id, followee_id),

			CONSTRAINT chk_profile_follows_not_self
				CHECK (follower_id <> followee_id)
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу profile_follows", zap.Error(err))
		return err
	}

	log.Info("Создание индексов")
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_profiles_title_code ON profiles(title_code)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_is_public ON profiles(is_public)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_social_links_user_id ON profile_social_links(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_social_links_user_id_position ON profile_social_links(user_id, position)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_follows_followee_id ON profile_follows(followee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_follows_follower_id ON profile_follows(follower_id)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Error("Не удалось создать индекс", zap.String("sql", idx), zap.Error(err))
			return err
		}
	}

	log.Info("Сидирование каталога титулов")
	if err := db.Exec(`
		INSERT INTO profile_titles_catalog (code, label, description, sort_order, is_active, is_system)
		VALUES
			('comer', 'Участник', 'Базовый титул нового пользователя', 10, TRUE, TRUE),
			('mentor', 'Ментор', 'Репетитор высшего уровня', 20, TRUE, TRUE)
		ON CONFLICT (code) DO NOTHING
	`).Error; err != nil {
		log.Error("Не удалось выполнить сидирование profile_titles_catalog", zap.Error(err))
		return err
	}

	log.Info("Миграция базы данных профилей успешно завершена")
	return nil
}
