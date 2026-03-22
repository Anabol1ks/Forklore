package kafka

import (
	"context"
	"strings"

	"ranking-service/internal/service"

	autheventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/events/v1"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type AuthHandler struct {
	svc    service.Service
	logger *zap.Logger
}

func NewAuthHandler(svc service.Service, logger *zap.Logger) *AuthHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuthHandler{svc: svc, logger: logger}
}

func (h *AuthHandler) HandleMessage(ctx context.Context, payload []byte) error {
	var envelope autheventsv1.AuthEventEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return err
	}

	switch event := envelope.Payload.(type) {
	case *autheventsv1.AuthEventEnvelope_UserRegistered:
		userID := mustParseUUID(event.UserRegistered.GetUserId().GetValue())
		username := strings.TrimSpace(event.UserRegistered.GetUsername())
		if userID == uuid.Nil {
			return nil
		}
		return h.svc.EnsureUser(ctx, userID, username)
	default:
		h.logger.Debug("unknown auth kafka event payload", zap.String("event_type", envelope.GetEventType().String()))
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
