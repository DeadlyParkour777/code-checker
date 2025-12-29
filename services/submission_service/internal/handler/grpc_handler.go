package handler

import (
	"bytes"
	"io"
	"log"
	"time"

	submission_service "github.com/DeadlyParkour777/code-checker/pkg/submission"
	"github.com/DeadlyParkour777/code-checker/services/submission_service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	submission_service.UnimplementedSubmissionServiceServer
	service service.Service
}

func NewGrpcHandler(service service.Service) *GrpcHandler {
	return &GrpcHandler{service: service}
}

func (h *GrpcHandler) CreateSubmission(stream submission_service.SubmissionService_CreateSubmissionServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive submission info: %v", err)
	}

	info := req.GetInfo()
	if info == nil {
		return status.Errorf(codes.InvalidArgument, "first message must be submission info")
	}

	log.Printf("Received submission info for user %s, problem %s", info.GetUserId(), info.GetProblemId())

	var codeData bytes.Buffer
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive code chunk: %v", err)
		}

		chunk := req.GetChunkData()
		if _, err := codeData.Write(chunk); err != nil {
			return status.Errorf(codes.Internal, "failed to write code chunk to buffer: %v", err)
		}
	}

	log.Printf("Received %d bytes of code", codeData.Len())

	submission, err := h.service.CreateSubmission(
		stream.Context(),
		info.GetUserId(),
		info.GetProblemId(),
		codeData.String(),
		info.GetLanguage(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create submission: %v", err)
	}

	resp := &submission_service.Submission{
		Id:        submission.ID,
		ProblemId: submission.ProblemID,
		UserId:    submission.UserID,
		Code:      submission.Code,
		Language:  submission.Language,
		Status:    submission.Status,
		CreatedAt: submission.CreatedAt.Format(time.RFC3339),
		UpdatedAt: submission.UpdatedAt.Format(time.RFC3339),
	}

	return stream.SendAndClose(resp)
}
