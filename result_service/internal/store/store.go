package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/DeadlyParkour777/code-checker/result_service/internal/types"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Store interface {
	UpdateSubmissionStatus(ctx context.Context, submissionID, status string) error
	GetUserSubmissions(ctx context.Context, userID string) ([]*types.Submission, error)
}

type store struct {
	db    *sql.DB
	cache *redis.Client
}

func NewStore(db *sql.DB, cache *redis.Client) Store {
	return &store{db: db, cache: cache}
}

func (s *store) UpdateSubmissionStatus(ctx context.Context, submissionID, status string) error {
	query := `UPDATE submissions SET status = $1, updated_at = $2 WHERE id = $3 RETURNING user_id`
	var userID string
	err := s.db.QueryRowContext(ctx, query, status, time.Now(), submissionID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to update submission in db: %w", err)
	}

	cacheKey := fmt.Sprintf("submissions:%s", userID)
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate cache for user %s: %v", userID, err)
	}

	log.Printf("Updated submission %s to status %s and invalidated cache for user %s", submissionID, status, userID)
	return nil
}

func (s *store) GetUserSubmissions(ctx context.Context, userID string) ([]*types.Submission, error) {
	cacheKey := fmt.Sprintf("submissions:%s", userID)

	cachedData, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		log.Printf("Cache HIT for user %s", userID)
		var submissions []*types.Submission
		if err := json.Unmarshal([]byte(cachedData), &submissions); err == nil {
			return submissions, nil
		}
	}

	log.Printf("Cache MISS for user %s", userID)

	var submissions []*types.Submission
	query := `SELECT id, problem_id, user_id, language, status, created_at, updated_at 
	          FROM submissions WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submissions from db: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		sub := &types.Submission{}
		if err := rows.Scan(&sub.ID, &sub.ProblemID, &sub.UserID, &sub.Language, &sub.Status, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, sub)
	}

	jsonData, err := json.Marshal(submissions)
	if err == nil {
		s.cache.Set(ctx, cacheKey, jsonData, 5*time.Minute)
	}

	return submissions, nil
}
