package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DeadlyParkour777/code-checker/services/problem_service/internal/types"
	"github.com/segmentio/kafka-go"
)

type fakeStore struct {
	createProblemFn         func(problem *types.Problem) (*types.Problem, error)
	getProblemFn            func(id string) (*types.Problem, error)
	listProblemsFn          func() ([]*types.Problem, error)
	createTestCaseFn        func(testCase *types.TestCase) (*types.TestCase, error)
	getTestCasesByProblemFn func(problemID string) ([]*types.TestCase, error)
}

func (f *fakeStore) CreateProblem(problem *types.Problem) (*types.Problem, error) {
	if f.createProblemFn == nil {
		return nil, errors.New("CreateProblem not implemented")
	}
	return f.createProblemFn(problem)
}

func (f *fakeStore) GetProblem(id string) (*types.Problem, error) {
	if f.getProblemFn == nil {
		return nil, errors.New("GetProblem not implemented")
	}
	return f.getProblemFn(id)
}

func (f *fakeStore) ListProblems() ([]*types.Problem, error) {
	if f.listProblemsFn == nil {
		return nil, errors.New("ListProblems not implemented")
	}
	return f.listProblemsFn()
}

func (f *fakeStore) CreateTestCase(testCase *types.TestCase) (*types.TestCase, error) {
	if f.createTestCaseFn == nil {
		return nil, errors.New("CreateTestCase not implemented")
	}
	return f.createTestCaseFn(testCase)
}

func (f *fakeStore) GetTestCasesByProblemID(problemID string) ([]*types.TestCase, error) {
	if f.getTestCasesByProblemFn == nil {
		return nil, errors.New("GetTestCasesByProblemID not implemented")
	}
	return f.getTestCasesByProblemFn(problemID)
}

type fakeWriter struct {
	messages []kafka.Message
	err      error
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.messages = append(f.messages, msgs...)
	return f.err
}

func TestCreateProblem_SuccessEmitsEvent(t *testing.T) {
	fixedTime := time.Date(2024, 12, 1, 10, 30, 0, 0, time.UTC)
	store := &fakeStore{
		createProblemFn: func(problem *types.Problem) (*types.Problem, error) {
			if problem.Title != "Two Sum" {
				t.Fatalf("unexpected title: %s", problem.Title)
			}
			if problem.Description != "Find indices" {
				t.Fatalf("unexpected description: %s", problem.Description)
			}
			problem.ID = "problem-1"
			problem.CreatedAt = fixedTime
			return problem, nil
		},
	}
	writer := &fakeWriter{}
	service := NewService(store, "problem_events", writer)

	created, err := service.CreateProblem(context.Background(), "Two Sum", "Find indices")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "problem-1" {
		t.Fatalf("unexpected id: %s", created.ID)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("expected 1 kafka message, got %d", len(writer.messages))
	}
	if writer.messages[0].Topic != "problem_events" {
		t.Fatalf("unexpected topic: %s", writer.messages[0].Topic)
	}

	var event types.ProblemEvent
	if err := json.Unmarshal(writer.messages[0].Value, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if event.EventType != "created" {
		t.Fatalf("unexpected event type: %s", event.EventType)
	}
	if event.Problem == nil || event.Problem.ID != "problem-1" {
		t.Fatalf("unexpected problem in event")
	}
}

func TestCreateProblem_StoreError(t *testing.T) {
	store := &fakeStore{
		createProblemFn: func(problem *types.Problem) (*types.Problem, error) {
			return nil, errors.New("db down")
		},
	}
	writer := &fakeWriter{}
	service := NewService(store, "problem_events", writer)

	_, err := service.CreateProblem(context.Background(), "Title", "Desc")
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(writer.messages) != 0 {
		t.Fatalf("expected no kafka messages on failure")
	}
}

func TestCreateProblem_KafkaErrorIgnored(t *testing.T) {
	store := &fakeStore{
		createProblemFn: func(problem *types.Problem) (*types.Problem, error) {
			problem.ID = "problem-2"
			problem.CreatedAt = time.Now().UTC()
			return problem, nil
		},
	}
	writer := &fakeWriter{err: errors.New("kafka down")}
	service := NewService(store, "problem_events", writer)

	created, err := service.CreateProblem(context.Background(), "Title", "Desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.ID != "problem-2" {
		t.Fatalf("unexpected created problem")
	}
	if len(writer.messages) != 1 {
		t.Fatalf("expected kafka write attempted")
	}
}

func TestGetProblem(t *testing.T) {
	store := &fakeStore{
		getProblemFn: func(id string) (*types.Problem, error) {
			if id != "problem-3" {
				t.Fatalf("unexpected id: %s", id)
			}
			return &types.Problem{ID: id}, nil
		},
	}
	service := NewService(store, "topic", &fakeWriter{})

	problem, err := service.GetProblem(context.Background(), "problem-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if problem.ID != "problem-3" {
		t.Fatalf("unexpected problem id: %s", problem.ID)
	}
}

func TestListProblems(t *testing.T) {
	store := &fakeStore{
		listProblemsFn: func() ([]*types.Problem, error) {
			return []*types.Problem{{ID: "p1"}, {ID: "p2"}}, nil
		},
	}
	service := NewService(store, "topic", &fakeWriter{})

	problems, err := service.ListProblems(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(problems))
	}
}

func TestCreateTestCase(t *testing.T) {
	store := &fakeStore{
		createTestCaseFn: func(testCase *types.TestCase) (*types.TestCase, error) {
			if testCase.ProblemID != "problem-4" {
				t.Fatalf("unexpected problem id: %s", testCase.ProblemID)
			}
			if testCase.Input != "1 2" || testCase.Output != "3" {
				t.Fatalf("unexpected test case data")
			}
			testCase.ID = "tc-1"
			return testCase, nil
		},
	}
	service := NewService(store, "topic", &fakeWriter{})

	created, err := service.CreateTestCase(context.Background(), "problem-4", "1 2", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "tc-1" {
		t.Fatalf("unexpected test case id: %s", created.ID)
	}
}

func TestGetTestCases(t *testing.T) {
	store := &fakeStore{
		getTestCasesByProblemFn: func(problemID string) ([]*types.TestCase, error) {
			if problemID != "problem-5" {
				t.Fatalf("unexpected problem id: %s", problemID)
			}
			return []*types.TestCase{{ID: "tc-1"}}, nil
		},
	}
	service := NewService(store, "topic", &fakeWriter{})

	cases, err := service.GetTestCases(context.Background(), "problem-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cases) != 1 || cases[0].ID != "tc-1" {
		t.Fatalf("unexpected test cases")
	}
}
