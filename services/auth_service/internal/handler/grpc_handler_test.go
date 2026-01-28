package handler

import (
	"context"
	"errors"
	"testing"

	authpb "github.com/DeadlyParkour777/code-checker/pkg/auth"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeService struct {
	registerFn      func(ctx context.Context, payload *types.UserRegisterPayload) (*types.User, error)
	loginFn         func(ctx context.Context, payload *types.UserLoginPayload) (*types.User, string, error)
	validateTokenFn func(ctx context.Context, token string) (string, string, error)
}

func (f *fakeService) Register(ctx context.Context, payload *types.UserRegisterPayload) (*types.User, error) {
	if f.registerFn == nil {
		return nil, errors.New("Register not implemented")
	}
	return f.registerFn(ctx, payload)
}

func (f *fakeService) Login(ctx context.Context, payload *types.UserLoginPayload) (*types.User, string, error) {
	if f.loginFn == nil {
		return nil, "", errors.New("Login not implemented")
	}
	return f.loginFn(ctx, payload)
}

func (f *fakeService) ValidateToken(ctx context.Context, token string) (string, string, error) {
	if f.validateTokenFn == nil {
		return "", "", errors.New("ValidateToken not implemented")
	}
	return f.validateTokenFn(ctx, token)
}

func TestRegister(t *testing.T) {
	svc := &fakeService{
		registerFn: func(_ context.Context, payload *types.UserRegisterPayload) (*types.User, error) {
			return &types.User{ID: "u1", Username: payload.Username}, nil
		},
	}
	h := NewGrpcHandler(svc)

	resp, err := h.Register(context.Background(), &authpb.RegisterRequest{Username: "alice", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetUserId() != "u1" {
		t.Fatalf("unexpected user id: %s", resp.GetUserId())
	}
}

func TestRegister_Error(t *testing.T) {
	svc := &fakeService{
		registerFn: func(_ context.Context, _ *types.UserRegisterPayload) (*types.User, error) {
			return nil, errors.New("boom")
		},
	}
	h := NewGrpcHandler(svc)

	_, err := h.Register(context.Background(), &authpb.RegisterRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %v", status.Code(err))
	}
}

func TestLogin(t *testing.T) {
	svc := &fakeService{
		loginFn: func(_ context.Context, payload *types.UserLoginPayload) (*types.User, string, error) {
			return &types.User{ID: "u1", Username: payload.Username}, "token", nil
		},
	}
	h := NewGrpcHandler(svc)

	resp, err := h.Login(context.Background(), &authpb.LoginRequest{Username: "alice", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetAccessToken() != "token" {
		t.Fatalf("unexpected token")
	}
}

func TestLogin_Error(t *testing.T) {
	svc := &fakeService{
		loginFn: func(_ context.Context, _ *types.UserLoginPayload) (*types.User, string, error) {
			return nil, "", errors.New("bad")
		},
	}
	h := NewGrpcHandler(svc)

	_, err := h.Login(context.Background(), &authpb.LoginRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", status.Code(err))
	}
}

func TestValidateToken(t *testing.T) {
	svc := &fakeService{
		validateTokenFn: func(_ context.Context, _ string) (string, string, error) {
			return "u1", "admin", nil
		},
	}
	h := NewGrpcHandler(svc)

	resp, err := h.ValidateToken(context.Background(), &authpb.ValidateRequest{Token: "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetValid() || resp.GetUserId() != "u1" || resp.GetRole() != "admin" {
		t.Fatalf("unexpected response")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := &fakeService{
		validateTokenFn: func(_ context.Context, _ string) (string, string, error) {
			return "", "", errors.New("invalid")
		},
	}
	h := NewGrpcHandler(svc)

	resp, err := h.ValidateToken(context.Background(), &authpb.ValidateRequest{Token: "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetValid() {
		t.Fatalf("expected invalid token")
	}
}

func TestRefreshToken_Unimplemented(t *testing.T) {
	h := NewGrpcHandler(&fakeService{})

	_, err := h.RefreshToken(context.Background(), &authpb.RefreshRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected unimplemented, got %v", status.Code(err))
	}
}
