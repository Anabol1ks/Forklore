package service

import (
	"context"
	"repository-service/internal/model"
	"time"

	"github.com/google/uuid"
)

type Pagination struct {
	Limit  int
	Offset int
}

type RepositoryService interface {
	CreateRepository(ctx context.Context, input CreateRepositoryInput) (*model.Repository, error)
	GetRepositoryByID(ctx context.Context, requesterID uuid.UUID, repoID uuid.UUID) (*model.Repository, error)
	GetRepositoryBySlug(ctx context.Context, requesterID uuid.UUID, requesterUsername string, ownerKey string, slug string) (*model.Repository, error)

	UpdateRepository(ctx context.Context, input UpdateRepositoryInput) (*model.Repository, error)
	DeleteRepository(ctx context.Context, requesterID uuid.UUID, repoID uuid.UUID) error

	ForkRepository(ctx context.Context, input ForkRepositoryInput) (*model.Repository, error)

	ListMyRepositories(ctx context.Context, ownerID uuid.UUID, pagination Pagination) ([]*model.Repository, int64, error)
	ListUserRepositories(ctx context.Context, requesterID uuid.UUID, requesterUsername string, ownerKey string, pagination Pagination) ([]*model.Repository, int64, error)
	ListForks(ctx context.Context, requesterID uuid.UUID, repoID uuid.UUID, pagination Pagination) ([]*model.Repository, int64, error)
	ReindexSearchIndex(ctx context.Context, batchSize int) error

	ListRepositoryTags(ctx context.Context) ([]*model.RepositoryTag, error)
}

type CreateRepositoryInput struct {
	OwnerID       uuid.UUID
	OwnerUsername string
	TagID         uuid.UUID
	Name          string
	Slug          string
	Description   string
	Visibility    model.RepositoryVisibility
	Type          model.RepositoryType
}

type UpdateRepositoryInput struct {
	RequesterID uuid.UUID
	RepoID      uuid.UUID
	TagID       *uuid.UUID
	Name        string
	Slug        string
	Description string
	Visibility  model.RepositoryVisibility
	Type        model.RepositoryType
}

type ForkRepositoryInput struct {
	RequesterID       uuid.UUID
	RequesterUsername string
	SourceRepoID      uuid.UUID
	Name              string
	Slug              string
	Description       string
	Visibility        model.RepositoryVisibility
}

type RepositoryTagOutput struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description *string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
