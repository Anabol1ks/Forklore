package grpcserver

import (
	"context"
	"search-service/internal/service"

	searchv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type SearchHandler struct {
	searchv1.UnimplementedSearchServiceServer

	service service.SearchService
	logger  *zap.Logger
}

func NewSearchHandler(service service.SearchService, logger *zap.Logger) *SearchHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SearchHandler{
		service: service,
		logger:  logger,
	}
}

func (h *SearchHandler) Search(ctx context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	tagID, err := parseOptionalProtoUUID(req.GetTagId(), "tag_id")
	if err != nil {
		return nil, err
	}

	ownerID, err := parseOptionalProtoUUID(req.GetOwnerId(), "owner_id")
	if err != nil {
		return nil, err
	}

	repoID, err := parseOptionalProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	hits, total, err := h.service.Search(ctx, service.SearchParams{
		Query:       req.GetQuery(),
		EntityTypes: toModelSearchEntityTypes(req.GetEntityTypes()),
		TagID:       tagID,
		OwnerID:     ownerID,
		RepoID:      repoID,
		Limit:       int(req.GetLimit()),
		Offset:      int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "search failed", err,
			zap.String("query", req.GetQuery()),
			zap.Uint32("limit", req.GetLimit()),
			zap.Uint32("offset", req.GetOffset()),
		)
	}

	h.logger.Debug("search completed",
		zap.String("query", req.GetQuery()),
		zap.Int("hits_count", len(hits)),
		zap.Uint64("total", uint64(total)),
	)

	return &searchv1.SearchResponse{
		Hits:  toProtoSearchHits(hits),
		Total: uint64(total),
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
