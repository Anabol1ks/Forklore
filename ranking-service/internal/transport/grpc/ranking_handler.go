package grpcserver

import (
	"context"
	"ranking-service/internal/service"

	rankingv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type RankingHandler struct {
	rankingv1.UnimplementedRankingServiceServer

	svc    service.Service
	logger *zap.Logger
}

func NewRankingHandler(svc service.Service, logger *zap.Logger) *RankingHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RankingHandler{svc: svc, logger: logger}
}

func (h *RankingHandler) GetOverallLeaderboard(ctx context.Context, req *rankingv1.GetLeaderboardRequest) (*rankingv1.GetLeaderboardResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	entries, total, err := h.svc.ListOverall(ctx, service.ListParams{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "get overall leaderboard failed", err)
	}

	return &rankingv1.GetLeaderboardResponse{Entries: toProtoEntries(entries), Total: uint64(total)}, nil
}

func (h *RankingHandler) GetMonthlyLeaderboard(ctx context.Context, req *rankingv1.GetLeaderboardRequest) (*rankingv1.GetLeaderboardResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	entries, total, err := h.svc.ListMonthly(ctx, service.ListParams{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "get monthly leaderboard failed", err)
	}

	return &rankingv1.GetLeaderboardResponse{Entries: toProtoEntries(entries), Total: uint64(total)}, nil
}

func (h *RankingHandler) GetSubjectLeaderboard(ctx context.Context, req *rankingv1.GetSubjectLeaderboardRequest) (*rankingv1.GetLeaderboardResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	tagID, err := parseRequiredUUID(req.GetTagId(), "tag_id")
	if err != nil {
		return nil, err
	}

	entries, total, err := h.svc.ListSubject(ctx, service.ListSubjectParams{
		TagID:  tagID,
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "get subject leaderboard failed", err)
	}

	return &rankingv1.GetLeaderboardResponse{Entries: toProtoEntries(entries), Total: uint64(total)}, nil
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
