package grpcserver

import (
	"context"
	"repository-service/internal/domain"
	"repository-service/internal/service"
	"strings"

	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RepositoryHandler struct {
	repositoryv1.UnimplementedRepositoryServiceServer

	service service.RepositoryService
	logger  *zap.Logger
}

func NewRepositoryHandler(service service.RepositoryService, logger *zap.Logger) *RepositoryHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &RepositoryHandler{
		service: service,
		logger:  logger,
	}
}

func (h *RepositoryHandler) CreateRepository(ctx context.Context, req *repositoryv1.CreateRepositoryRequest) (*repositoryv1.CreateRepositoryResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "create repository: missing claims", domain.ErrUnauthorized)
	}

	tagID, err := parseProtoUUID(req.GetTagId(), "tag_id")
	if err != nil {
		return nil, err
	}

	repo, err := h.service.CreateRepository(ctx, service.CreateRepositoryInput{
		OwnerID:       claims.UserID,
		OwnerUsername: claims.Username,
		TagID:         tagID,
		Name:          req.GetName(),
		Slug:          req.GetSlug(),
		Description:   req.GetDescription(),
		Visibility:    toModelRepositoryVisibility(req.GetVisibility()),
		Type:          toModelRepositoryType(req.GetType()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "create repository failed", err,
			zap.String("owner_id", claims.UserID.String()),
			zap.String("name", req.GetName()),
			zap.String("slug", req.GetSlug()),
			zap.String("tag_id", tagID.String()),
		)
	}

	h.logger.Info("repository created",
		zap.String("repo_id", repo.ID.String()),
		zap.String("owner_id", repo.OwnerID.String()),
		zap.String("slug", repo.Slug),
	)

	return &repositoryv1.CreateRepositoryResponse{
		Repository: toProtoRepository(repo),
	}, nil
}

