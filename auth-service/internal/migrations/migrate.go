package migrations

import (
	model "auth-service/internal/models"
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	log.Info("Начало миграции базы данных аутентификации")

	// Расширения (генераторы UUID, крипта, триграммы)
	log.Info("Создание расширений PostgreSQL")
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		log.Error("Не удалось включить расширение pgcrypto", zap.Error(err))
		return err
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Error("Не удалось включить расширение uuid-ossp", zap.Error(err))
		return err
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pg_trgm`).Error; err != nil {
		log.Error("Не удалось включить расширение pg_trgm", zap.Error(err))
		return err
	}
	log.Info("Расширения PostgreSQL успешно созданы")

	log.Info("Создание базовых таблиц")
	modelsAny := []any{
		&model.User{},
		&model.RefreshSession{},
	}
	if err := db.AutoMigrate(modelsAny...); err != nil {
		log.Error("Не удалось создать базовые таблицы", zap.Error(err))
		return err
	}
	log.Info("Базовые таблицы успешно созданы")

	log.Info("Применение дополнительных ограничений и индексов (CHECK, FK с ON DELETE, индексы)")

	stmts := []string{
		// CHECK constraints
		`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_role') THEN
		ALTER TABLE users ADD CONSTRAINT chk_users_role CHECK (role IN ('user','admin'));
		END IF;
		END$$;`,
		`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_status') THEN
		ALTER TABLE users ADD CONSTRAINT chk_users_status CHECK (status IN ('active','blocked','deleted'));
		END IF;
		END$$;`,
		`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_username_len') THEN
		ALTER TABLE users ADD CONSTRAINT chk_users_username_len CHECK (char_length(username) BETWEEN 3 AND 32);
		END IF;
		END$$;`,
		`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_email_len') THEN
		ALTER TABLE users ADD CONSTRAINT chk_users_email_len CHECK (char_length(email) BETWEEN 5 AND 254);
		END IF;
		END$$;`,

		// Foreign key with ON DELETE SET NULL for rotated_from_session_id
		`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_refresh_sessions_rotated_from') THEN
		ALTER TABLE refresh_sessions ADD CONSTRAINT fk_refresh_sessions_rotated_from FOREIGN KEY (rotated_from_session_id) REFERENCES refresh_sessions(id) ON DELETE SET NULL;
		END IF;
		END$$;`,

		// Indexes (if not exist)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);`,
		`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_sessions_token_hash ON refresh_sessions(token_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user_id ON refresh_sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user_id_revoked_at ON refresh_sessions(user_id, revoked_at);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_sessions_expires_at ON refresh_sessions(expires_at);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_sessions_revoked_at ON refresh_sessions(revoked_at);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			log.Error("Ошибка при применении SQL-выражения миграции", zap.Error(err))
			return err
		}
	}
	return nil
}
