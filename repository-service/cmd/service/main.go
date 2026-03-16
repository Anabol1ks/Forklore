package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	migrations "repository-service/internal/migration"
	"syscall"

	"repository-service/config"
	"repository-service/internal/repository"
	"repository-service/internal/service"
	grpcserver "repository-service/internal/transport/grpc"

	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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

	// Keep runtime schema in sync for safe zero-downtime field additions.
	if err := migrations.AutoMigrate(context.Background(), db, log); err != nil {
		log.Fatal("failed to auto-migrate repository database", zap.Error(err))
	}

	repos := repository.New(db)
	repoService := service.NewRepositoryService(repos)
	repoHandler := grpcserver.NewRepositoryHandler(repoService, log)

	tokenManager := authjwt.NewJWTVerifier(cfg.Auth.JWTSecret)
	authInterceptor := grpcserver.NewAuthInterceptor(tokenManager, log)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor.UnaryServerInterceptor(),
		),
	)

	repositoryv1.RegisterRepositoryServiceServer(grpcServer, repoHandler)

	// Health check — стандартный gRPC health protocol (используется load balancer'ами и Kubernetes)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Server reflection — позволяет grpcurl/bloomrpc/Postman обнаруживать сервисы (только в dev)
	if isDev {
		reflection.Register(grpcServer)
		log.Info("gRPC reflection enabled")
	}

	addr := ":" + cfg.Port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("failed to listen", zap.String("addr", addr), zap.Error(err))
	}

	go func() {
		log.Info("gRPC server started", zap.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server stopped with error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down gRPC server...")
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	grpcServer.GracefulStop()
	log.Info("gRPC server stopped")
}
