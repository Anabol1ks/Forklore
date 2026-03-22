package kafka

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"profile-service/internal/service"

	autheventsv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/events/v1"
)

type Handler struct {
	profileService service.ProfileService
	logger         *zap.Logger
}

func NewHandler(profileService service.ProfileService, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Handler{
		profileService: profileService,
		logger:         logger,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, payload []byte) error {
	var envelope autheventsv1.AuthEventEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return err
	}

	switch event := envelope.Payload.(type) {
	case *autheventsv1.AuthEventEnvelope_UserRegistered:
		userID := mustParseUUID(event.UserRegistered.GetUserId().GetValue())
		username := strings.TrimSpace(event.UserRegistered.GetUsername())

		if userID == uuid.Nil || username == "" {
			h.logger.Warn("skip invalid user.registered event",
				zap.String("event_id", envelope.GetEventId()),
			)
			return nil
		}

		h.logger.Info("processing user.registered event",
			zap.String("event_id", envelope.GetEventId()),
			zap.String("user_id", userID.String()),
			zap.String("username", username),
		)

		return h.profileService.CreateProfileIfNotExists(ctx, service.CreateProfileInput{
			UserID:   userID,
			Username: username,
		})

	default:
		h.logger.Warn("unknown auth kafka event payload",
			zap.String("event_id", envelope.GetEventId()),
			zap.String("event_type", envelope.GetEventType().String()),
		)
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
