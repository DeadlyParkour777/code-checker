package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DeadlyParkour777/code-checker/auth_service/internal/store"
	"github.com/DeadlyParkour777/code-checker/auth_service/internal/types"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Service interface {
	Register(ctx context.Context, payload *types.UserRegisterPayload) (*types.User, error)
	Login(ctx context.Context, payload *types.UserLoginPayload) (*types.User, string, error)
	ValidateToken(ctx context.Context, tokenString string) (string, string, error)
}

type service struct {
	store     store.Store
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewService(store store.Store, jwtSecret string, tokenTTL time.Duration) *service {
	return &service{
		store:     store,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

func (s *service) Register(ctx context.Context, payload *types.UserRegisterPayload) (*types.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &types.User{
		Username: payload.Username,
		Password: string(hashedPassword),
		Role:     "user",
	}

	return s.store.CreateUser(user)
}

func (s *service) Login(ctx context.Context, payload *types.UserLoginPayload) (*types.User, string, error) {
	user, err := s.store.GetUserByUsername(payload.Username)
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password)); err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	claims := JWTClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign token: %w", err)
	}

	return user, signedToken, nil
}

func (s *service) ValidateToken(ctx context.Context, tokenString string) (string, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return "", "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if ok && token.Valid {
		return claims.UserID, claims.Role, nil
	}

	user, err := s.store.GetUserByID(claims.UserID)
	if err != nil {
		return "", "", fmt.Errorf("user not found")
	}

	if user.Role != claims.Role {
		return "", "", fmt.Errorf("user role has changed")
	}

	return "", "", fmt.Errorf("invalid token")
}
