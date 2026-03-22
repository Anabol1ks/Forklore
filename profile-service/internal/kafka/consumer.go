package kafka

import (
	"context"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type ConsumerConfig struct {
	Brokers         []string
	Topic           string
	GroupID         string
	MinBytes        int
	MaxBytes        int
	MaxWait         time.Duration
	CommitInterval  time.Duration
	ReadLagInterval time.Duration
	StartOffset     int64
	HandleTimeout   time.Duration
}

type Consumer struct {
	reader  *kafka.Reader
	handler *Handler
	logger  *zap.Logger
}

func NewConsumer(cfg ConsumerConfig, handler *Handler, logger *zap.Logger) *Consumer {
	if logger == nil {
		logger = zap.NewNop()
	}

	if cfg.MinBytes <= 0 {
		cfg.MinBytes = 1
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 10e6
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 2 * time.Second
	}
	if cfg.CommitInterval <= 0 {
		cfg.CommitInterval = time.Second
	}
	if cfg.ReadLagInterval == 0 {
		cfg.ReadLagInterval = -1
	}
	if cfg.StartOffset == 0 {
		cfg.StartOffset = kafka.FirstOffset
	}
	if cfg.HandleTimeout <= 0 {
		cfg.HandleTimeout = 10 * time.Second
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         cfg.Brokers,
		GroupID:         cfg.GroupID,
		Topic:           cfg.Topic,
		MinBytes:        cfg.MinBytes,
		MaxBytes:        cfg.MaxBytes,
		MaxWait:         cfg.MaxWait,
		CommitInterval:  cfg.CommitInterval,
		ReadLagInterval: cfg.ReadLagInterval,
		StartOffset:     cfg.StartOffset,
	})

	return &Consumer{
		reader:  reader,
		handler: handler,
		logger:  logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("profile kafka consumer started")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				c.logger.Info("profile kafka consumer stopped by context")
				return nil
			}

			c.logger.Error("failed to fetch kafka message", zap.Error(err))
			return err
		}

		msgCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = c.handler.HandleMessage(msgCtx, msg.Value)
		cancel()

		if err != nil {
			c.logger.Error("failed to handle kafka message",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.ByteString("key", msg.Key),
				zap.Error(err),
			)

			// offset не коммитим — сообщение будет обработано повторно
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("failed to commit kafka message",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
			continue
		}

		c.logger.Debug("profile kafka message processed",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)
	}
}

func (c *Consumer) Close() error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
