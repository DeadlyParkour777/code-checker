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

	KafkaBrokers       []string
	ProblemEventsTopic string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func ConfigInit() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	return Config{
		GRPCPort:           getEnv("GRPC_PORT", "8002"),
		KafkaBrokers:       strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ","),
		ProblemEventsTopic: getEnv("PROBLEM_EVENTS_TOPIC", "problem_events"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "admin"),
		DBName:             getEnv("DB_NAME", "authdb"),
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
