package migrations

import (
	"context"
	"fmt"
	"repository-service/internal/model"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	modelsAny := []any{
		&model.RepositoryTag{},
		&model.Repository{},
		&model.RepositoryStar{},
	}
	if err := db.AutoMigrate(modelsAny...); err != nil {
		log.Error("Не удалось создать базовые таблицы", zap.Error(err))
		return err
	}
	log.Info("Базовые таблицы успешно созданы")

	// CHECK-ограничения
	log.Info("Создание CHECK-ограничений")
	checks := []struct {
		table string
		name  string
		sql   string
	}{
		{
			"repository_tags",
			"chk_repository_tags_name_len",
			`ALTER TABLE repository_tags ADD CONSTRAINT chk_repository_tags_name_len
				CHECK (char_length(name) BETWEEN 2 AND 64)`,
		},
		{
			"repository_tags",
			"chk_repository_tags_slug_len",
			`ALTER TABLE repository_tags ADD CONSTRAINT chk_repository_tags_slug_len
				CHECK (char_length(slug) BETWEEN 2 AND 64)`,
		},
		{
			"repository_tags",
			"chk_repository_tags_slug_format",
			`ALTER TABLE repository_tags ADD CONSTRAINT chk_repository_tags_slug_format
				CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$')`,
		},
		{
			"repositories",
			"chk_repositories_name_len",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_name_len
				CHECK (char_length(name) BETWEEN 3 AND 100)`,
		},
		{
			"repositories",
			"chk_repositories_slug_len",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_slug_len
				CHECK (char_length(slug) BETWEEN 3 AND 64)`,
		},
		{
			"repositories",
			"chk_repositories_slug_format",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_slug_format
				CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{1,62}[a-z0-9])?$')`,
		},
		{
			"repositories",
			"chk_repositories_visibility",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_visibility
				CHECK (visibility IN ('public', 'private'))`,
		},
		{
			"repositories",
			"chk_repositories_type",
			`ALTER TABLE repositories ADD CONSTRAINT chk_repositories_type
				CHECK (type IN ('article', 'notes', 'mixed'))`,
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
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_repositories_owner_slug_active
			ON repositories(owner_id, slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_owner_id_active
			ON repositories(owner_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_owner_username_active
			ON repositories(owner_username) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_tag_id_active
			ON repositories(tag_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_parent_repo_id_active
			ON repositories(parent_repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_visibility_active
			ON repositories(visibility) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_created_at_active
			ON repositories(created_at DESC) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_repository_stars_user_repo
			ON repository_stars(user_id, repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_repository_stars_user_created_at
			ON repository_stars(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_repository_stars_repo_id
			ON repository_stars(repo_id)`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Error("Не удалось создать индекс", zap.String("sql", idx), zap.Error(err))
			return err
		}
	}
	log.Info("Частичные индексы успешно созданы")

	if err := seedDefaultTags(db, log); err != nil {
		return err
	}

	if err := seedUniversitySubjectTags(db, log); err != nil {
		return err
	}

	return nil
}

func seedDefaultTags(db *gorm.DB, log *zap.Logger) error {
	defaultTags := []*model.RepositoryTag{
		{Name: "Математика", Slug: "mathematics", Description: stringPtr("Материалы по математике"), IsActive: true},
		{Name: "Физика", Slug: "physics", Description: stringPtr("Материалы по физике"), IsActive: true},
		{Name: "Программирование", Slug: "programming", Description: stringPtr("Материалы по программированию"), IsActive: true},
		{Name: "Алгоритмы", Slug: "algorithms", Description: stringPtr("Алгоритмы и структуры данных"), IsActive: true},
		{Name: "Базы данных", Slug: "databases", Description: stringPtr("Материалы по базам данных"), IsActive: true},
		{Name: "Машинное обучение", Slug: "machine-learning", Description: stringPtr("ML и AI"), IsActive: true},
		{Name: "Сети", Slug: "networks", Description: stringPtr("Компьютерные сети"), IsActive: true},
		{Name: "Операционные системы", Slug: "operating-systems", Description: stringPtr("ОС и системное ПО"), IsActive: true},
		{Name: "Экономика", Slug: "economics", Description: stringPtr("Материалы по экономике"), IsActive: true},
		{Name: "Право", Slug: "law", Description: stringPtr("Материалы по праву"), IsActive: true},
	}

	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slug"}},
		DoNothing: true,
	}).Create(&defaultTags).Error; err != nil {
		log.Error("Не удалось вставить дефолтные теги", zap.Error(err))
		return err
	}

	return nil
}

