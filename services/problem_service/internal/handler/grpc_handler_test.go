package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	problem_service "github.com/DeadlyParkour777/code-checker/pkg/problem"
	"github.com/DeadlyParkour777/code-checker/services/problem_service/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeService struct {
	createProblemFn  func(ctx context.Context, title, description string) (*types.Problem, error)
	getProblemFn     func(ctx context.Context, id string) (*types.Problem, error)
	listProblemsFn   func(ctx context.Context) ([]*types.Problem, error)
	createTestCaseFn func(ctx context.Context, problemID, input, output string) (*types.TestCase, error)
	getTestCasesFn   func(ctx context.Context, problemID string) ([]*types.TestCase, error)
}

func (f *fakeService) CreateProblem(ctx context.Context, title, description string) (*types.Problem, error) {
	if f.createProblemFn == nil {
		return nil, errors.New("CreateProblem not implemented")
	}
	return f.createProblemFn(ctx, title, description)
}

func (f *fakeService) GetProblem(ctx context.Context, id string) (*types.Problem, error) {
	if f.getProblemFn == nil {
		return nil, errors.New("GetProblem not implemented")
	}
	return f.getProblemFn(ctx, id)
}

func (f *fakeService) ListProblems(ctx context.Context) ([]*types.Problem, error) {
	if f.listProblemsFn == nil {
		return nil, errors.New("ListProblems not implemented")
	}
	return f.listProblemsFn(ctx)
}

func (f *fakeService) CreateTestCase(ctx context.Context, problemID, input, output string) (*types.TestCase, error) {
	if f.createTestCaseFn == nil {
		return nil, errors.New("CreateTestCase not implemented")
	}
	return f.createTestCaseFn(ctx, problemID, input, output)
}

func (f *fakeService) GetTestCases(ctx context.Context, problemID string) ([]*types.TestCase, error) {
	if f.getTestCasesFn == nil {
		return nil, errors.New("GetTestCases not implemented")
	}
	return f.getTestCasesFn(ctx, problemID)
}

func TestCreateProblem(t *testing.T) {
	fixedTime := time.Date(2024, 11, 1, 9, 0, 0, 0, time.UTC)
	service := &fakeService{
		createProblemFn: func(_ context.Context, title, description string) (*types.Problem, error) {
			return &types.Problem{ID: "p1", Title: title, Description: description, CreatedAt: fixedTime}, nil
		},
	}
	handler := NewGrpcHandler(service)

	resp, err := handler.CreateProblem(context.Background(), &problem_service.CreateProblemRequest{Title: "T", Description: "D"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetId() != "p1" {
		t.Fatalf("unexpected id: %s", resp.GetId())
	}
	if resp.GetCreatedAt() != fixedTime.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %s", resp.GetCreatedAt())
	}
}

func TestCreateProblem_Error(t *testing.T) {
	service := &fakeService{
		createProblemFn: func(_ context.Context, _, _ string) (*types.Problem, error) {
			return nil, errors.New("boom")
		},
	}
	handler := NewGrpcHandler(service)

	_, err := handler.CreateProblem(context.Background(), &problem_service.CreateProblemRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", status.Code(err))
	}
}

func TestGetProblem(t *testing.T) {
	fixedTime := time.Date(2024, 11, 2, 9, 0, 0, 0, time.UTC)
	service := &fakeService{
		getProblemFn: func(_ context.Context, id string) (*types.Problem, error) {
			return &types.Problem{ID: id, Title: "T", Description: "D", CreatedAt: fixedTime}, nil
		},
	}
	handler := NewGrpcHandler(service)

	resp, err := handler.GetProblem(context.Background(), &problem_service.GetProblemRequest{Id: "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetId() != "p2" {
		t.Fatalf("unexpected id: %s", resp.GetId())
	}
	if resp.GetCreatedAt() != fixedTime.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %s", resp.GetCreatedAt())
	}
}

func TestGetProblem_Error(t *testing.T) {
	service := &fakeService{
		getProblemFn: func(_ context.Context, _ string) (*types.Problem, error) {
			return nil, errors.New("not found")
		},
	}
	handler := NewGrpcHandler(service)

	_, err := handler.GetProblem(context.Background(), &problem_service.GetProblemRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", status.Code(err))
	}
}

func TestListProblems(t *testing.T) {
	fixedTime := time.Date(2024, 11, 3, 9, 0, 0, 0, time.UTC)
	service := &fakeService{
		listProblemsFn: func(_ context.Context) ([]*types.Problem, error) {
			return []*types.Problem{{ID: "p1", CreatedAt: fixedTime}, {ID: "p2", CreatedAt: fixedTime}}, nil
		},
	}
	handler := NewGrpcHandler(service)

	resp, err := handler.ListProblems(context.Background(), &problem_service.ListProblemsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetProblems()) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(resp.GetProblems()))
	}
}

func TestListProblems_Error(t *testing.T) {
	service := &fakeService{
		listProblemsFn: func(_ context.Context) ([]*types.Problem, error) {
			return nil, errors.New("db")
		},
	}
	handler := NewGrpcHandler(service)

	_, err := handler.ListProblems(context.Background(), &problem_service.ListProblemsRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %v", status.Code(err))
	}
}

func TestCreateTestCase(t *testing.T) {
	service := &fakeService{
		createTestCaseFn: func(_ context.Context, problemID, input, output string) (*types.TestCase, error) {
			return &types.TestCase{ID: "tc-1", ProblemID: problemID, Input: input, Output: output}, nil
		},
	}
	handler := NewGrpcHandler(service)

	resp, err := handler.CreateTestCase(context.Background(), &problem_service.CreateTestCaseRequest{
		ProblemId:  "p1",
		InputData:  "1 2",
		OutputData: "3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetId() != "tc-1" {
		t.Fatalf("unexpected id: %s", resp.GetId())
	}
}

func TestCreateTestCase_Error(t *testing.T) {
	service := &fakeService{
		createTestCaseFn: func(_ context.Context, _, _, _ string) (*types.TestCase, error) {
			return nil, errors.New("db")
		},
	}
	handler := NewGrpcHandler(service)

	_, err := handler.CreateTestCase(context.Background(), &problem_service.CreateTestCaseRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %v", status.Code(err))
	}
}

func TestGetTestCases(t *testing.T) {
	service := &fakeService{
		getTestCasesFn: func(_ context.Context, problemID string) ([]*types.TestCase, error) {
			return []*types.TestCase{{ID: "tc-1", ProblemID: problemID}}, nil
		},
	}
	handler := NewGrpcHandler(service)

	resp, err := handler.GetTestCases(context.Background(), &problem_service.GetTestCasesRequest{ProblemId: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetTestCases()) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(resp.GetTestCases()))
	}
}

func TestGetTestCases_Error(t *testing.T) {
	service := &fakeService{
		getTestCasesFn: func(_ context.Context, _ string) ([]*types.TestCase, error) {
			return nil, errors.New("db")
		},
	}
	handler := NewGrpcHandler(service)

	_, err := handler.GetTestCases(context.Background(), &problem_service.GetTestCasesRequest{ProblemId: "p1"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %v", status.Code(err))
	}
}
