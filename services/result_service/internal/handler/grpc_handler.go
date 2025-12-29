package handler

import (
	"context"
	"time"

	resultpb "github.com/DeadlyParkour777/code-checker/pkg/result"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	resultpb.UnimplementedResultServiceServer
	service service.Service
}

func NewGrpcHandler(svc service.Service) *GrpcHandler {
	return &GrpcHandler{service: svc}
}

func (h *GrpcHandler) GetUserSubmissions(ctx context.Context, req *resultpb.GetUserSubmissionsRequest) (*resultpb.GetUserSubmissionsResponse, error) {
	submissions, err := h.service.GetUserSubmissions(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get submissions: %v", err)
	}

	pbSubmissions := make([]*resultpb.Submission, len(submissions))
	for i, sub := range submissions {
		pbSubmissions[i] = &resultpb.Submission{
			Id:        sub.ID,
			ProblemId: sub.ProblemID,
			UserId:    sub.UserID,
			Language:  sub.Language,
			Status:    sub.Status,
			CreatedAt: sub.CreatedAt.Format(time.RFC3339),
			UpdatedAt: sub.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &resultpb.GetUserSubmissionsResponse{Submissions: pbSubmissions}, nil
}
