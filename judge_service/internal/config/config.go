package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	KafkaBrokers            []string
	SubmissionTopic         string
	ResultTopic             string
	GroupID                 string
	ExecutionTimeoutSeconds int
	HostTempPath            string
	ProblemServiceAddr      string
}

func ConfigInit() Config {
	// godotenv.Load()

	brokersStr := getEnv("KAFKA_BROKERS", "localhost:9092")

	timeout, _ := strconv.Atoi(getEnv("EXECUTION_TIMEOUT_SECONDS", "2"))

	return Config{
		KafkaBrokers:            strings.Split(brokersStr, ","),
		SubmissionTopic:         getEnv("SUBMISSION_TOPIC", "submissions"),
		ResultTopic:             getEnv("RESULT_TOPIC", "results"),
		GroupID:                 getEnv("GROUP_ID", "judge-group"),
		ExecutionTimeoutSeconds: timeout,
		HostTempPath:            getEnv("HOST_TEMP_PATH", "/tmp/submissions"),
		ProblemServiceAddr:      getEnv("PROBLEM_SERVICE_ADDR", "problem-service:8002"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
