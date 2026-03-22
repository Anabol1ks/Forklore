package service

import (
	"context"
	"repository-service/internal/model"
)

type RankingEventPublisher interface {
	PublishRepositoryCreated(ctx context.Context, repo *model.Repository) error
	PublishRepositoryVisibilityChanged(ctx context.Context, repo *model.Repository, delta int64) error
	PublishRepositoryStarred(ctx context.Context, repo *model.Repository, delta int64) error
	PublishRepositoryUnstarred(ctx context.Context, repo *model.Repository, delta int64) error
	PublishRepositoryForked(ctx context.Context, sourceRepo *model.Repository, delta int64) error
}

type noopRankingEventPublisher struct{}

func NewNoopRankingEventPublisher() RankingEventPublisher {
	return noopRankingEventPublisher{}
}

func (noopRankingEventPublisher) PublishRepositoryCreated(context.Context, *model.Repository) error {
	return nil
}

func (noopRankingEventPublisher) PublishRepositoryVisibilityChanged(context.Context, *model.Repository, int64) error {
	return nil
}

func (noopRankingEventPublisher) PublishRepositoryStarred(context.Context, *model.Repository, int64) error {
	return nil
}

func (noopRankingEventPublisher) PublishRepositoryUnstarred(context.Context, *model.Repository, int64) error {
	return nil
}

func (noopRankingEventPublisher) PublishRepositoryForked(context.Context, *model.Repository, int64) error {
	return nil
}
