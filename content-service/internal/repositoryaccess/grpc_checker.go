package repositoryaccess

import (
	"content-service/internal/domain"
	"content-service/internal/service"
	"context"
	"strings"
	"time"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

var _ service.RepositoryAccessChecker = (*GRPCChecker)(nil)

type GRPCChecker struct {
	client         repositoryv1.RepositoryServiceClient
	logger         *zap.Logger
	requestTimeout time.Duration
}

func NewGRPCChecker(
	client repositoryv1.RepositoryServiceClient,
	logger *zap.Logger,
	requestTimeout time.Duration,
) *GRPCChecker {
	if logger == nil {
		logger = zap.NewNop()
	}
	if requestTimeout <= 0 {
		requestTimeout = 3 * time.Second
	}

	return &GRPCChecker{
		client:         client,
		logger:         logger,
		requestTimeout: requestTimeout,
	}
}

func (c *GRPCChecker) EnsureCanRead(ctx context.Context, repoID, requesterID uuid.UUID) error {
	callCtx, cancel := context.WithTimeout(withForwardedAuthorization(ctx), c.requestTimeout)
	defer cancel()

	_, err := c.client.GetRepositoryById(callCtx, &repositoryv1.GetRepositoryByIdRequest{
		RepoId: toProtoUUID(repoID),
	})
	if err != nil {
		mapped := mapRepositoryGRPCError(err)
		c.logger.Warn("repository read access check failed",
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
			zap.Error(err),
		)
		return mapped
	}

	return nil
}

func (c *GRPCChecker) EnsureCanWrite(ctx context.Context, repoID, requesterID uuid.UUID) error {
	if requesterID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	callCtx, cancel := context.WithTimeout(withForwardedAuthorization(ctx), c.requestTimeout)
	defer cancel()

	resp, err := c.client.GetRepositoryById(callCtx, &repositoryv1.GetRepositoryByIdRequest{
		RepoId: toProtoUUID(repoID),
	})
	if err != nil {
		mapped := mapRepositoryGRPCError(err)
		c.logger.Warn("repository write access precheck failed",
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
			zap.Error(err),
		)
		return mapped
	}

	if resp.GetRepository() == nil || resp.GetRepository().GetOwnerId() == nil {
		c.logger.Error("repository service returned empty owner",
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
		)
		return domain.ErrRepositoryNotFound
	}

	ownerID, err := uuid.Parse(strings.TrimSpace(resp.GetRepository().GetOwnerId().GetValue()))
	if err != nil {
		c.logger.Error("failed to parse repository owner id",
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
			zap.String("owner_id", resp.GetRepository().GetOwnerId().GetValue()),
			zap.Error(err),
		)
		return domain.ErrRepositoryNotFound
	}

	if ownerID != requesterID {
		return domain.ErrContentAccessDenied
	}

	return nil
}

func withForwardedAuthorization(ctx context.Context) context.Context {
	incomingMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	authValues := incomingMD.Get("authorization")
	if len(authValues) == 0 {
		return ctx
	}

	outgoingMD := metadata.Pairs("authorization", authValues[0])
	return metadata.NewOutgoingContext(ctx, outgoingMD)
}

func mapRepositoryGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch grpcstatus.Code(err) {
	case codes.NotFound:
		return domain.ErrRepositoryNotFound
	case codes.PermissionDenied:
		return domain.ErrContentAccessDenied
	case codes.Unauthenticated:
		return domain.ErrUnauthorized
	default:
		return err
	}
}

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{
		Value: id.String(),
	}
}
