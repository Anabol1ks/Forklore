package kafka

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	autheventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/events/v1"
	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
)

type ProducerConfig struct {
	Brokers      []string
	AuthTopic    string
	ClientID     string
	WriteTimeout time.Duration
	BatchTimeout time.Duration
}

type Producer interface {
	PublishUserRegistered(ctx context.Context, userID uuid.UUID, username, email string) error
	Close() error
}

type producer struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

func NewProducer(cfg ProducerConfig, logger *zap.Logger) Producer {
	if logger == nil {
		logger = zap.NewNop()
	}

	if cfg.ClientID == "" {
		cfg.ClientID = "forklore-auth-service"
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 5 * time.Second
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 50 * time.Millisecond
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.AuthTopic,
		Balancer:     &kafka.Hash{},
		WriteTimeout: cfg.WriteTimeout,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &producer{
		writer: writer,
		logger: logger,
		topic:  cfg.AuthTopic,
	}
}

func (p *producer) PublishUserRegistered(ctx context.Context, userID uuid.UUID, username, email string) error {
	envelope := &autheventsv1.AuthEventEnvelope{
		EventId:    uuid.NewString(),
		EventType:  autheventsv1.AuthEventType_AUTH_EVENT_TYPE_USER_REGISTERED,
		OccurredAt: timestamppb.Now(),
		Payload: &autheventsv1.AuthEventEnvelope_UserRegistered{
			UserRegistered: &autheventsv1.UserRegistered{
				UserId: &commonv1.UUID{
					Value: userID.String(),
				},
				Username: username,
				Email:    email,
			},
		},
	}

	payload, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(userID.String()),
		Value: payload,
		Time:  time.Now().UTC(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return err
	}

	p.logger.Info("auth event published",
		zap.String("topic", p.topic),
		zap.String("event_type", envelope.GetEventType().String()),
		zap.String("user_id", userID.String()),
	)

	return nil
}

func (p *producer) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
