package kafka

import (
	"context"
	"time"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	rankingeventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/events/v1"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RankingProducerConfig struct {
	Brokers      []string
	Topic        string
	ClientID     string
	WriteTimeout time.Duration
	BatchTimeout time.Duration
}

type RankingProducer interface {
	PublishUserFollowed(ctx context.Context, userID uuid.UUID, delta int64) error
	PublishUserUnfollowed(ctx context.Context, userID uuid.UUID, delta int64) error
	Close() error
}

type rankingProducer struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

func NewRankingProducer(cfg RankingProducerConfig, logger *zap.Logger) RankingProducer {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "forklore-profile-service"
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 5 * time.Second
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 50 * time.Millisecond
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		WriteTimeout: cfg.WriteTimeout,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &rankingProducer{
		writer: writer,
		logger: logger,
		topic:  cfg.Topic,
	}
}

func (p *rankingProducer) PublishUserFollowed(ctx context.Context, userID uuid.UUID, delta int64) error {
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_USER_FOLLOWED,
		OccurredAt: timestamppb.Now(),
		Payload: &rankingeventsv1.RankingEventEnvelope_UserFollowed{
			UserFollowed: &rankingeventsv1.UserFollowed{
				UserId: &commonv1.UUID{Value: userID.String()},
				Delta:  nonZeroDelta(delta),
			},
		},
	}
	return p.publish(ctx, userID.String(), envelope)
}

func (p *rankingProducer) PublishUserUnfollowed(ctx context.Context, userID uuid.UUID, delta int64) error {
	envelope := &rankingeventsv1.RankingEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  rankingeventsv1.RankingEventType_RANKING_EVENT_TYPE_USER_UNFOLLOWED,
		OccurredAt: timestamppb.Now(),
		Payload: &rankingeventsv1.RankingEventEnvelope_UserUnfollowed{
			UserUnfollowed: &rankingeventsv1.UserUnfollowed{
				UserId: &commonv1.UUID{Value: userID.String()},
				Delta:  nonZeroDelta(delta),
			},
		},
	}
	return p.publish(ctx, userID.String(), envelope)
}

func (p *rankingProducer) publish(ctx context.Context, key string, envelope *rankingeventsv1.RankingEventEnvelope) error {
	payload, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	}); err != nil {
		return err
	}

	p.logger.Info("ranking event published",
		zap.String("topic", p.topic),
		zap.String("event_type", envelope.GetEventType().String()),
		zap.String("key", key),
	)

	return nil
}

func (p *rankingProducer) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func nonZeroDelta(value int64) int64 {
	if value == 0 {
		return 1
	}
	return value
}
