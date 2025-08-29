package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBHost       string
	DBPort       int
	DBUser       string
	DBPassword   string
	DBName       string
	JWTSecretKey string
	GRPCport     string
}

func ConfigInit() *Config {
	// err := godotenv.Load()
	// if err != nil {
	// 	panic(err)
	// }

	return &Config{
		DBHost:       GetString("DB_HOST", "localhost"),
		DBPort:       GetInt("DB_PORT", 5433),
		DBUser:       GetString("DB_USER", "postgres"),
		DBPassword:   GetString("DB_PASSWORD", "admin"),
		DBName:       GetString("DB_NAME", "authdb"),
		JWTSecretKey: GetString("JWT_KEY", "secret"),
		GRPCport:     GetString("GRPC_PORT", "8001"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func GetString(key string, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	return v
}

func GetInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	vInt, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return vInt
}
