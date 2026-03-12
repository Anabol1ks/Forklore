package migrations

import (
	"context"
	"repository-service/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	log.Info("Начало миграции базы данных репозиториев")

	// Расширения
	log.Info("Создание расширений PostgreSQL")
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		log.Error("Не удалось включить расширение pgcrypto", zap.Error(err))
		return err
	}
	log.Info("Расширения PostgreSQL успешно созданы")

	// Таблицы через GORM AutoMigrate
	log.Info("Создание базовых таблиц")
	if err := db.AutoMigrate(&model.Repository{}); err != nil {
		log.Error("Не удалось создать базовые таблицы", zap.Error(err))
		return err
	}
	log.Info("Базовые таблицы успешно созданы")

	// CHECK-ограничения
	log.Info("Создание CHECK-ограничений")
	checks := []struct {
		name string
		sql  string
	}{
		{
			"chk_repositories_name_len",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_name_len
				CHECK (char_length(name) BETWEEN 3 AND 100)`,
		},
		{
			"chk_repositories_slug_len",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_slug_len
				CHECK (char_length(slug) BETWEEN 3 AND 64)`,
		},
		{
			"chk_repositories_slug_format",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_slug_format
				CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{1,62}[a-z0-9])?$')`,
		},
		{
			"chk_repositories_visibility",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_visibility
				CHECK (visibility IN ('public', 'private'))`,
		},
		{
			"chk_repositories_type",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_type
				CHECK (type IN ('article', 'notes', 'mixed'))`,
		},
	}
	for _, c := range checks {
		if err := addConstraintIfNotExists(db, "repositories", c.name, c.sql); err != nil {
			log.Error("Не удалось создать ограничение", zap.String("constraint", c.name), zap.Error(err))
			return err
		}
	}
	log.Info("CHECK-ограничения успешно созданы")

	// Частичные индексы
	log.Info("Создание частичных индексов")
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_repositories_owner_slug_active
			ON repositories(owner_id, slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_owner_id_active
			ON repositories(owner_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_parent_repo_id_active
			ON repositories(parent_repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_visibility_active
			ON repositories(visibility) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_created_at_active
			ON repositories(created_at DESC) WHERE deleted_at IS NULL`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Error("Не удалось создать индекс", zap.String("sql", idx), zap.Error(err))
			return err
		}
	}
	log.Info("Частичные индексы успешно созданы")

	return nil
}

// addConstraintIfNotExists добавляет ограничение, если оно ещё не существует.
func addConstraintIfNotExists(db *gorm.DB, table, constraint, ddl string) error {
	var exists bool
	err := db.Raw(
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE table_name = ? AND constraint_name = ?
		)`, table, constraint,
	).Scan(&exists).Error
	if err != nil {
		return err
	}
	if !exists {
		return db.Exec(ddl).Error
	}
	return nil
}
