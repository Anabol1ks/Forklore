package kafka

import (
	"content-service/internal/model"
	"content-service/internal/service"
	"context"
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

func (p *Producer) PublishDocumentUpserted(ctx context.Context, document *model.Document, content string, metadata *service.RepositoryMetadata) error {
	if document == nil || metadata == nil {
		return nil
	}

	envelope := &searcheventsv1.SearchEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  commonv1.SearchEventType_SEARCH_EVENT_TYPE_DOCUMENT_UPSERTED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &searcheventsv1.SearchEventEnvelope_DocumentUpserted{
			DocumentUpserted: &searcheventsv1.DocumentUpserted{
				DocumentId: toProtoUUID(document.ID),
				RepoId:     toProtoUUID(document.RepoID),
				OwnerId:    toProtoUUID(metadata.OwnerID),
				TagId:      toProtoUUID(metadata.TagID),
				Title:      document.Title,
				Content:    content,
				TagName:    metadata.TagName,
				IsPublic:   metadata.IsPublic,
				UpdatedAt:  timestamppb.New(document.UpdatedAt.UTC()),
			},
		},
	}

	return p.publish(ctx, document.ID.String(), envelope)
}

func (p *Producer) PublishDocumentDeleted(ctx context.Context, documentID uuid.UUID) error {
	envelope := &searcheventsv1.SearchEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  commonv1.SearchEventType_SEARCH_EVENT_TYPE_DOCUMENT_DELETED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &searcheventsv1.SearchEventEnvelope_DocumentDeleted{
			DocumentDeleted: &searcheventsv1.DocumentDeleted{DocumentId: toProtoUUID(documentID)},
		},
	}

	return p.publish(ctx, documentID.String(), envelope)
}

func (p *Producer) PublishFileUpserted(ctx context.Context, file *model.File, mimeType string, metadata *service.RepositoryMetadata) error {
	if file == nil || metadata == nil {
		return nil
	}

	envelope := &searcheventsv1.SearchEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  commonv1.SearchEventType_SEARCH_EVENT_TYPE_FILE_UPSERTED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &searcheventsv1.SearchEventEnvelope_FileUpserted{
			FileUpserted: &searcheventsv1.FileUpserted{
				FileId:    toProtoUUID(file.ID),
				RepoId:    toProtoUUID(file.RepoID),
				OwnerId:   toProtoUUID(metadata.OwnerID),
				TagId:     toProtoUUID(metadata.TagID),
				FileName:  file.FileName,
				MimeType:  mimeType,
				TagName:   metadata.TagName,
				IsPublic:  metadata.IsPublic,
				UpdatedAt: timestamppb.New(file.UpdatedAt.UTC()),
			},
		},
	}

	return p.publish(ctx, file.ID.String(), envelope)
}

func (p *Producer) PublishFileDeleted(ctx context.Context, fileID uuid.UUID) error {
	envelope := &searcheventsv1.SearchEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  commonv1.SearchEventType_SEARCH_EVENT_TYPE_FILE_DELETED,
		OccurredAt: timestamppb.New(time.Now().UTC()),
		Payload: &searcheventsv1.SearchEventEnvelope_FileDeleted{
			FileDeleted: &searcheventsv1.FileDeleted{FileId: toProtoUUID(fileID)},
		},
	}

	return p.publish(ctx, fileID.String(), envelope)
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
