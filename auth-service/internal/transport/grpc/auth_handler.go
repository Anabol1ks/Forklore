package grpcserver

import (
	"auth-service/internal/domain"
	"auth-service/internal/service"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
)

type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	out, err := h.authService.Register(ctx, service.RegisterInput{
		Username: req.GetUsername(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
		Meta: service.ClientMeta{
			DeviceName: req.GetDeviceName(),
			UserAgent:  req.GetUserAgent(),
			IP:         req.GetIp(),
		},
	})
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return toProtoAuthResponse(out), nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	out, err := h.authService.Login(ctx, service.LoginInput{
		Login:    req.GetLogin(),
		Password: req.GetPassword(),
		Meta: service.ClientMeta{
			DeviceName: req.GetDeviceName(),
			UserAgent:  req.GetUserAgent(),
			IP:         req.GetIp(),
		},
	})
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return toProtoAuthResponse(out), nil
}

func (h *AuthHandler) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.AuthResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	out, err := h.authService.Refresh(ctx, service.RefreshInput{
		RefreshToken: req.GetRefreshToken(),
		Meta: service.ClientMeta{
			DeviceName: req.GetDeviceName(),
			UserAgent:  req.GetUserAgent(),
			IP:         req.GetIp(),
		},
	})
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return toProtoAuthResponse(out), nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	if err := h.authService.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, ToGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *AuthHandler) LogoutAll(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, ToGRPCError(domain.ErrUnauthorized)
	}

	if err := h.authService.LogoutAll(ctx, claims.UserID); err != nil {
		return nil, ToGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *AuthHandler) Introspect(ctx context.Context, req *authv1.IntrospectRequest) (*authv1.IntrospectResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	out, err := h.authService.Introspect(ctx, req.GetAccessToken())
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return toProtoIntrospectResponse(out), nil
}

func (h *AuthHandler) GetMe(ctx context.Context, _ *emptypb.Empty) (*authv1.GetMeResponse, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, ToGRPCError(domain.ErrUnauthorized)
	}

	user, err := h.authService.GetMe(ctx, claims.UserID)
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return &authv1.GetMeResponse{
		User: toProtoUser(user),
	}, nil
}

type protoValidator interface {
	ValidateAll() error
}

func validateProto(v protoValidator) error {
	if err := v.ValidateAll(); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}
