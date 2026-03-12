package main

import (
	"auth-service/config"
	"auth-service/internal/repository"
	"auth-service/internal/security"
	"auth-service/internal/service"
	grpcserver "auth-service/internal/transport/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
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

	repos := repository.New(db)

	passwordManager := security.NewBcryptPasswordManager(cfg.Auth.BcryptCost)
	tokenManager := security.NewJWTTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTIssuer,
		cfg.Auth.AccessTokenTTL,
	)

	authService := service.NewAuthService(repos, passwordManager, tokenManager, cfg.Auth.RefreshTokenTTL)

	authHandler := grpcserver.NewAuthHandler(authService)
	loggingInterceptor := grpcserver.NewLoggingInterceptor(log)
	authInterceptor := grpcserver.NewAuthInterceptor(tokenManager)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor.UnaryServerInterceptor(),
			authInterceptor.UnaryServerInterceptor(),
		),
	)

	authv1.RegisterAuthServiceServer(grpcServer, authHandler)

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
