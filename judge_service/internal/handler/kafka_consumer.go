package handler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/DeadlyParkour777/code-checker/judge_service/internal/service"
	"github.com/DeadlyParkour777/code-checker/judge_service/internal/types"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	service service.Service
}

func NewKafkaConsumer(svc service.Service) *KafkaConsumer {
	return &KafkaConsumer{service: svc}
}

func (h *KafkaConsumer) ProcessMessage(ctx context.Context, msg kafka.Message) {
	var submission types.SubmissionEvent
	if err := json.Unmarshal(msg.Value, &submission); err != nil {
		log.Printf("Failed to unmarshal submission, skipping message: %v", err)
		return
	}

	h.service.ProcessSubmission(ctx, &submission)
}
