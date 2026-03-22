package main

import (
	"auth-service/config"
	"auth-service/internal/kafka"
	model "auth-service/internal/models"
	"auth-service/internal/repository"
	"auth-service/internal/security"
	"auth-service/internal/service"
	grpcserver "auth-service/internal/transport/grpc"
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"github.com/google/uuid"
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

	eventProducer := kafka.NewProducer(kafka.ProducerConfig{
		Brokers:   cfg.Kafka.Brokers,
		AuthTopic: cfg.Kafka.AuthTopic,
		ClientID:  cfg.Kafka.ClientID,
	}, log)
	defer func() {
		if err := eventProducer.Close(); err != nil {
			log.Warn("failed to close kafka producer", zap.Error(err))
		}
	}()

	bootstrapCtx, cancelBootstrap := context.WithTimeout(context.Background(), 20*time.Second)
	if err := publishBootstrapUserRegisteredEvents(bootstrapCtx, repos, eventProducer, log); err != nil {
		log.Warn("bootstrap publish for existing users failed", zap.Error(err))
	}
	cancelBootstrap()

	authHandler := grpcserver.NewAuthHandler(authService, eventProducer, log)
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

func publishBootstrapUserRegisteredEvents(
	ctx context.Context,
	repos *repository.Repository,
	producer grpcserver.UserRegisteredPublisher,
	log *zap.Logger,
) error {
	if repos == nil || producer == nil {
		return nil
	}

	users, err := repos.User.List(ctx)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		log.Info("no users found for bootstrap profile events")
		return nil
	}

	published := 0
	for _, user := range users {
		if !shouldPublishBootstrapEvent(user) {
			continue
		}

		if err := producer.PublishUserRegistered(ctx, user.ID, user.Username, user.Email); err != nil {
			log.Warn("failed to publish bootstrap user.registered event",
				zap.String("user_id", user.ID.String()),
				zap.String("username", user.Username),
				zap.Error(err),
			)
			continue
		}

		published++
	}

	log.Info("bootstrap user.registered publish completed",
		zap.Int("total_users", len(users)),
		zap.Int("published", published),
	)

	return nil
}

func shouldPublishBootstrapEvent(user *model.User) bool {
	if user == nil {
		return false
	}

	if user.ID == uuid.Nil {
		return false
	}

	if strings.TrimSpace(user.Username) == "" || strings.TrimSpace(user.Email) == "" {
		return false
	}

	return true
}
