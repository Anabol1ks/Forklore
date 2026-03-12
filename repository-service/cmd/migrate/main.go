package main

import (
	"context"
	"os"
	"repository-service/config"
	migrations "repository-service/internal/migration"

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

	db := database.ConnectDB(&cfg.DB.Config, log)
	defer database.CloseDB(db, log)

	ctx := context.Background()

	if err := migrations.AutoMigrate(ctx, db, log); err != nil {
		log.Fatal("Ошибка при выполнении миграции", zap.Error(err))
	}
}