func seedUniversitySubjectTags(db *gorm.DB, log *zap.Logger) error {
	mireaSubjects := []string{
		"Высшая математика", "Линейная алгебра", "Аналитическая геометрия", "Дискретная математика", "Теория вероятностей",
		"Математическая статистика", "Физика", "Теоретическая механика", "Электротехника", "Электроника",
		"Цифровая схемотехника", "Архитектура ЭВМ", "Операционные системы", "Компьютерные сети", "Системное программирование",
		"Программирование на C", "Программирование на C++", "Программирование на Java", "Программирование на Python", "Веб-программирование",
		"Алгоритмы и структуры данных", "Базы данных", "Проектирование БД", "Инженерия ПО", "Тестирование ПО",
		"DevOps", "Контейнеризация", "Микросервисы", "Информационная безопасность", "Криптография",
		"Защита информации", "Машинное обучение", "Нейронные сети", "Компьютерное зрение", "Обработка сигналов",
		"Цифровая обработка изображений", "Робототехника", "Микроконтроллеры", "Встраиваемые системы", "Телекоммуникации",
		"Радиотехника", "Теория автоматического управления", "Моделирование систем", "САПР", "Экономика ИТ",
		"Менеджмент проектов", "Английский для ИТ", "Право в ИТ", "Научно-исследовательская работа", "Подготовка ВКР",
	}

	mguSubjects := []string{
		"Математический анализ", "Линейная алгебра", "Дифференциальные уравнения", "Функциональный анализ", "Теория чисел",
		"Топология", "Дифференциальная геометрия", "Теория вероятностей", "Математическая статистика", "Математическая логика",
		"Алгебра", "Комплексный анализ", "Уравнения матфизики", "Численные методы", "Оптимизация",
		"Физика", "Квантовая механика", "Электродинамика", "Термодинамика", "Статистическая физика",
		"Астрономия", "Методы вычислений", "Информатика", "Алгоритмы", "Структуры данных",
		"Программирование на Python", "Программирование на C++", "Системы управления БД", "Машинное обучение", "Искусственный интеллект",
		"Компьютерная лингвистика", "Биоинформатика", "Эконометрика", "Теория игр", "Макроэкономика",
		"Микроэкономика", "Финансовая математика", "Право", "Конституционное право", "Международное право",
		"История", "Философия", "Социология", "Политология", "Психология",
		"Академическое письмо", "Английский язык", "Немецкий язык", "Научный семинар", "Подготовка ВКР",
	}

	tags := make([]*model.RepositoryTag, 0, 100)
	tags = append(tags, buildUniversityTags("МИРЭА", "mirea", mireaSubjects)...)
	tags = append(tags, buildUniversityTags("МГУ", "msu", mguSubjects)...)

	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slug"}},
		DoNothing: true,
	}).Create(&tags).Error; err != nil {
		log.Error("Не удалось вставить университетские предметные теги", zap.Error(err))
		return err
	}

	log.Info("Университетские предметные теги готовы", zap.Int("count", len(tags)))
	return nil
}

func buildUniversityTags(universityName, universitySlug string, subjects []string) []*model.RepositoryTag {
	tags := make([]*model.RepositoryTag, 0, len(subjects))
	for index, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}

		name := fmt.Sprintf("%s • %s", universityName, subject)
		slug := fmt.Sprintf("%s-subject-%02d", universitySlug, index+1)
		description := fmt.Sprintf("%s: %s", universityName, subject)

		tags = append(tags, &model.RepositoryTag{
			Name:        name,
			Slug:        slug,
			Description: stringPtr(description),
			IsActive:    true,
		})
	}

	return tags
}

func stringPtr(v string) *string {
	return &v
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
