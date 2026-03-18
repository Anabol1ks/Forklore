package kafka

import (
	"context"
	"repository-service/internal/model"
	"strings"
	"time"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	searcheventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/events/v1"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProducerConfig struct {
	Brokers []string
	Topic   string
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg ProducerConfig) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
			BatchTimeout: 50 * time.Millisecond,
		},
	}
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func (p *Producer) PublishRepositoryUpserted(ctx context.Context, repo *model.Repository) error {
	if repo == nil {
		return nil
	}

	tagName := ""
	if repo.Tag != nil {
		tagName = repo.Tag.Name
	}

	ownerUsername := strings.TrimSpace(repo.OwnerUsername)
	repoSlug := strings.TrimSpace(repo.Slug)
	searchTitle := repo.Name
	if ownerUsername != "" && repoSlug != "" {
		searchTitle = ownerUsername + "/" + repoSlug
	}

	searchDescription := strings.TrimSpace(derefString(repo.Description))
	if repo.Name != "" {
		if searchDescription != "" {
			searchDescription = repo.Name + "\n" + searchDescription
		} else {
			searchDescription = repo.Name
		}
	}
	if repoSlug != "" {
		searchDescription = strings.TrimSpace(searchDescription + "\nslug:" + repoSlug)
	}

	envelope := &searcheventsv1.SearchEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  commonv1.SearchEventType_SEARCH_EVENT_TYPE_REPOSITORY_UPSERTED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &searcheventsv1.SearchEventEnvelope_RepositoryUpserted{
			RepositoryUpserted: &searcheventsv1.RepositoryUpserted{
				RepoId:      toProtoUUID(repo.ID),
				OwnerId:     toProtoUUID(repo.OwnerID),
				TagId:       toProtoUUID(repo.TagID),
				Title:       searchTitle,
				Description: searchDescription,
				TagName:     tagName,
				IsPublic:    repo.Visibility == model.RepositoryVisibilityPublic,
				UpdatedAt:   timestamppb.New(repo.UpdatedAt.UTC()),
			},
		},
	}

	return p.publish(ctx, repo.ID.String(), envelope)
}

func (p *Producer) PublishRepositoryDeleted(ctx context.Context, repoID uuid.UUID) error {
	envelope := &searcheventsv1.SearchEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  commonv1.SearchEventType_SEARCH_EVENT_TYPE_REPOSITORY_DELETED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &searcheventsv1.SearchEventEnvelope_RepositoryDeleted{
			RepositoryDeleted: &searcheventsv1.RepositoryDeleted{
				RepoId: toProtoUUID(repoID),
			},
		},
	}

	return p.publish(ctx, repoID.String(), envelope)
}

func (p *Producer) publish(ctx context.Context, key string, envelope *searcheventsv1.SearchEventEnvelope) error {
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

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{Value: id.String()}
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
