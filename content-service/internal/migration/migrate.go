package migration

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	log.Info("Начало миграции базы данных контента")

	// Расширения
	log.Info("Создание расширений PostgreSQL")
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		log.Error("Не удалось включить расширение pgcrypto", zap.Error(err))
		return err
	}
	log.Info("Расширения PostgreSQL успешно созданы")

	// Создание таблиц через raw SQL
	log.Info("Создание таблиц")

	// documents
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			repo_id UUID NOT NULL,
			author_id UUID NOT NULL,
			title VARCHAR(200) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			format VARCHAR(16) NOT NULL DEFAULT 'markdown',
			current_version_id UUID NULL,
			latest_draft_updated_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ NULL
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу documents", zap.Error(err))
		return err
	}

	// document_versions
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS document_versions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			author_id UUID NOT NULL,
			version_number INTEGER NOT NULL,
			content TEXT NOT NULL,
			change_summary VARCHAR(255) NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу document_versions", zap.Error(err))
		return err
	}

	// document_drafts
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS document_drafts (
			document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
			content TEXT NOT NULL DEFAULT '',
			updated_by UUID NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу document_drafts", zap.Error(err))
		return err
	}

	// files
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			repo_id UUID NOT NULL,
			uploaded_by UUID NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			current_version_id UUID NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ NULL
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу files", zap.Error(err))
		return err
	}

	// file_versions
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS file_versions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			uploaded_by UUID NOT NULL,
			version_number INTEGER NOT NULL,
			storage_key TEXT NOT NULL,
			mime_type VARCHAR(255) NOT NULL,
			size_bytes BIGINT NOT NULL,
			checksum_sha256 VARCHAR(64) NULL,
			change_summary VARCHAR(255) NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		log.Error("Не удалось создать таблицу file_versions", zap.Error(err))
		return err
	}

	log.Info("Таблицы успешно созданы")

	// CHECK-ограничения
	log.Info("Создание CHECK-ограничений")
	checks := []struct {
		table string
		name  string
		sql   string
	}{
		{
			"documents",
			"chk_documents_title_len",
			`ALTER TABLE documents ADD CONSTRAINT chk_documents_title_len
				CHECK (char_length(title) BETWEEN 1 AND 200)`,
		},
		{
			"documents",
			"chk_documents_slug_len",
			`ALTER TABLE documents ADD CONSTRAINT chk_documents_slug_len
				CHECK (char_length(slug) BETWEEN 1 AND 100)`,
		},
		{
			"documents",
			"chk_documents_slug_format",
			`ALTER TABLE documents ADD CONSTRAINT chk_documents_slug_format
				CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{0,98}[a-z0-9])?$')`,
		},
		{
			"documents",
			"chk_documents_format",
			`ALTER TABLE documents ADD CONSTRAINT chk_documents_format
				CHECK (format IN ('markdown'))`,
		},
		{
			"document_versions",
			"chk_document_versions_version_number",
			`ALTER TABLE document_versions ADD CONSTRAINT chk_document_versions_version_number
				CHECK (version_number >= 1)`,
		},
		{
			"document_versions",
			"chk_document_versions_change_summary_len",
			`ALTER TABLE document_versions ADD CONSTRAINT chk_document_versions_change_summary_len
				CHECK (change_summary IS NULL OR char_length(change_summary) <= 255)`,
		},
		{
			"files",
			"chk_files_file_name_len",
			`ALTER TABLE files ADD CONSTRAINT chk_files_file_name_len
				CHECK (char_length(file_name) BETWEEN 1 AND 255)`,
		},
		{
			"file_versions",
			"chk_file_versions_version_number",
			`ALTER TABLE file_versions ADD CONSTRAINT chk_file_versions_version_number
				CHECK (version_number >= 1)`,
		},
		{
			"file_versions",
			"chk_file_versions_size_bytes",
			`ALTER TABLE file_versions ADD CONSTRAINT chk_file_versions_size_bytes
				CHECK (size_bytes >= 0)`,
		},
		{
			"file_versions",
			"chk_file_versions_checksum_len",
			`ALTER TABLE file_versions ADD CONSTRAINT chk_file_versions_checksum_len
				CHECK (checksum_sha256 IS NULL OR char_length(checksum_sha256) = 64)`,
		},
		{
			"file_versions",
			"chk_file_versions_change_summary_len",
			`ALTER TABLE file_versions ADD CONSTRAINT chk_file_versions_change_summary_len
				CHECK (change_summary IS NULL OR char_length(change_summary) <= 255)`,
		},
	}
	for _, c := range checks {
		if err := addConstraintIfNotExists(db, c.table, c.name, c.sql); err != nil {
			log.Error("Не удалось создать ограничение", zap.String("constraint", c.name), zap.Error(err))
			return err
		}
	}
	log.Info("CHECK-ограничения успешно созданы")

	// Частичные индексы
	log.Info("Создание частичных индексов")
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_documents_repo_slug_active
			ON documents(repo_id, slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_documents_repo_id_active
			ON documents(repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_documents_author_id_active
			ON documents(author_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_documents_created_at_active
			ON documents(created_at DESC) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_document_versions_document_version
			ON document_versions(document_id, version_number)`,
		`CREATE INDEX IF NOT EXISTS idx_document_versions_document_id_created_at
			ON document_versions(document_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_files_repo_id_active
			ON files(repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_files_uploaded_by_active
			ON files(uploaded_by) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_files_created_at_active
			ON files(created_at DESC) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_file_versions_file_version
			ON file_versions(file_id, version_number)`,
		`CREATE INDEX IF NOT EXISTS idx_file_versions_file_id_created_at
			ON file_versions(file_id, created_at DESC)`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Error("Не удалось создать индекс", zap.String("sql", idx), zap.Error(err))
			return err
		}
	}
	log.Info("Частичные индексы успешно созданы")

	// Foreign Key для documents.current_version_id
	log.Info("Добавление внешних ключей")
	if err := addForeignKeyIfNotExists(db,
		"documents",
		"fk_documents_current_version",
		`ALTER TABLE documents ADD CONSTRAINT fk_documents_current_version
		FOREIGN KEY (current_version_id) REFERENCES document_versions(id) ON DELETE SET NULL`,
	); err != nil {
		log.Warn("Не удалось добавить FK fk_documents_current_version", zap.Error(err))
	}

	if err := addForeignKeyIfNotExists(db,
		"files",
		"fk_files_current_version",
		`ALTER TABLE files ADD CONSTRAINT fk_files_current_version
		FOREIGN KEY (current_version_id) REFERENCES file_versions(id) ON DELETE SET NULL`,
	); err != nil {
		log.Warn("Не удалось добавить FK fk_files_current_version", zap.Error(err))
	}

	log.Info("Внешние ключи успешно добавлены")
	log.Info("Миграция базы данных контента успешно завершена")
	return nil
}

// addConstraintIfNotExists добавляет CHECK ограничение, если оно ещё не существует.
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

// addForeignKeyIfNotExists добавляет FK, если оно ещё не существует.
func addForeignKeyIfNotExists(db *gorm.DB, table, fkName, ddl string) error {
	var exists bool
	err := db.Raw(
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'
		)`, table, fkName,
	).Scan(&exists).Error
	if err != nil {
		return err
	}
	if !exists {
		return db.Exec(ddl).Error
	}
	return nil
}
