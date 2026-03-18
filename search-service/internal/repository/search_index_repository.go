package repository

import (
	"context"
	"fmt"
	"search-service/internal/model"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type searchIndexRepository struct {
	db *gorm.DB
}

func NewSearchIndexRepository(db *gorm.DB) SearchIndexRepository {
	return &searchIndexRepository{db: db}
}

func (r *searchIndexRepository) UpsertRepository(ctx context.Context, item *model.SearchIndexItem) error {
	return r.upsert(ctx, item)
}

func (r *searchIndexRepository) UpsertDocument(ctx context.Context, item *model.SearchIndexItem) error {
	return r.upsert(ctx, item)
}

func (r *searchIndexRepository) UpsertFile(ctx context.Context, item *model.SearchIndexItem) error {
	return r.upsert(ctx, item)
}

func (r *searchIndexRepository) upsert(ctx context.Context, item *model.SearchIndexItem) error {
	sql := `
INSERT INTO search_index_items (
    entity_type,
    entity_id,
    repo_id,
    owner_id,
    tag_id,
    title,
    description,
    content,
    tag_name,
    mime_type,
    is_public,
    search_vector,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    setweight(to_tsvector('simple', coalesce(?, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(?, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(?, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(?, '')), 'C') ||
    setweight(to_tsvector('simple', coalesce(?, '')), 'D'),
    ?
)
ON CONFLICT (entity_type, entity_id)
DO UPDATE SET
    repo_id = EXCLUDED.repo_id,
    owner_id = EXCLUDED.owner_id,
    tag_id = EXCLUDED.tag_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    content = EXCLUDED.content,
    tag_name = EXCLUDED.tag_name,
    mime_type = EXCLUDED.mime_type,
    is_public = EXCLUDED.is_public,
    search_vector = EXCLUDED.search_vector,
    updated_at = EXCLUDED.updated_at
`

	return r.db.WithContext(ctx).Exec(
		sql,
		item.EntityType,
		item.EntityID,
		item.RepoID,
		item.OwnerID,
		item.TagID,
		item.Title,
		item.Description,
		item.Content,
		item.TagName,
		item.MimeType,
		item.IsPublic,

		item.Title,
		item.TagName,
		item.Description,
		item.Content,
		item.MimeType,

		item.UpdatedAt,
	).Error
}

func (r *searchIndexRepository) DeleteByEntity(ctx context.Context, entityType model.SearchEntityType, entityID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Exec(
			`DELETE FROM search_index_items WHERE entity_type = ? AND entity_id = ?`,
			entityType,
			entityID,
		).Error
}

func (r *searchIndexRepository) DeleteByRepoID(ctx context.Context, repoID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Exec(
			`DELETE FROM search_index_items WHERE repo_id = ?`,
			repoID,
		).Error
}

func (r *searchIndexRepository) PropagateRepositoryMetadata(ctx context.Context, patch RepositoryMetadataPatch) error {
	sql := `
UPDATE search_index_items
SET owner_id = ?,
    tag_id = ?,
    tag_name = ?,
    is_public = ?
WHERE repo_id = ?
  AND entity_type IN ('document', 'file')
`
	return r.db.WithContext(ctx).Exec(
		sql,
		patch.OwnerID,
		patch.TagID,
		patch.TagName,
		patch.IsPublic,
		patch.RepoID,
	).Error
}

func (r *searchIndexRepository) Search(ctx context.Context, params SearchParams) ([]*SearchHit, int64, error) {
	limit, offset := normalizePagination(params.Limit, params.Offset)
	normalizedQuery := strings.TrimSpace(params.Query)
	ilikeQuery := "%" + normalizedQuery + "%"

	baseWhere := []string{"is_public = TRUE"}
	args := make([]any, 0)

	if normalizedQuery != "" {
		baseWhere = append(baseWhere, `(search_vector @@ plainto_tsquery('simple', ?) OR title ILIKE ? OR coalesce(description, '') ILIKE ? OR coalesce(content, '') ILIKE ?)`)
		args = append(args, normalizedQuery, ilikeQuery, ilikeQuery, ilikeQuery)
	}

	if len(params.EntityTypes) > 0 {
		placeholders := make([]string, 0, len(params.EntityTypes))
		for _, t := range params.EntityTypes {
			placeholders = append(placeholders, "?")
			args = append(args, t)
		}
		baseWhere = append(baseWhere, "entity_type IN ("+strings.Join(placeholders, ",")+")")
	}

	if params.TagID != nil {
		baseWhere = append(baseWhere, "tag_id = ?")
		args = append(args, *params.TagID)
	}

	if params.OwnerID != nil {
		baseWhere = append(baseWhere, "owner_id = ?")
		args = append(args, *params.OwnerID)
	}

	if params.RepoID != nil {
		baseWhere = append(baseWhere, "repo_id = ?")
		args = append(args, *params.RepoID)
	}

	whereClause := strings.Join(baseWhere, " AND ")

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM search_index_items WHERE %s`, whereClause)

	var total int64
	if err := r.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	selectSQL := fmt.Sprintf(`
SELECT
    entity_type,
    entity_id,
    repo_id,
    owner_id,
    tag_id,
    title,
    description,
    CASE
        WHEN ? <> '' THEN ts_headline(
            'simple',
            coalesce(content, description, title, ''),
            plainto_tsquery('simple', ?),
            'MaxWords=20, MinWords=8'
        )
		ELSE NULL
    END AS snippet,
	CASE WHEN ? <> '' THEN
		ts_rank(search_vector, plainto_tsquery('simple', ?))
		+ CASE WHEN title ILIKE ? THEN 0.3 ELSE 0 END
		+ CASE WHEN coalesce(description, '') ILIKE ? THEN 0.1 ELSE 0 END
		+ CASE WHEN coalesce(content, '') ILIKE ? THEN 0.05 ELSE 0 END
	ELSE 0 END AS rank,
    updated_at
FROM search_index_items
WHERE %s
ORDER BY rank DESC, updated_at DESC
LIMIT ? OFFSET ?
`, whereClause)

	queryArgs := make([]any, 0, len(args)+9)
	queryArgs = append(queryArgs,
		normalizedQuery,
		normalizedQuery,
		normalizedQuery,
		normalizedQuery,
		ilikeQuery,
		ilikeQuery,
		ilikeQuery,
	)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, limit, offset)

	var hits []*SearchHit
	if err := r.db.WithContext(ctx).Raw(selectSQL, queryArgs...).Scan(&hits).Error; err != nil {
		return nil, 0, err
	}

	return hits, total, nil
}

func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
