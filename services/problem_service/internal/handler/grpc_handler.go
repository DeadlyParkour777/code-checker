package handler

import (
	"context"
	"time"

	problem_service "github.com/DeadlyParkour777/code-checker/pkg/problem"
	"github.com/DeadlyParkour777/code-checker/services/problem_service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	problem_service.UnimplementedProblemServiceServer
	service service.Service
}

func NewGrpcHandler(service service.Service) *GrpcHandler {
	return &GrpcHandler{service: service}
}

func (h *GrpcHandler) CreateProblem(ctx context.Context, req *problem_service.CreateProblemRequest) (*problem_service.Problem, error) {
	problem, err := h.service.CreateProblem(ctx, req.GetTitle(), req.GetDescription())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create problem: %v", err)
	}

	return &problem_service.Problem{
		Id:          problem.ID,
		Title:       problem.Title,
		Description: problem.Description,
		CreatedAt:   problem.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *GrpcHandler) GetProblem(ctx context.Context, req *problem_service.GetProblemRequest) (*problem_service.Problem, error) {
	problem, err := h.service.GetProblem(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "problem not found: %v", err)
	}

	return &problem_service.Problem{
		Id:          problem.ID,
		Title:       problem.Title,
		Description: problem.Description,
		CreatedAt:   problem.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *GrpcHandler) ListProblems(ctx context.Context, req *problem_service.ListProblemsRequest) (*problem_service.ListProblemsResponse, error) {
	problems, err := h.service.ListProblems(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list problems: %v", err)
	}

	var pbProblems []*problem_service.Problem
	for _, problem := range problems {
		pbProblems = append(pbProblems, &problem_service.Problem{
			Id:          problem.ID,
			Title:       problem.Title,
			Description: problem.Description,
			CreatedAt:   problem.CreatedAt.Format(time.RFC3339),
		})
	}

	return &problem_service.ListProblemsResponse{Problems: pbProblems}, nil
}

func (h *GrpcHandler) CreateTestCase(ctx context.Context, req *problem_service.CreateTestCaseRequest) (*problem_service.TestCase, error) {
	testCase, err := h.service.CreateTestCase(ctx, req.GetProblemId(), req.GetInputData(), req.GetOutputData())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create test case: %v", err)
	}

	return &problem_service.TestCase{
		Id:         testCase.ID,
		ProblemId:  testCase.ProblemID,
		InputData:  testCase.Input,
		OutputData: testCase.Output,
	}, nil
}

func (h *GrpcHandler) GetTestCases(ctx context.Context, req *problem_service.GetTestCasesRequest) (*problem_service.GetTestCasesResponse, error) {
	testCases, err := h.service.GetTestCases(ctx, req.GetProblemId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get test cases: %v", err)
	}

	var pbTestCases []*problem_service.TestCase
	for _, tc := range testCases {
		pbTestCases = append(pbTestCases, &problem_service.TestCase{
			Id:         tc.ID,
			ProblemId:  tc.ProblemID,
			InputData:  tc.Input,
			OutputData: tc.Output,
		})
	}

	return &problem_service.GetTestCasesResponse{TestCases: pbTestCases}, nil
}
