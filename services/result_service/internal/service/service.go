package service

import (
	"context"
	"log"

	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/store"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/types"
)

type Service interface {
	ProcessResult(ctx context.Context, result *types.ResultEvent)
	GetUserSubmissions(ctx context.Context, userID string) ([]*types.Submission, error)
}

type service struct {
	store store.Store
}

func NewService(store store.Store) Service {
	return &service{store: store}
}

func (s *service) ProcessResult(ctx context.Context, result *types.ResultEvent) {
	go func() {
		err := s.store.UpdateSubmissionStatus(context.Background(), result.SubmissionID, result.Status)
		if err != nil {
			log.Printf("Error processing result for submission %s: %v", result.SubmissionID, err)
		}
	}()
}

func (s *service) GetUserSubmissions(ctx context.Context, userID string) ([]*types.Submission, error) {
	return s.store.GetUserSubmissions(ctx, userID)
}
