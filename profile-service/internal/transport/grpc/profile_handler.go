package grpcserver

import (
	"context"
	"profile-service/internal/domain"
	"profile-service/internal/service"

	profilev1 "github.com/Anabol1ks/Forklore/pkg/pb/profile/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProfileHandler struct {
	profilev1.UnimplementedProfileServiceServer

	service service.ProfileService
	logger  *zap.Logger
}

func NewProfileHandler(service service.ProfileService, logger *zap.Logger) *ProfileHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ProfileHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ProfileHandler) GetMyProfile(ctx context.Context, _ *emptypb.Empty) (*profilev1.GetProfileResponse, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "get my profile: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "get my profile: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	state, err := h.service.GetMyProfile(ctx, claims.UserID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get my profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	return &profilev1.GetProfileResponse{
		Profile: toProtoProfile(state.Profile, state.FollowersCount, state.FollowingCount),
	}, nil
}

func (h *ProfileHandler) GetProfileByUserId(ctx context.Context, req *profilev1.GetProfileByUserIdRequest) (*profilev1.GetProfileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	userID, err := parseProtoUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	state, err := h.service.GetProfileByUserID(ctx, requesterID, userID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get profile by user id failed", err,
			zap.String("user_id", userID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &profilev1.GetProfileResponse{
		Profile: toProtoProfile(state.Profile, state.FollowersCount, state.FollowingCount),
	}, nil
}

func (h *ProfileHandler) GetProfileByUsername(ctx context.Context, req *profilev1.GetProfileByUsernameRequest) (*profilev1.GetProfileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	state, err := h.service.GetProfileByUsername(ctx, requesterID, req.GetUsername())
	if err != nil {
		return nil, LogAndMapError(h.logger, "get profile by username failed", err,
			zap.String("username", req.GetUsername()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &profilev1.GetProfileResponse{
		Profile: toProtoProfile(state.Profile, state.FollowersCount, state.FollowingCount),
	}, nil
}

func (h *ProfileHandler) UpdateProfile(ctx context.Context, req *profilev1.UpdateProfileRequest) (*profilev1.UpdateProfileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "update profile: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "update profile: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	state, err := h.service.UpdateProfile(ctx, service.UpdateProfileInput{
		RequesterID: claims.UserID,
		DisplayName: req.GetDisplayName(),
		Bio:         req.GetBio(),
		AvatarURL:   req.GetAvatarUrl(),
		CoverURL:    req.GetCoverUrl(),
		Location:    req.GetLocation(),
		WebsiteURL:  req.GetWebsiteUrl(),
		IsPublic:    req.GetIsPublic(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "update profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("profile updated",
		zap.String("requester_id", claims.UserID.String()),
	)

	return &profilev1.UpdateProfileResponse{
		Profile: toProtoProfile(state.Profile, state.FollowersCount, state.FollowingCount),
	}, nil
}

func (h *ProfileHandler) UpdateProfileReadme(ctx context.Context, req *profilev1.UpdateProfileReadmeRequest) (*profilev1.UpdateProfileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "update profile readme: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "update profile readme: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	state, err := h.service.UpdateProfileReadme(ctx, service.UpdateProfileReadmeInput{
		RequesterID:    claims.UserID,
		ReadmeMarkdown: req.GetReadmeMarkdown(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "update profile readme failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("profile readme updated",
		zap.String("requester_id", claims.UserID.String()),
	)

	return &profilev1.UpdateProfileResponse{
		Profile: toProtoProfile(state.Profile, state.FollowersCount, state.FollowingCount),
	}, nil
}

func (h *ProfileHandler) ListProfileSocialLinks(ctx context.Context, req *profilev1.ListProfileSocialLinksRequest) (*profilev1.ListProfileSocialLinksResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	userID, err := parseProtoUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	links, err := h.service.ListProfileSocialLinks(ctx, requesterID, userID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "list profile social links failed", err,
			zap.String("user_id", userID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &profilev1.ListProfileSocialLinksResponse{
		SocialLinks: toProtoProfileSocialLinks(links),
	}, nil
}

func (h *ProfileHandler) UpsertProfileSocialLink(ctx context.Context, req *profilev1.UpsertProfileSocialLinkRequest) (*profilev1.UpsertProfileSocialLinkResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "upsert social link: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "upsert social link: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	socialLinkID, err := parseOptionalProtoUUID(req.GetSocialLinkId(), "social_link_id")
	if err != nil {
		return nil, err
	}

	link, err := h.service.UpsertProfileSocialLink(ctx, service.UpsertProfileSocialLinkInput{
		RequesterID:  claims.UserID,
		SocialLinkID: socialLinkID,
		Platform:     toModelSocialPlatform(req.GetPlatform()),
		URL:          req.GetUrl(),
		Label:        req.GetLabel(),
		Position:     req.GetPosition(),
		IsVisible:    req.GetIsVisible(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "upsert social link failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("profile social link upserted",
		zap.String("requester_id", claims.UserID.String()),
		zap.String("social_link_id", link.ID.String()),
	)

	return &profilev1.UpsertProfileSocialLinkResponse{
		SocialLink: toProtoProfileSocialLink(link),
	}, nil
}

func (h *ProfileHandler) DeleteProfileSocialLink(ctx context.Context, req *profilev1.DeleteProfileSocialLinkRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "delete social link: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "delete social link: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	socialLinkID, err := parseProtoUUID(req.GetSocialLinkId(), "social_link_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteProfileSocialLink(ctx, claims.UserID, socialLinkID); err != nil {
		return nil, LogAndMapError(h.logger, "delete social link failed", err,
			zap.String("requester_id", claims.UserID.String()),
			zap.String("social_link_id", socialLinkID.String()),
		)
	}

	h.logger.Info("profile social link deleted",
		zap.String("requester_id", claims.UserID.String()),
		zap.String("social_link_id", socialLinkID.String()),
	)

	return &emptypb.Empty{}, nil
}

func (h *ProfileHandler) FollowUser(ctx context.Context, req *profilev1.FollowUserRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "follow user: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "follow user: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	followeeID, err := parseProtoUUID(req.GetFolloweeId(), "followee_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.FollowUser(ctx, claims.UserID, followeeID); err != nil {
		return nil, LogAndMapError(h.logger, "follow user failed", err,
			zap.String("follower_id", claims.UserID.String()),
			zap.String("followee_id", followeeID.String()),
		)
	}

	h.logger.Info("user followed",
		zap.String("follower_id", claims.UserID.String()),
		zap.String("followee_id", followeeID.String()),
	)

	return &emptypb.Empty{}, nil
}

func (h *ProfileHandler) UnfollowUser(ctx context.Context, req *profilev1.UnfollowUserRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "unfollow user: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "unfollow user: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	followeeID, err := parseProtoUUID(req.GetFolloweeId(), "followee_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.UnfollowUser(ctx, claims.UserID, followeeID); err != nil {
		return nil, LogAndMapError(h.logger, "unfollow user failed", err,
			zap.String("follower_id", claims.UserID.String()),
			zap.String("followee_id", followeeID.String()),
		)
	}

	h.logger.Info("user unfollowed",
		zap.String("follower_id", claims.UserID.String()),
		zap.String("followee_id", followeeID.String()),
	)

	return &emptypb.Empty{}, nil
}

func (h *ProfileHandler) ListFollowers(ctx context.Context, req *profilev1.ListFollowersRequest) (*profilev1.ListProfilePreviewsResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	userID, err := parseProtoUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	items, total, err := h.service.ListFollowers(ctx, requesterID, userID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list followers failed", err,
			zap.String("user_id", userID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &profilev1.ListProfilePreviewsResponse{
		Profiles: toProtoProfilePreviews(items),
		Total:    uint64(total),
	}, nil
}

func (h *ProfileHandler) ListFollowing(ctx context.Context, req *profilev1.ListFollowingRequest) (*profilev1.ListProfilePreviewsResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	userID, err := parseProtoUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	items, total, err := h.service.ListFollowing(ctx, requesterID, userID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list following failed", err,
			zap.String("user_id", userID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &profilev1.ListProfilePreviewsResponse{
		Profiles: toProtoProfilePreviews(items),
		Total:    uint64(total),
	}, nil
}

func (h *ProfileHandler) ListAvailableTitles(ctx context.Context, _ *emptypb.Empty) (*profilev1.ListAvailableTitlesResponse, error) {
	titles, err := h.service.ListAvailableTitles(ctx)
	if err != nil {
		return nil, LogAndMapError(h.logger, "list available titles failed", err)
	}

	return &profilev1.ListAvailableTitlesResponse{
		Titles: toProtoProfileTitles(titles),
	}, nil
}

func (h *ProfileHandler) SetProfileTitle(ctx context.Context, req *profilev1.SetProfileTitleRequest) (*profilev1.UpdateProfileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "set profile title: missing claims", domain.ErrUnauthorized)
	}

	if err := h.ensureRequesterProfile(ctx, claims); err != nil {
		return nil, LogAndMapError(h.logger, "set profile title: ensure profile failed", err,
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	state, err := h.service.SetProfileTitle(ctx, service.SetProfileTitleInput{
		RequesterID: claims.UserID,
		TitleCode:   req.GetTitleCode(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "set profile title failed", err,
			zap.String("requester_id", claims.UserID.String()),
			zap.String("title_code", req.GetTitleCode()),
		)
	}

	h.logger.Info("profile title updated",
		zap.String("requester_id", claims.UserID.String()),
		zap.String("title_code", req.GetTitleCode()),
	)

	return &profilev1.UpdateProfileResponse{
		Profile: toProtoProfile(state.Profile, state.FollowersCount, state.FollowingCount),
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

func (h *ProfileHandler) ensureRequesterProfile(ctx context.Context, claims *authjwt.AccessClaims) error {
	if claims == nil {
		return domain.ErrUnauthorized
	}

	return h.service.CreateProfileIfNotExists(ctx, service.CreateProfileInput{
		UserID:   claims.UserID,
		Username: claims.Username,
	})
}
