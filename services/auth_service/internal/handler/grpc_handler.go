package handler

import (
	"context"
	"log"

	authpb "github.com/DeadlyParkour777/code-checker/pkg/auth"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/service"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	service service.Service
	authpb.UnimplementedAuthServiceServer
}

func NewGrpcHandler(service service.Service) *GrpcHandler {
	return &GrpcHandler{service: service}
}

func (h *GrpcHandler) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	u := &types.UserRegisterPayload{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
	}

	user, err := h.service.Register(ctx, u)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}

	return &authpb.RegisterResponse{
		UserId:  user.ID,
		Message: "User registered",
	}, nil
}

func (h *GrpcHandler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	u := &types.UserLoginPayload{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
	}

	user, token, err := h.service.Login(ctx, u)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login failed: %v", err)
	}

	return &authpb.LoginResponse{AccessToken: token, UserId: user.ID}, nil
}

func (h *GrpcHandler) ValidateToken(ctx context.Context, req *authpb.ValidateRequest) (*authpb.ValidateResponse, error) {
	userID, role, err := h.service.ValidateToken(ctx, req.GetToken())
	if err != nil {
		return &authpb.ValidateResponse{Valid: false}, nil
	}
	log.Println("role", role)

	return &authpb.ValidateResponse{Valid: true, UserId: userID, Role: role}, nil
}

func (h *GrpcHandler) RefreshToken(ctx context.Context, req *authpb.RefreshRequest) (*authpb.RefreshResponse, error) {
	// TODO: Обновление токена. Redis

	return nil, status.Errorf(codes.Unimplemented, "method RefreshToken not implemented")
}
