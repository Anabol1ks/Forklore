package migration

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	_ = ctx

	log.Info("Начало миграции базы данных поиска")

	log.Info("Создание расширений PostgreSQL")
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		log.Error("Не удалось включить расширение pgcrypto", zap.Error(err))
		return err
	}
	log.Info("Расширения PostgreSQL успешно созданы")

	log.Info("Создание таблицы search_index_items")
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS search_index_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_type VARCHAR(32) NOT NULL,
			entity_id UUID NOT NULL,

			repo_id UUID NULL,
			owner_id UUID NULL,
			tag_id UUID NULL,

			title VARCHAR(255) NOT NULL,
			description TEXT NULL,
			content TEXT NULL,
			tag_name VARCHAR(128) NULL,
			mime_type VARCHAR(255) NULL,

			is_public BOOLEAN NOT NULL DEFAULT TRUE,

			search_vector tsvector NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу search_index_items", zap.Error(err))
		return err
	}

	log.Info("Обновление уникальности search_index_items: entity_type + entity_id")
	uniques := []string{
		`ALTER TABLE search_index_items DROP CONSTRAINT IF EXISTS search_index_items_entity_id_key`,
		`DROP INDEX IF EXISTS search_index_items_entity_id_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_search_index_items_entity ON search_index_items(entity_type, entity_id)`,
	}
	for _, s := range uniques {
		if err := db.Exec(s).Error; err != nil {
			log.Error("Не удалось обновить уникальные ограничения", zap.String("sql", s), zap.Error(err))
			return err
		}
	}

	if err := addConstraintIfNotExists(
		db,
		"search_index_items",
		"chk_search_index_items_entity_type",
		`ALTER TABLE search_index_items
		 ADD CONSTRAINT chk_search_index_items_entity_type
		 CHECK (entity_type IN ('repository', 'document', 'file'))`,
	); err != nil {
		log.Error("Не удалось создать CHECK-ограничение для search_index_items", zap.Error(err))
		return err
	}

	log.Info("Создание функции и триггера пересчета search_vector")
	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION update_search_index_items_search_vector()
		RETURNS trigger AS $$
		BEGIN
			NEW.search_vector :=
				setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
				setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B') ||
				setweight(to_tsvector('simple', COALESCE(NEW.content, '')), 'C') ||
				setweight(to_tsvector('simple', COALESCE(NEW.tag_name, '')), 'B') ||
				setweight(to_tsvector('simple', COALESCE(NEW.mime_type, '')), 'D');
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`).Error; err != nil {
		log.Error("Не удалось создать функцию update_search_index_items_search_vector", zap.Error(err))
		return err
	}

	if err := db.Exec(`DROP TRIGGER IF EXISTS trg_search_index_items_search_vector ON search_index_items`).Error; err != nil {
		log.Error("Не удалось удалить старый триггер trg_search_index_items_search_vector", zap.Error(err))
		return err
	}

	if err := db.Exec(`
		CREATE TRIGGER trg_search_index_items_search_vector
		BEFORE INSERT OR UPDATE OF title, description, content, tag_name, mime_type
		ON search_index_items
		FOR EACH ROW
		EXECUTE FUNCTION update_search_index_items_search_vector()
	`).Error; err != nil {
		log.Error("Не удалось создать триггер trg_search_index_items_search_vector", zap.Error(err))
		return err
	}

	if err := db.Exec(`
		UPDATE search_index_items
		SET search_vector =
			setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
			setweight(to_tsvector('simple', COALESCE(description, '')), 'B') ||
			setweight(to_tsvector('simple', COALESCE(content, '')), 'C') ||
			setweight(to_tsvector('simple', COALESCE(tag_name, '')), 'B') ||
			setweight(to_tsvector('simple', COALESCE(mime_type, '')), 'D')
	`).Error; err != nil {
		log.Error("Не удалось выполнить backfill search_vector", zap.Error(err))
		return err
	}

	log.Info("Создание индексов search_index_items")
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_entity_type ON search_index_items(entity_type)`,
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_repo_id ON search_index_items(repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_owner_id ON search_index_items(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_tag_id ON search_index_items(tag_id)`,
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_is_public ON search_index_items(is_public)`,
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_updated_at ON search_index_items(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_search_index_items_search_vector ON search_index_items USING GIN(search_vector)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Error("Не удалось создать индекс", zap.String("sql", idx), zap.Error(err))
			return err
		}
	}

	log.Info("Миграция базы данных поиска успешно завершена")
	return nil
}

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
