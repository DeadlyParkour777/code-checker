package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"sync"

	resultpb "github.com/DeadlyParkour777/code-checker/pkg/result"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/config"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/handler"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/service"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/store"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	grpcServer   *grpc.Server
	kafkaReader  *kafka.Reader
	kafkaHandler *handler.KafkaConsumer
	db           *sql.DB
	cfg          config.Config
}

func New(cfg config.Config) (*App, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	log.Println("Successfully connected to PostgreSQL")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	log.Println("Successfully connected to Redis")

	appStore := store.NewStore(db, redisClient)
	appService := service.NewService(appStore)

	grpcHandler := handler.NewGrpcHandler(appService)
	grpcServer := grpc.NewServer()
	resultpb.RegisterResultServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.ResultTopic,
		GroupID: cfg.GroupID,
	})
	kafkaHandler := handler.NewKafkaConsumer(appService)

	return &App{
		grpcServer:   grpcServer,
		kafkaReader:  kafkaReader,
		kafkaHandler: kafkaHandler,
		db:           db,
		cfg:          cfg,
	}, nil
}

func (a *App) Run() error {
	defer a.db.Close()
	defer a.kafkaReader.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		listenAddr := fmt.Sprintf(":%s", a.cfg.GRPCPort)

		lis, err := net.Listen("tcp", listenAddr)
		if err != nil {
			log.Fatalf("gRPC failed to listen: %v", err)
		}
		log.Printf("gRPC server started on %s", listenAddr)
		if err := a.grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	wg.Add(2)
	go func() {
		defer wg.Done()
		a.kafkaHandler.Start(context.Background(), a.kafkaReader)
	}()

	log.Println("Result service is running...")
	wg.Wait()
	log.Println("Result service is shutting down...")
	return nil
}
