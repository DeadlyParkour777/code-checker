package handler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/DeadlyParkour777/code-checker/result_service/internal/service"
	"github.com/DeadlyParkour777/code-checker/result_service/internal/types"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	service service.Service
}

func NewKafkaConsumer(svc service.Service) *KafkaConsumer {
	return &KafkaConsumer{service: svc}
}

func (h *KafkaConsumer) Start(ctx context.Context, reader *kafka.Reader) {
	log.Println("Kafka consumer worker started. Waiting for results...")
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("could not fetch message: %v", err)
			if ctx.Err() != nil {
				break
			}
			continue
		}

		var result types.ResultEvent
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			log.Printf("failed to unmarshal result event, skipping: %v", err)
			reader.CommitMessages(ctx, msg)
			continue
		}

		h.service.ProcessResult(ctx, &result)

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("failed to commit message: %v", err)
		}
	}
}
