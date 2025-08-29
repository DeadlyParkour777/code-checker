package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	problem_service "github.com/DeadlyParkour777/code-checker/pkg/problem"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/store"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/types"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Service interface {
	CreateSubmission(ctx context.Context, userID, problemID, code, language string) (*types.Submission, error)
	// GetSubmission(...)
}

type service struct {
	store                store.Store
	kafkaProducer        *kafka.Writer
	problemServiceClient problem_service.ProblemServiceClient
}

func NewService(store store.Store, problemServiceAddr string, kafkaProducer *kafka.Writer) (Service, error) {
	conn, err := grpc.NewClient(problemServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to problem service: %w", err)
	}
	problemServiceClient := problem_service.NewProblemServiceClient(conn)
	return &service{
		store:                store,
		kafkaProducer:        kafkaProducer,
		problemServiceClient: problemServiceClient,
	}, nil
}

func (s *service) CreateSubmission(ctx context.Context, userID, problemID, code, language string) (*types.Submission, error) {
	submission := &types.Submission{
		ProblemID: problemID,
		UserID:    userID,
		Code:      code,
		Language:  language,
	}

	createdSubmission, err := s.store.CreateSubmission(submission)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission in store: %w", err)
	}
	event := types.SubmissionEvent{
		SubmissionID: createdSubmission.ID,
		ProblemID:    createdSubmission.ProblemID,
		Code:         createdSubmission.Code,
		Language:     createdSubmission.Language,
	}
	message, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submission event: %w", err)
	}

	err = s.kafkaProducer.WriteMessages(ctx, kafka.Message{
		// Topic: "submissions",
		Value: message,
		Time:  time.Now(),
	})

	if err != nil {
		log.Printf("Failed to write submission event: %v", err)
		return nil, err
	}

	return createdSubmission, nil
}
