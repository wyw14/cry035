package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment     string
	Address         string
	DatabaseURL     string
	AttachmentDir   string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	MaxUploadBytes  int64
	CORSOrigin      string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:     getEnv("APP_ENV", "development"),
		Address:         getEnv("HTTP_ADDRESS", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		AttachmentDir:   getEnv("ATTACHMENT_DIR", "./var/attachments"),
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		MaxUploadBytes:  10 << 20,
		CORSOrigin:      getEnv("CORS_ORIGIN", "http://localhost:5173"),
	}
	if value := os.Getenv("REQUEST_TIMEOUT"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse REQUEST_TIMEOUT: %w", err)
		}
		cfg.RequestTimeout = duration
	}
	if value := os.Getenv("MAX_UPLOAD_BYTES"); value != "" {
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil || limit <= 0 {
			return Config{}, fmt.Errorf("MAX_UPLOAD_BYTES must be a positive integer")
		}
		cfg.MaxUploadBytes = limit
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
