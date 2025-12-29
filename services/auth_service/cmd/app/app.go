package app

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	authpb "github.com/DeadlyParkour777/code-checker/pkg/auth"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/config"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/handler"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/service"
	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	grpcServer *grpc.Server
	db         *sql.DB
	cfg        *config.Config
}

func New(cfg *config.Config) (*App, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	log.Println("Successfully connected to PostgreSQL")

	userStore := store.NewStore(db)
	authService := service.NewService(userStore, cfg.JWTSecretKey, 15*time.Minute)
	grpcHandler := handler.NewGrpcHandler(authService)

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	return &App{
		grpcServer: grpcServer,
		db:         db,
		cfg:        cfg,
	}, nil
}

func (a *App) Run() error {
	defer a.db.Close()

	listenAddr := fmt.Sprintf(":%s", a.cfg.GRPCport)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		log.Printf("gRPC server started at %s", a.cfg.GRPCport)
		if err := a.grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	a.grpcServer.GracefulStop()
	log.Println("Server gracefully stopped")

	return nil
}
