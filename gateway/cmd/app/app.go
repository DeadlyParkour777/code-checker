package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/DeadlyParkour777/code-checker/gateway/internal/cache"
	"github.com/DeadlyParkour777/code-checker/gateway/internal/config"
	"github.com/DeadlyParkour777/code-checker/gateway/internal/handler"
	authpb "github.com/DeadlyParkour777/code-checker/pkg/auth"
	problempb "github.com/DeadlyParkour777/code-checker/pkg/problem"
	resultpb "github.com/DeadlyParkour777/code-checker/pkg/result"
	submissionpb "github.com/DeadlyParkour777/code-checker/pkg/submission"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	httpServer *http.Server
}

func New(cfg config.Config) (*App, error) {
	log.Println("Initializing gRPC clients...")

	authConn, err := grpc.NewClient(cfg.AuthServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to auth service: %v", err)
	}
	authClient := authpb.NewAuthServiceClient(authConn)

	problemConn, err := grpc.NewClient(cfg.ProblemServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to problem service: %v", err)
	}
	problemClient := problempb.NewProblemServiceClient(problemConn)

	submissionConn, err := grpc.NewClient(cfg.SubmissionServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to submission service: %v", err)
	}
	submissionClient := submissionpb.NewSubmissionServiceClient(submissionConn)

	resultConn, err := grpc.NewClient(cfg.ResultServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to result service: %v", err)
	}
	resultClient := resultpb.NewResultServiceClient(resultConn)

	log.Println("gRPC clients initialized")

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	jwtCache := cache.NewRedisJWTCache(redisClient)
	log.Println("Redis cache initialized")

	httpHandler := handler.NewHandler(
		authClient,
		problemClient,
		submissionClient,
		resultClient,
		jwtCache,
	)
	log.Println("HTTP handler initialized")

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	server := &http.Server{
		Addr:    addr,
		Handler: httpHandler.Routes(),
	}

	return &App{httpServer: server}, nil
}

func (a *App) Run() error {
	log.Printf("Gateway HTTP server started on %s", a.httpServer.Addr)
	return a.httpServer.ListenAndServe()
}
