package app

import (
	"database/sql"
	"fmt"
	"log"
	"net"

	submission_service "github.com/DeadlyParkour777/code-checker/pkg/submission"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/config"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/handler"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/service"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/store"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	cfg config.Config

	grpcServer  *grpc.Server
	db          *sql.DB
	kafkaWriter *kafka.Writer
}

func New(cfg config.Config) (*App, error) {
	log.Println("Initializing submission service...")

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	log.Println("Successfully connected to PostgreSQL")

	kafkaProducer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      cfg.KafkaBrokers,
		Topic:        cfg.SubmissionsTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: int(kafka.RequireOne),
	})
	log.Println("Kafka producer initialized")

	appStore := store.NewStore(db)
	appService, err := service.NewService(
		appStore,
		cfg.ProblemServiceAddr,
		kafkaProducer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	grpcHandler := handler.NewGrpcHandler(appService)

	grpcServer := grpc.NewServer()
	submission_service.RegisterSubmissionServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	return &App{
		cfg:         cfg,
		grpcServer:  grpcServer,
		db:          db,
		kafkaWriter: kafkaProducer,
	}, nil
}

func (a *App) Run() error {
	defer a.db.Close()
	defer a.kafkaWriter.Close()

	listenAddr := fmt.Sprintf(":%s", a.cfg.GRPCPort)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("gRPC server started on port %s", listenAddr)
	if err := a.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}
	return nil
}
