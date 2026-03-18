package service

import (
	"context"
	"repository-service/internal/model"

	"github.com/google/uuid"
)

type SearchEventPublisher interface {
	PublishRepositoryUpserted(ctx context.Context, repo *model.Repository) error
	PublishRepositoryDeleted(ctx context.Context, repoID uuid.UUID) error
}

type noopSearchEventPublisher struct{}

func NewNoopSearchEventPublisher() SearchEventPublisher {
	return noopSearchEventPublisher{}
}

func (noopSearchEventPublisher) PublishRepositoryUpserted(context.Context, *model.Repository) error {
	return nil
}

func (noopSearchEventPublisher) PublishRepositoryDeleted(context.Context, uuid.UUID) error {
	return nil
}
