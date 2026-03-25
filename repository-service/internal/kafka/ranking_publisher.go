package kafka

import (
	"context"
	"repository-service/internal/model"
	"time"

	rankingeventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/events/v1"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RankingProducerConfig struct {
	Brokers []string
	Topic   string
}

type RankingProducer struct {
	writer *kafka.Writer
}

func NewRankingProducer(cfg RankingProducerConfig) *RankingProducer {
	return &RankingProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
			BatchTimeout: 50 * time.Millisecond,
		},
	}
}

func (p *RankingProducer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func (p *RankingProducer) PublishRepositoryCreated(ctx context.Context, repo *model.Repository) error {
	if repo == nil {
		return nil
	}
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_REPOSITORY_CREATED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &rankingeventsv1.RankingEventEnvelope_RepositoryCreated{
			RepositoryCreated: &rankingeventsv1.RepositoryCreated{
				OwnerId:  toProtoUUID(repo.OwnerID),
				RepoId:   toProtoUUID(repo.ID),
				TagId:    toProtoUUID(repo.TagID),
				IsPublic: repo.Visibility == model.RepositoryVisibilityPublic,
			},
		},
	}
	return p.publish(ctx, repo.ID.String(), envelope)
}

func (p *RankingProducer) PublishRepositoryVisibilityChanged(ctx context.Context, repo *model.Repository, delta int64) error {
	if repo == nil {
		return nil
	}
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_REPOSITORY_VISIBILITY_CHANGED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &rankingeventsv1.RankingEventEnvelope_RepositoryVisibilityChanged{
			RepositoryVisibilityChanged: &rankingeventsv1.RepositoryVisibilityChanged{
				OwnerId:  toProtoUUID(repo.OwnerID),
				RepoId:   toProtoUUID(repo.ID),
				TagId:    toProtoUUID(repo.TagID),
				IsPublic: repo.Visibility == model.RepositoryVisibilityPublic,
				Delta:    nonZeroDelta(delta),
			},
		},
	}
	return p.publish(ctx, repo.ID.String(), envelope)
}

func (p *RankingProducer) PublishRepositoryStarred(ctx context.Context, repo *model.Repository, delta int64) error {
	if repo == nil {
		return nil
	}
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_REPOSITORY_STARRED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &rankingeventsv1.RankingEventEnvelope_RepositoryStarred{
			RepositoryStarred: &rankingeventsv1.RepositoryStarred{
				OwnerId: toProtoUUID(repo.OwnerID),
				RepoId:  toProtoUUID(repo.ID),
				TagId:   toProtoUUID(repo.TagID),
				Delta:   nonZeroDelta(delta),
			},
		},
	}
	return p.publish(ctx, repo.ID.String(), envelope)
}

func (p *RankingProducer) PublishRepositoryUnstarred(ctx context.Context, repo *model.Repository, delta int64) error {
	if repo == nil {
		return nil
	}
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_REPOSITORY_UNSTARRED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &rankingeventsv1.RankingEventEnvelope_RepositoryUnstarred{
			RepositoryUnstarred: &rankingeventsv1.RepositoryUnstarred{
				OwnerId: toProtoUUID(repo.OwnerID),
				RepoId:  toProtoUUID(repo.ID),
				TagId:   toProtoUUID(repo.TagID),
				Delta:   nonZeroDelta(delta),
			},
		},
	}
	return p.publish(ctx, repo.ID.String(), envelope)
}

func (p *RankingProducer) PublishRepositoryForked(ctx context.Context, sourceRepo *model.Repository, delta int64) error {
	if sourceRepo == nil {
		return nil
	}
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_REPOSITORY_FORKED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &rankingeventsv1.RankingEventEnvelope_RepositoryForked{
			RepositoryForked: &rankingeventsv1.RepositoryForked{
				OwnerId: toProtoUUID(sourceRepo.OwnerID),
				RepoId:  toProtoUUID(sourceRepo.ID),
				TagId:   toProtoUUID(sourceRepo.TagID),
				Delta:   nonZeroDelta(delta),
			},
		},
	}
	return p.publish(ctx, sourceRepo.ID.String(), envelope)
}

func (p *RankingProducer) publish(ctx context.Context, key string, envelope *rankingeventsv1.RankingEventEnvelope) error {
	payload, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	})
}

func nonZeroDelta(value int64) int64 {
	if value == 0 {
		return 1
	}
	return value
}
