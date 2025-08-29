package store

import (
	"database/sql"
	"fmt"

	"github.com/DeadlyParkour777/code-checker/auth_service/internal/types"
)

type Store interface {
	CreateUser(user *types.User) (*types.User, error)
	GetUserByUsername(username string) (*types.User, error)
	GetUserByID(id string) (*types.User, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *store {
	return &store{db: db}
}

func (s *store) CreateUser(user *types.User) (*types.User, error) {
	query := `INSERT INTO users (username, password, role) VALUES ($1, $2, $3) RETURNING id, created_at`

	err := s.db.QueryRow(query, user.Username, user.Password, user.Role).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("could not create user: %w", err)
	}

	return user, nil
}

func (s *store) GetUserByUsername(username string) (*types.User, error) {
	user := new(types.User)
	query := `SELECT id, username, password, role, created_at FROM users WHERE username = $1`

	err := s.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not get user: %w", err)
	}

	return user, nil
}

func (s *store) GetUserByID(id string) (*types.User, error) {
	user := new(types.User)
	query := `SELECT id, username, password, role, created_at FROM users WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with id %s not found", id)
		}
		return nil, fmt.Errorf("could not get user by id: %w", err)
	}

	return user, nil
}
