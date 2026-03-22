package kafka

import (
	"context"
	"time"

	"ranking-service/internal/service"

	rankingeventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/events/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type RankingHandler struct {
	svc    service.Service
	logger *zap.Logger
}

func NewRankingHandler(svc service.Service, logger *zap.Logger) *RankingHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RankingHandler{svc: svc, logger: logger}
}

func (h *RankingHandler) HandleMessage(ctx context.Context, payload []byte) error {
	var envelope rankingeventsv1.RankingEventEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return err
	}

	occurredAt := envelope.GetOccurredAt().AsTime()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	switch event := envelope.Payload.(type) {
	case *rankingeventsv1.RankingEventEnvelope_UserFollowed:
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "user.followed", UserID: mustParseUUID(event.UserFollowed.GetUserId().GetValue()), Delta: nonZeroDelta(event.UserFollowed.GetDelta()), OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_UserUnfollowed:
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "user.unfollowed", UserID: mustParseUUID(event.UserUnfollowed.GetUserId().GetValue()), Delta: nonZeroDelta(event.UserUnfollowed.GetDelta()), OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_RepositoryCreated:
		tagID := mustParseUUID(event.RepositoryCreated.GetTagId().GetValue())
		repoID := mustParseUUID(event.RepositoryCreated.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "repo.created", OwnerID: mustParseUUID(event.RepositoryCreated.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, IsPublic: event.RepositoryCreated.GetIsPublic(), Points: 5, OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_RepositoryVisibilityChanged:
		tagID := mustParseUUID(event.RepositoryVisibilityChanged.GetTagId().GetValue())
		repoID := mustParseUUID(event.RepositoryVisibilityChanged.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "repo.visibility.changed", OwnerID: mustParseUUID(event.RepositoryVisibilityChanged.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Delta: nonZeroDelta(event.RepositoryVisibilityChanged.GetDelta()), IsPublic: event.RepositoryVisibilityChanged.GetIsPublic(), OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_RepositoryStarred:
		tagID := mustParseUUID(event.RepositoryStarred.GetTagId().GetValue())
		repoID := mustParseUUID(event.RepositoryStarred.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "repo.starred", OwnerID: mustParseUUID(event.RepositoryStarred.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Delta: nonZeroDelta(event.RepositoryStarred.GetDelta()), OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_RepositoryUnstarred:
		tagID := mustParseUUID(event.RepositoryUnstarred.GetTagId().GetValue())
		repoID := mustParseUUID(event.RepositoryUnstarred.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "repo.unstarred", OwnerID: mustParseUUID(event.RepositoryUnstarred.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Delta: nonZeroDelta(event.RepositoryUnstarred.GetDelta()), OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_RepositoryForked:
		tagID := mustParseUUID(event.RepositoryForked.GetTagId().GetValue())
		repoID := mustParseUUID(event.RepositoryForked.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "repo.forked", OwnerID: mustParseUUID(event.RepositoryForked.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Delta: nonZeroDelta(event.RepositoryForked.GetDelta()), OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_DocumentCreated:
		tagID := mustParseUUID(event.DocumentCreated.GetTagId().GetValue())
		repoID := mustParseUUID(event.DocumentCreated.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "document.created", OwnerID: mustParseUUID(event.DocumentCreated.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Points: 4, OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_DocumentVersionCreated:
		tagID := mustParseUUID(event.DocumentVersionCreated.GetTagId().GetValue())
		repoID := mustParseUUID(event.DocumentVersionCreated.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "document.version.created", OwnerID: mustParseUUID(event.DocumentVersionCreated.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Points: 2, OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_FileCreated:
		tagID := mustParseUUID(event.FileCreated.GetTagId().GetValue())
		repoID := mustParseUUID(event.FileCreated.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "file.created", OwnerID: mustParseUUID(event.FileCreated.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Points: 2, OccurredAt: occurredAt})
	case *rankingeventsv1.RankingEventEnvelope_FileVersionCreated:
		tagID := mustParseUUID(event.FileVersionCreated.GetTagId().GetValue())
		repoID := mustParseUUID(event.FileVersionCreated.GetRepoId().GetValue())
		return h.svc.ApplyEvent(ctx, service.UserEvent{Type: "file.version.created", OwnerID: mustParseUUID(event.FileVersionCreated.GetOwnerId().GetValue()), RepoID: &repoID, TagID: &tagID, Points: 1, OccurredAt: occurredAt})
	default:
		h.logger.Debug("unknown ranking kafka event payload", zap.String("event_type", envelope.GetEventType().String()))
		return nil
	}
}

func nonZeroDelta(value int64) int64 {
	if value == 0 {
		return 1
	}
	return value
}
