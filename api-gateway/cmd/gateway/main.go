package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/config"
	_ "api-gateway/docs"
	"api-gateway/internal/clients"
	"api-gateway/internal/handlers"
	"api-gateway/internal/router"

	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

//	@title			Forklore API
//	@version		1.0
//	@description	API Gateway для микросервисов Forklore

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Введите "Bearer" и через пробел JWT access токен.

func main() {
	_ = godotenv.Load("../.env")
	isDev := os.Getenv("ENV") == "development"
	if err := logger.Init(isDev); err != nil {
		panic(err)
	}
	defer logger.Sync()

	log := logger.L()
	cfg := config.Load(log)

	// ── gRPC clients ──
	authClient, err := clients.NewAuthClient(cfg.AuthServiceAddr)
	if err != nil {
		log.Fatal("failed to connect to auth-service", zap.Error(err))
	}
	defer authClient.Close()

	repositoryClient, err := clients.NewRepositoryClient(cfg.RepositoryServiceAddr)
	if err != nil {
		log.Fatal("failed to connect to repository-service", zap.Error(err))
	}
	defer repositoryClient.Close()

	// ── Handlers ──
	authHandler := handlers.NewAuthHandler(authClient)
	repositoryHandler := handlers.NewRepositoryHandler(repositoryClient)

	// ── Router ──
	r := router.Setup(log, authHandler, repositoryHandler)

	// ── HTTP server ──
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("API Gateway started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	// ── Graceful shutdown ──
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down API Gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	log.Info("API Gateway stopped")
}
