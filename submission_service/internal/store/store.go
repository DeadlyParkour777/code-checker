package store

import (
	"database/sql"
	"fmt"

	"github.com/DeadlyParkour777/code-checker/submission_service/internal/types"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Store interface {
	CreateSubmission(submission *types.Submission) (*types.Submission, error)
	GetSubmission(id string) (*types.Submission, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreateSubmission(submission *types.Submission) (*types.Submission, error) {
	submission.ID = uuid.New().String()
	submission.Status = "Pending"

	query := `INSERT INTO submissions (id, problem_id, user_id, code, language, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			  RETURNING created_at, updated_at`

	err := s.db.QueryRow(query,
		submission.ID,
		submission.ProblemID,
		submission.UserID,
		submission.Code,
		submission.Language,
		submission.Status,
	).Scan(&submission.CreatedAt, &submission.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	return submission, nil
}

func (s *store) GetSubmission(id string) (*types.Submission, error) {
	submission := &types.Submission{}
	query := `SELECT id, problem_id, user_id, code, language, status, created_at, updated_at
			  FROM submissions WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(
		&submission.ID,
		&submission.ProblemID,
		&submission.UserID,
		&submission.Code,
		&submission.Language,
		&submission.Status,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	return submission, nil
}