func (h *RepositoryHandler) GetRepositoryById(ctx context.Context, req *repositoryv1.GetRepositoryByIdRequest) (*repositoryv1.GetRepositoryResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	repo, err := h.service.GetRepositoryByID(ctx, requesterID, repoID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get repository by id failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	h.logger.Debug("repository fetched by id",
		zap.String("repo_id", repoID.String()),
		zap.String("requester_id", requesterID.String()),
	)

	return &repositoryv1.GetRepositoryResponse{
		Repository: toProtoRepository(repo),
	}, nil
}

func (h *RepositoryHandler) GetRepositoryBySlug(ctx context.Context, req *repositoryv1.GetRepositoryBySlugRequest) (*repositoryv1.GetRepositoryResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	ownerKey := strings.TrimSpace(req.GetOwnerId().GetValue())
	if ownerKey == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "owner_id is required")
	}

	requesterID := requesterIDFromContext(ctx)
	requesterUsername := requesterUsernameFromContext(ctx)

	repo, err := h.service.GetRepositoryBySlug(ctx, requesterID, requesterUsername, ownerKey, req.GetSlug())
	if err != nil {
		return nil, LogAndMapError(h.logger, "get repository by slug failed", err,
			zap.String("owner_key", ownerKey),
			zap.String("slug", req.GetSlug()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	h.logger.Debug("repository fetched by slug",
		zap.String("owner_key", ownerKey),
		zap.String("slug", req.GetSlug()),
		zap.String("requester_id", requesterID.String()),
	)

	return &repositoryv1.GetRepositoryResponse{
		Repository: toProtoRepository(repo),
	}, nil
}

func (h *RepositoryHandler) UpdateRepository(ctx context.Context, req *repositoryv1.UpdateRepositoryRequest) (*repositoryv1.UpdateRepositoryResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "update repository: missing claims", domain.ErrUnauthorized)
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	tagID, err := parseOptionalProtoUUID(req.GetTagId(), "tag_id")
	if err != nil {
		return nil, err
	}

	repo, err := h.service.UpdateRepository(ctx, service.UpdateRepositoryInput{
		RequesterID: claims.UserID,
		RepoID:      repoID,
		TagID:       tagID,
		Name:        req.GetName(),
		Slug:        req.GetSlug(),
		Description: req.GetDescription(),
		Visibility:  toModelRepositoryVisibility(req.GetVisibility()),
		Type:        toModelRepositoryType(req.GetType()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "update repository failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("repository updated",
		zap.String("repo_id", repo.ID.String()),
		zap.String("requester_id", claims.UserID.String()),
		zap.String("slug", repo.Slug),
	)

	return &repositoryv1.UpdateRepositoryResponse{
		Repository: toProtoRepository(repo),
	}, nil
}

func (h *RepositoryHandler) DeleteRepository(ctx context.Context, req *repositoryv1.DeleteRepositoryRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "delete repository: missing claims", domain.ErrUnauthorized)
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteRepository(ctx, claims.UserID, repoID); err != nil {
		return nil, LogAndMapError(h.logger, "delete repository failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("repository deleted",
		zap.String("repo_id", repoID.String()),
		zap.String("requester_id", claims.UserID.String()),
	)

	return &emptypb.Empty{}, nil
}

func (h *RepositoryHandler) ForkRepository(ctx context.Context, req *repositoryv1.ForkRepositoryRequest) (*repositoryv1.ForkRepositoryResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "fork repository: missing claims", domain.ErrUnauthorized)
	}

	sourceRepoID, err := parseProtoUUID(req.GetSourceRepoId(), "source_repo_id")
	if err != nil {
		return nil, err
	}

	repo, err := h.service.ForkRepository(ctx, service.ForkRepositoryInput{
		RequesterID:       claims.UserID,
		RequesterUsername: claims.Username,
		SourceRepoID:      sourceRepoID,
		Name:              req.GetName(),
		Slug:              req.GetSlug(),
		Description:       req.GetDescription(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "fork repository failed", err,
			zap.String("source_repo_id", sourceRepoID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("repository forked",
		zap.String("repo_id", repo.ID.String()),
		zap.String("source_repo_id", sourceRepoID.String()),
		zap.String("requester_id", claims.UserID.String()),
	)

	return &repositoryv1.ForkRepositoryResponse{
		Repository: toProtoRepository(repo),
	}, nil
}

func (h *RepositoryHandler) ListMyRepositories(ctx context.Context, req *repositoryv1.ListMyRepositoriesRequest) (*repositoryv1.ListRepositoriesResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "list my repositories: missing claims", domain.ErrUnauthorized)
	}

	repos, total, err := h.service.ListMyRepositories(ctx, claims.UserID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list my repositories failed", err,
			zap.String("owner_id", claims.UserID.String()),
		)
	}

	h.logger.Debug("repositories listed for owner",
		zap.String("owner_id", claims.UserID.String()),
		zap.Int("count", len(repos)),
		zap.Uint64("total", uint64(total)),
	)

	return &repositoryv1.ListRepositoriesResponse{
		Repositories: toProtoRepositories(repos),
		Total:        uint64(total),
	}, nil
}

func (h *RepositoryHandler) ListUserRepositories(ctx context.Context, req *repositoryv1.ListUserRepositoriesRequest) (*repositoryv1.ListRepositoriesResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	ownerKey := strings.TrimSpace(req.GetOwnerId().GetValue())
	if ownerKey == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "owner_id is required")
	}

	requesterID := requesterIDFromContext(ctx)
	requesterUsername := requesterUsernameFromContext(ctx)

	repos, total, err := h.service.ListUserRepositories(ctx, requesterID, requesterUsername, ownerKey, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list user repositories failed", err,
			zap.String("owner_key", ownerKey),
			zap.String("requester_id", requesterID.String()),
		)
	}

	h.logger.Debug("repositories listed for user",
		zap.String("owner_key", ownerKey),
		zap.String("requester_id", requesterID.String()),
		zap.Int("count", len(repos)),
		zap.Uint64("total", uint64(total)),
	)

	return &repositoryv1.ListRepositoriesResponse{
		Repositories: toProtoRepositories(repos),
		Total:        uint64(total),
	}, nil
}

func (h *RepositoryHandler) ListForks(ctx context.Context, req *repositoryv1.ListForksRequest) (*repositoryv1.ListRepositoriesResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	repos, total, err := h.service.ListForks(ctx, requesterID, repoID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list repository forks failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	h.logger.Debug("repository forks listed",
		zap.String("repo_id", repoID.String()),
		zap.String("requester_id", requesterID.String()),
		zap.Int("count", len(repos)),
		zap.Uint64("total", uint64(total)),
	)

	return &repositoryv1.ListRepositoriesResponse{
		Repositories: toProtoRepositories(repos),
		Total:        uint64(total),
	}, nil
}

func (h *RepositoryHandler) ListRepositoryTags(ctx context.Context, _ *emptypb.Empty) (*repositoryv1.ListRepositoryTagsResponse, error) {
	tags, err := h.service.ListRepositoryTags(ctx)
	if err != nil {
		return nil, LogAndMapError(h.logger, "list repository tags failed", err)
	}

	h.logger.Debug("repository tags listed",
		zap.Int("count", len(tags)),
	)

	return &repositoryv1.ListRepositoryTagsResponse{
		Tags: toProtoTags(tags),
	}, nil
}

type protoValidator interface {
	ValidateAll() error
}

func validateProto(v protoValidator) error {
	if err := v.ValidateAll(); err != nil {
		return grpcstatus.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func requesterIDFromContext(ctx context.Context) uuid.UUID {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return uuid.Nil
	}
	return claims.UserID
}

func requesterUsernameFromContext(ctx context.Context) string {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return ""
	}
	return claims.Username
}
