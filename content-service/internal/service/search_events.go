package service

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
)

type RepositoryMetadata struct {
	RepoID   uuid.UUID
	OwnerID  uuid.UUID
	TagID    uuid.UUID
	TagName  string
	IsPublic bool
}

type SearchEventPublisher interface {
	PublishDocumentUpserted(ctx context.Context, document *model.Document, content string, metadata *RepositoryMetadata) error
	PublishDocumentDeleted(ctx context.Context, documentID uuid.UUID) error
	PublishFileUpserted(ctx context.Context, file *model.File, mimeType string, metadata *RepositoryMetadata) error
	PublishFileDeleted(ctx context.Context, fileID uuid.UUID) error
}

type noopSearchEventPublisher struct{}

func NewNoopSearchEventPublisher() SearchEventPublisher {
	return noopSearchEventPublisher{}
}

func (noopSearchEventPublisher) PublishDocumentUpserted(context.Context, *model.Document, string, *RepositoryMetadata) error {
	return nil
}

func (noopSearchEventPublisher) PublishDocumentDeleted(context.Context, uuid.UUID) error {
	return nil
}

func (noopSearchEventPublisher) PublishFileUpserted(context.Context, *model.File, string, *RepositoryMetadata) error {
	return nil
}

func (noopSearchEventPublisher) PublishFileDeleted(context.Context, uuid.UUID) error {
	return nil
}
