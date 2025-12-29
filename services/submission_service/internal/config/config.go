package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPCPort string

	KafkaBrokers     []string
	SubmissionsTopic string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	AuthServiceAddr    string
	ProblemServiceAddr string
}

func ConfigInit() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	return Config{
		GRPCPort:           getEnv("GRPC_PORT", "8004"),
		KafkaBrokers:       strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ","),
		SubmissionsTopic:   getEnv("SUBMISSIONS_TOPIC", "submissions"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "admin"),
		DBName:             getEnv("DB_NAME", "authdb"),
		AuthServiceAddr:    getEnv("AUTH_SERVICE_ADDR", "auth-service:8001"),
		ProblemServiceAddr: getEnv("PROBLEM_SERVICE_ADDR", "problem-service:8002"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
