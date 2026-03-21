package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"go.uber.org/zap"
)

type Config struct {
	Port  string
	DB    DB
	Auth  Auth
	Kafka KafkaConfig
}

type DB struct {
	database.Config
}

type Auth struct {
	JWTSecret string
}

type KafkaConfig struct {
	Brokers   []string
	AuthTopic string
	GroupID   string
}

func Load(log *zap.Logger) *Config {
	return &Config{
		Port: getEnv("PROFILE_PORT", log),
		DB: DB{
			Config: database.Config{
				Host:     getEnv("DB_PROFILE_HOST", log),
				Port:     getEnv("DB_PROFILE_PORT", log),
				User:     getEnv("DB_PROFILE_USER", log),
				Password: getEnv("DB_PROFILE_PASSWORD", log),
				Name:     getEnv("DB_PROFILE_NAME", log),
				SSLMode:  getEnv("DB_PROFILE_SSLMODE", log),
			},
		},
		Auth: Auth{
			JWTSecret: getEnv("JWTSecret", log),
		},
		Kafka: KafkaConfig{
			Brokers:   splitAndTrim(getEnv("KAFKA_BROKERS", log)),
			AuthTopic: getEnv("KAFKA_PROFILE_TOPIC", log),
			GroupID:   getEnv("KAFKA_PROFILE_GROUP_ID", log),
		},
	}
}

func getEnv(key string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	log.Error("Обязательная переменная окружения не установлена", zap.String("key", key))
	panic("missing required environment variable: " + key)
}

func parseSecondsDuration(s string) time.Duration {
	if strings.HasSuffix(s, "s") {
		secondsStr := strings.TrimSuffix(s, "s")
		seconds, err := strconv.Atoi(secondsStr)
		if err != nil {
			log.Printf("Ошибка парсинга длительности: %v", err)
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Ошибка парсинга длительности: %v", err)
		return 0
	}
	return duration
}

func parseDurationWithDays(s string) time.Duration {
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := time.ParseDuration(daysStr + "h")
		if err != nil {
			log.Printf("Ошибка парсинга TTL: %v", err)
			return 0
		}
		return time.Duration(24) * days
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return duration
}

func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	for _, p := range strings.Split(s, ",") {
		pt := strings.TrimSpace(p)
		if pt != "" {
			parts = append(parts, pt)
		}
	}
	return parts
}
