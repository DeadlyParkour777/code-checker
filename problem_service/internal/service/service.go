package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/DeadlyParkour777/code-checker/problem_service/internal/store"
	"github.com/DeadlyParkour777/code-checker/problem_service/internal/types"
	"github.com/segmentio/kafka-go"
)

type Service interface {
	CreateProblem(ctx context.Context, title, description string) (*types.Problem, error)
	GetProblem(ctx context.Context, id string) (*types.Problem, error)
	ListProblems(ctx context.Context) ([]*types.Problem, error)
	CreateTestCase(ctx context.Context, problemID, input, output string) (*types.TestCase, error)
	GetTestCases(ctx context.Context, problemID string) ([]*types.TestCase, error)
}

type service struct {
	store         store.Store
	kafkaTopic    string
	kafkaProducer *kafka.Writer
}

func NewService(store store.Store, kafkaTopic string, kafkaProducer *kafka.Writer) Service {
	return &service{
		store:         store,
		kafkaTopic:    kafkaTopic,
		kafkaProducer: kafkaProducer,
	}
}

func (s *service) CreateProblem(ctx context.Context, title, description string) (*types.Problem, error) {
	problem := &types.Problem{
		Title:       title,
		Description: description,
	}

	createdProblem, err := s.store.CreateProblem(problem)
	if err != nil {
		return nil, fmt.Errorf("failed to create problem: %w", err)
	}

	event := types.ProblemEvent{
		EventType: "created",
		Problem:   createdProblem,
	}
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal problem event: %v", err)
	}

	err = s.kafkaProducer.WriteMessages(ctx, kafka.Message{
		Topic: s.kafkaTopic,
		Value: message,
		Time:  time.Now(),
	})
	if err != nil {
		log.Printf("Failed to produce problem event: %v", err)
	}
	return createdProblem, nil
}

func (s *service) GetProblem(ctx context.Context, id string) (*types.Problem, error) {
	return s.store.GetProblem(id)
}

func (s *service) ListProblems(ctx context.Context) ([]*types.Problem, error) {
	return s.store.ListProblems()
}

func (s *service) CreateTestCase(ctx context.Context, problemID, input, output string) (*types.TestCase, error) {
	testCase := &types.TestCase{
		ProblemID: problemID,
		Input:     input,
		Output:    output,
	}
	return s.store.CreateTestCase(testCase)
}

func (s *service) GetTestCases(ctx context.Context, problemID string) ([]*types.TestCase, error) {
	return s.store.GetTestCasesByProblemID(problemID)
}
