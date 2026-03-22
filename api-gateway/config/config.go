package config

import (
	"os"

	"go.uber.org/zap"
)

type Config struct {
	Port                  string
	AuthServiceAddr       string
	RepositoryServiceAddr string
	ContentServiceAddr    string
	SearchServiceAddr     string
	ProfileServiceAddr    string
}

func Load(log *zap.Logger) *Config {
	return &Config{
		Port:                  getEnv("GATEWAY_PORT", "8080", log),
		AuthServiceAddr:       getEnv("AUTH_SERVICE_ADDR", "localhost:8081", log),
		RepositoryServiceAddr: getEnv("REPOSITORY_SERVICE_ADDR", "localhost:8082", log),
		ContentServiceAddr:    getEnv("CONTENT_SERVICE_ADDR", "localhost:8083", log),
		SearchServiceAddr:     getEnv("SEARCH_SERVICE_ADDR", "localhost:8084", log),
		ProfileServiceAddr:    getEnv("PROFILE_SERVICE_ADDR", "localhost:8085", log),
	}
}

func getEnv(key, fallback string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	log.Warn("Переменная окружения не установлена, используется значение по умолчанию",
		zap.String("key", key),
		zap.String("default", fallback),
	)
	return fallback
}
