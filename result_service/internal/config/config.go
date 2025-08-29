package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GRPCPort     string
	KafkaBrokers []string
	ResultTopic  string
	GroupID      string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func ConfigInit() Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	return Config{
		GRPCPort:      getEnv("GRPC_PORT", "8003"),
		KafkaBrokers:  strings.Split(getEnv("KAFKA_BROKERS", "kafka:29092"), ","),
		ResultTopic:   "results",
		GroupID:       "result-group",
		DBHost:        getEnv("DB_HOST", "auth-db"),
		DBPort:        "5432",
		DBUser:        "postgres",
		DBPassword:    "admin",
		DBName:        "authdb",
		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
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
