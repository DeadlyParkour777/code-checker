package app

import (
	"context"
	"log"
	"time"

	"github.com/DeadlyParkour777/code-checker/services/judge_service/internal/config"
	"github.com/DeadlyParkour777/code-checker/services/judge_service/internal/handler"
	"github.com/DeadlyParkour777/code-checker/services/judge_service/internal/service"
	"github.com/segmentio/kafka-go"
)

type App struct {
	kafkaReader *kafka.Reader
	handler     *handler.KafkaConsumer
}

func New(cfg config.Config) (*App, error) {
	log.Println("Initializing application components...")

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.SubmissionTopic,
		GroupID:        cfg.GroupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	log.Println("Kafka reader initialized")

	// dialer := &kafka.Dialer{
	// 	Timeout:   10 * time.Second,
	// 	DualStack: true,
	// }

	// transport := &kafka.Transport{
	// 	Dial: dialer.DialFunc,
	// 	SASL: plain.Mechanism{
	// 		Username: "",
	// 		Password: "",
	// 	},
	// }

	kafkaProducer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Topic:        cfg.ResultTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		MaxAttempts:  10,
		// Transport:    transport,
	}
	log.Println("Kafka producer initialized")

	appService := service.NewService(
		kafkaProducer,
		time.Duration(cfg.ExecutionTimeoutSeconds)*time.Second,
		cfg.HostTempPath,
		cfg.ProblemServiceAddr,
	)
	log.Println("Service layer initialized")

	kafkaHandler := handler.NewKafkaConsumer(appService)
	log.Println("Kafka handler initialized")

	return &App{
		kafkaReader: kafkaReader,
		handler:     kafkaHandler,
	}, nil
}

func (a *App) Run() error {
	defer a.kafkaReader.Close()

	log.Println("Judge service worker started. Waiting for submissions...")
	ctx := context.Background()

	for {
		msg, err := a.kafkaReader.FetchMessage(ctx)
		if err != nil {
			log.Printf("could not fetch message: %v", err)
			return err
		}

		a.handler.ProcessMessage(ctx, msg)

		if err := a.kafkaReader.CommitMessages(ctx, msg); err != nil {
			log.Printf("failed to commit message: %v", err)
		}
	}
}
