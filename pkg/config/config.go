package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPass     string
	PostgresDatabase string
	PostgresSSLMode  string
	JWTSecret        string

	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func Load(envFiles ...string) (*Config, error) {
	if len(envFiles) > 0 {
		if err := godotenv.Load(envFiles...); err != nil {
			slog.Warn("env file not found", "files", envFiles)
		}
	} else {
		if err := godotenv.Load(); err != nil {
			slog.Warn("env file not found, using system environment variables")
		}
	}

	portValue := getEnvWithDefault("PORT", "8080")
	maxConns, _ := strconv.Atoi(getEnvWithDefault("DB_MAX_CONNS", "25"))
	minConns, _ := strconv.Atoi(getEnvWithDefault("DB_MIN_CONNS", "5"))
	postgresSSL := getEnvWithDefault("POSTGRES_SSL_MODE", "disable")

	postgresHost, err := getEnvRequired("POSTGRES_HOST")
	if err != nil {
		return nil, err
	}
	postgresPort, err := getEnvRequired("POSTGRES_PORT")
	if err != nil {
		return nil, err
	}
	postgresUser, err := getEnvRequired("POSTGRES_USER")
	if err != nil {
		return nil, err
	}
	postgresPass, err := getEnvRequired("POSTGRES_PASSWORD")
	if err != nil {
		return nil, err
	}
	postgresDB, err := getEnvRequired("POSTGRES_DB")
	if err != nil {
		return nil, err
	}
	jwtSecret, err := getEnvRequired("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:              portValue,
		JWTSecret:         jwtSecret,
		PostgresHost:      postgresHost,
		PostgresPort:      postgresPort,
		PostgresUser:      postgresUser,
		PostgresPass:      postgresPass,
		PostgresDatabase:  postgresDB,
		PostgresSSLMode:   postgresSSL,
		MaxConns:          int32(maxConns),
		MinConns:          int32(minConns),
		MaxConnLifetime:   getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime:   getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		HealthCheckPeriod: getEnvAsDuration("DB_HEALTH_CHECK_PERIOD", time.Minute),
	}

	slog.Info("configuration loaded", "port", cfg.Port, "db_host", cfg.PostgresHost)

	return cfg, nil
}

// for variables with default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// for required variables
func getEnvRequired(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return duration
}
