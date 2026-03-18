package kafka

import (
	"context"
	"search-service/internal/service"
	"strings"

	searcheventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/events/v1"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Handler struct {
	service service.SearchService
	logger  *zap.Logger
}

func NewHandler(service service.SearchService, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, payload []byte) error {
	var envelope searcheventsv1.SearchEventEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return err
	}

	switch event := envelope.Payload.(type) {
	case *searcheventsv1.SearchEventEnvelope_RepositoryUpserted:
		return h.service.ApplyRepositoryUpsert(ctx, service.RepositoryUpsertPayload{
			RepoID:      mustParseUUID(event.RepositoryUpserted.GetRepoId().GetValue()),
			OwnerID:     mustParseUUID(event.RepositoryUpserted.GetOwnerId().GetValue()),
			TagID:       mustParseUUID(event.RepositoryUpserted.GetTagId().GetValue()),
			Title:       event.RepositoryUpserted.GetTitle(),
			Description: event.RepositoryUpserted.GetDescription(),
			TagName:     event.RepositoryUpserted.GetTagName(),
			IsPublic:    event.RepositoryUpserted.GetIsPublic(),
			UpdatedAt:   event.RepositoryUpserted.GetUpdatedAt().AsTime(),
		})

	case *searcheventsv1.SearchEventEnvelope_RepositoryDeleted:
		return h.service.ApplyRepositoryDeleted(ctx, service.RepositoryDeletedPayload{
			RepoID: mustParseUUID(event.RepositoryDeleted.GetRepoId().GetValue()),
		})

	case *searcheventsv1.SearchEventEnvelope_DocumentUpserted:
		return h.service.ApplyDocumentUpsert(ctx, service.DocumentUpsertPayload{
			DocumentID: mustParseUUID(event.DocumentUpserted.GetDocumentId().GetValue()),
			RepoID:     mustParseUUID(event.DocumentUpserted.GetRepoId().GetValue()),
			OwnerID:    mustParseUUID(event.DocumentUpserted.GetOwnerId().GetValue()),
			TagID:      mustParseUUID(event.DocumentUpserted.GetTagId().GetValue()),
			Title:      event.DocumentUpserted.GetTitle(),
			Content:    event.DocumentUpserted.GetContent(),
			TagName:    event.DocumentUpserted.GetTagName(),
			IsPublic:   event.DocumentUpserted.GetIsPublic(),
			UpdatedAt:  event.DocumentUpserted.GetUpdatedAt().AsTime(),
		})

	case *searcheventsv1.SearchEventEnvelope_DocumentDeleted:
		return h.service.ApplyDocumentDeleted(ctx, service.DocumentDeletedPayload{
			DocumentID: mustParseUUID(event.DocumentDeleted.GetDocumentId().GetValue()),
		})

	case *searcheventsv1.SearchEventEnvelope_FileUpserted:
		return h.service.ApplyFileUpsert(ctx, service.FileUpsertPayload{
			FileID:    mustParseUUID(event.FileUpserted.GetFileId().GetValue()),
			RepoID:    mustParseUUID(event.FileUpserted.GetRepoId().GetValue()),
			OwnerID:   mustParseUUID(event.FileUpserted.GetOwnerId().GetValue()),
			TagID:     mustParseUUID(event.FileUpserted.GetTagId().GetValue()),
			FileName:  event.FileUpserted.GetFileName(),
			MimeType:  event.FileUpserted.GetMimeType(),
			TagName:   event.FileUpserted.GetTagName(),
			IsPublic:  event.FileUpserted.GetIsPublic(),
			UpdatedAt: event.FileUpserted.GetUpdatedAt().AsTime(),
		})

	case *searcheventsv1.SearchEventEnvelope_FileDeleted:
		return h.service.ApplyFileDeleted(ctx, service.FileDeletedPayload{
			FileID: mustParseUUID(event.FileDeleted.GetFileId().GetValue()),
		})

	default:
		h.logger.Warn("unknown kafka search event payload")
		return nil
	}
}

func mustParseUUID(value string) uuid.UUID {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}

	return id
}
