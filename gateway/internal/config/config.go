package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string

	AuthServiceAddr       string
	ProblemServiceAddr    string
	SubmissionServiceAddr string
	ResultServiceAddr     string

	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func ConfigInit() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	return Config{
		HTTPPort:              getEnv("HTTP_PORT", "8000"),
		AuthServiceAddr:       getEnv("AUTH_SERVICE_ADDR", "auth-service:8001"),
		ProblemServiceAddr:    getEnv("PROBLEM_SERVICE_ADDR", "problem-service:8002"),
		SubmissionServiceAddr: getEnv("SUBMISSION_SERVICE_ADDR", "submission-service:8004"),
		ResultServiceAddr:     getEnv("RESULT_SERVICE_ADDR", "result-service:8003"),
		RedisAddr:             getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		RedisDB:               redisDB,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
