package service

import (
	"context"
	"search-service/internal/model"
	"time"

	"github.com/google/uuid"
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

type RepositoryUpsertPayload struct {
	RepoID      uuid.UUID
	OwnerID     uuid.UUID
	TagID       uuid.UUID
	Title       string
	Description string
	TagName     string
	IsPublic    bool
	UpdatedAt   time.Time
}

type RepositoryDeletedPayload struct {
	RepoID uuid.UUID
}

type DocumentUpsertPayload struct {
	DocumentID uuid.UUID
	RepoID     uuid.UUID
	OwnerID    uuid.UUID
	TagID      uuid.UUID
	Title      string
	Content    string
	TagName    string
	IsPublic   bool
	UpdatedAt  time.Time
}

type DocumentDeletedPayload struct {
	DocumentID uuid.UUID
}

type FileUpsertPayload struct {
	FileID    uuid.UUID
	RepoID    uuid.UUID
	OwnerID   uuid.UUID
	TagID     uuid.UUID
	FileName  string
	MimeType  string
	TagName   string
	IsPublic  bool
	UpdatedAt time.Time
}

type FileDeletedPayload struct {
	FileID uuid.UUID
}

type SearchService interface {
	Search(ctx context.Context, params SearchParams) ([]*SearchHit, int64, error)

	ApplyRepositoryUpsert(ctx context.Context, payload RepositoryUpsertPayload) error
	ApplyRepositoryDeleted(ctx context.Context, payload RepositoryDeletedPayload) error

	ApplyDocumentUpsert(ctx context.Context, payload DocumentUpsertPayload) error
	ApplyDocumentDeleted(ctx context.Context, payload DocumentDeletedPayload) error

	ApplyFileUpsert(ctx context.Context, payload FileUpsertPayload) error
	ApplyFileDeleted(ctx context.Context, payload FileDeletedPayload) error
}
