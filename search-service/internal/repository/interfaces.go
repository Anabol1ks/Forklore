package repository

import (
	"context"
	"search-service/internal/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SearchParams struct {
	Query       string
	EntityTypes []model.SearchEntityType
	TagID       *uuid.UUID
	OwnerID     *uuid.UUID
	RepoID      *uuid.UUID
	Limit       int
	Offset      int
}

type SearchHit struct {
	EntityType  model.SearchEntityType
	EntityID    uuid.UUID
	RepoID      *uuid.UUID
	OwnerID     *uuid.UUID
	TagID       *uuid.UUID
	Title       string
	Description *string
	Snippet     *string
	Rank        float64
	UpdatedAt   time.Time
}

type RepositoryMetadataPatch struct {
	RepoID   uuid.UUID
	OwnerID  uuid.UUID
	TagID    uuid.UUID
	TagName  string
	IsPublic bool
}

type SearchIndexRepository interface {
	Search(ctx context.Context, params SearchParams) ([]*SearchHit, int64, error)

	UpsertRepository(ctx context.Context, item *model.SearchIndexItem) error
	UpsertDocument(ctx context.Context, item *model.SearchIndexItem) error
	UpsertFile(ctx context.Context, item *model.SearchIndexItem) error

	DeleteByEntity(ctx context.Context, entityType model.SearchEntityType, entityID uuid.UUID) error
	DeleteByRepoID(ctx context.Context, repoID uuid.UUID) error

	PropagateRepositoryMetadata(ctx context.Context, patch RepositoryMetadataPatch) error
}

type Repository struct {
	db     *gorm.DB
	Search SearchIndexRepository
}
