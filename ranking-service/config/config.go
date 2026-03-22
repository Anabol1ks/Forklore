package config

import (
	"os"
	"strings"

	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"go.uber.org/zap"
)

type Config struct {
	Port  string
	DB    DB
	Auth  Auth
	Kafka Kafka
}

type DB struct {
	database.Config
}

type Auth struct {
	JWTSecret string
}

type Kafka struct {
	Brokers     []string
	Topic       string
	GroupID     string
	AuthTopic   string
	AuthGroupID string
	MinBytes    int
	MaxBytes    int
}

func Load(log *zap.Logger) *Config {
	return &Config{
		Port: getEnvRequired("RANKING_PORT", log),
		DB: DB{Config: database.Config{
			Host:     getEnvRequired("DB_RANKING_HOST", log),
			Port:     getEnvRequired("DB_RANKING_PORT", log),
			User:     getEnvRequired("DB_RANKING_USER", log),
			Password: getEnvRequired("DB_RANKING_PASSWORD", log),
			Name:     getEnvRequired("DB_RANKING_NAME", log),
			SSLMode:  getEnvRequired("DB_RANKING_SSLMODE", log),
		}},
		Auth: Auth{
			JWTSecret: getEnvRequired("JWTSecret", log),
		},
		Kafka: Kafka{
			Brokers:     splitAndTrim(getEnvRequired("KAFKA_BROKERS", log)),
			Topic:       getEnvDefault("KAFKA_RANKING_TOPIC", "ranking.events.v1"),
			GroupID:     getEnvDefault("KAFKA_RANKING_GROUP_ID", "ranking-service"),
			AuthTopic:   getEnvDefault("KAFKA_AUTH_TOPIC", "forklore.auth.events.v1"),
			AuthGroupID: getEnvDefault("KAFKA_RANKING_AUTH_GROUP_ID", "ranking-service-auth"),
			MinBytes:    1,
			MaxBytes:    10e6,
		},
	}
}

func getEnvRequired(key string, log *zap.Logger) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Error("required environment variable is missing", zap.String("key", key))
	panic("missing required environment variable: " + key)
}

func getEnvDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
