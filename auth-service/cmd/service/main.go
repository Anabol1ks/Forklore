package main

import (
	"auth-service/config"
	"auth-service/internal/repository"
	"auth-service/internal/security"
	"os"

	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"github.com/joho/godotenv"
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

	repo := repository.New(db)
	_ = repo

	passwordManager := security.NewBcryptPasswordManager(cfg.Auth.BcryptCost)
	tokenManager := security.NewJWTTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTIssuer,
		cfg.Auth.AccessTokenTTL,
	)

	_ = passwordManager
	_ = tokenManager
}
