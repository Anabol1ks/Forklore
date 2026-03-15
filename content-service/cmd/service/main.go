package main

import (
	"content-service/config"
	"content-service/internal/repository"
	"os"

	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load("../.env")
	isDev := os.Getenv("ENV") == "development"
	if err := logger.Init(isDev); err != nil {
		panic(err)
	}
	defer logger.Sync()

	log := logger.L()
	cfg := config.Load(log)

	log.Info("CFG", zap.Any("cfg", cfg))

	db := database.ConnectDB(&cfg.DB.Config, log)
	defer database.CloseDB(db, log)

	repos := repository.New(db)
	_ = repos
}
