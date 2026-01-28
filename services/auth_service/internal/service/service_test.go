package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/types"
	"golang.org/x/crypto/bcrypt"
)

type fakeStore struct {
	createUserFn      func(user *types.User) (*types.User, error)
	getByUsernameFn   func(username string) (*types.User, error)
	getByIDFn         func(id string) (*types.User, error)
	lastCreatedUser   *types.User
	lastUsernameQuery string
	lastIDQuery       string
}

func (f *fakeStore) CreateUser(user *types.User) (*types.User, error) {
	f.lastCreatedUser = user
	if f.createUserFn == nil {
		return nil, errors.New("CreateUser not implemented")
	}
	return f.createUserFn(user)
}

func (f *fakeStore) GetUserByUsername(username string) (*types.User, error) {
	f.lastUsernameQuery = username
	if f.getByUsernameFn == nil {
		return nil, errors.New("GetUserByUsername not implemented")
	}
	return f.getByUsernameFn(username)
}

func (f *fakeStore) GetUserByID(id string) (*types.User, error) {
	f.lastIDQuery = id
	if f.getByIDFn == nil {
		return nil, errors.New("GetUserByID not implemented")
	}
	return f.getByIDFn(id)
}

func TestRegister_HashesPasswordAndSetsRole(t *testing.T) {
	store := &fakeStore{
		createUserFn: func(user *types.User) (*types.User, error) {
			user.ID = "u1"
			return user, nil
		},
	}
	service := NewService(store, "secret", time.Hour)

	created, err := service.Register(context.Background(), &types.UserRegisterPayload{
		Username: "alice",
		Password: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "u1" {
		t.Fatalf("unexpected id: %s", created.ID)
	}
	if store.lastCreatedUser.Role != "user" {
		t.Fatalf("expected role user, got %s", store.lastCreatedUser.Role)
	}
	if store.lastCreatedUser.Password == "pass123" {
		t.Fatalf("password was not hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.lastCreatedUser.Password), []byte("pass123")); err != nil {
		t.Fatalf("hash does not match: %v", err)
	}
}

func TestLogin_SuccessReturnsToken(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	store := &fakeStore{
		getByUsernameFn: func(username string) (*types.User, error) {
			return &types.User{ID: "u1", Username: username, Password: string(hash), Role: "user"}, nil
		},
	}
	service := NewService(store, "secret", time.Hour)

	user, token, err := service.Login(context.Background(), &types.UserLoginPayload{
		Username: "alice",
		Password: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "u1" {
		t.Fatalf("unexpected user id: %s", user.ID)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	store := &fakeStore{
		getByUsernameFn: func(username string) (*types.User, error) {
			return &types.User{ID: "u1", Username: username, Password: string(hash), Role: "user"}, nil
		},
	}
	service := NewService(store, "secret", time.Hour)

	_, _, err := service.Login(context.Background(), &types.UserLoginPayload{
		Username: "alice",
		Password: "wrong",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	store := &fakeStore{
		getByUsernameFn: func(username string) (*types.User, error) {
			return &types.User{ID: "u1", Username: username, Password: string(hash), Role: "admin"}, nil
		},
	}
	service := NewService(store, "secret", time.Hour)

	_, token, err := service.Login(context.Background(), &types.UserLoginPayload{
		Username: "alice",
		Password: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uid, role, err := service.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "u1" || role != "admin" {
		t.Fatalf("unexpected claims: %s %s", uid, role)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	service := NewService(&fakeStore{}, "secret", time.Hour)

	_, _, err := service.ValidateToken(context.Background(), "invalid.token")
	if err == nil {
		t.Fatalf("expected error")
	}
}
