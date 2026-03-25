package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"profile-service/config"
	"profile-service/internal/kafka"
	"profile-service/internal/repository"
	"profile-service/internal/service"
	grpcserver "profile-service/internal/transport/grpc"
	"syscall"
	"time"

	profilev1 "github.com/Anabol1ks/Forklore/pkg/pb/profile/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repos := repository.New(db)
	rankingProducer := kafka.NewRankingProducer(kafka.RankingProducerConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.RankingTopic,
	}, log)
	defer func() {
		if err := rankingProducer.Close(); err != nil {
			log.Warn("failed to close ranking producer", zap.Error(err))
		}
	}()

	profileService := service.NewProfileService(
		repos,
		"comer",
	)

	kafkaHandler := kafka.NewHandler(profileService, log)

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:         cfg.Kafka.Brokers,
		Topic:           cfg.Kafka.AuthTopic,
		GroupID:         cfg.Kafka.GroupID,
		MinBytes:        1,
		MaxBytes:        10e6,
		MaxWait:         2 * time.Second,
		CommitInterval:  time.Second,
		ReadLagInterval: -1,
		StartOffset:     0,
		HandleTimeout:   10 * time.Second,
	}, kafkaHandler, log)

	profileHandler := grpcserver.NewProfileHandler(profileService, rankingProducer, log)

	tokenManager := authjwt.NewJWTVerifier(cfg.Auth.JWTSecret)
	authInterceptor := grpcserver.NewAuthInterceptor(tokenManager, log)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor.UnaryServerInterceptor(),
		),
	)

	profilev1.RegisterProfileServiceServer(grpcServer, profileHandler)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	addr := ":" + cfg.Port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("failed to listen", zap.String("addr", addr), zap.Error(err))
	}

	errCh := make(chan error, 2)

	go func() {
		log.Info("gRPC server started", zap.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server stopped with error", zap.Error(err))
		}
	}()

	go func() {
		if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			log.Error("service runtime error", zap.Error(err))
		}
	}

	stop()
	log.Info("shutting down services")
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	if err := consumer.Close(); err != nil {
		log.Error("failed to close kafka consumer", zap.Error(err))
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Info("gRPC server stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Warn("gRPC graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}

	log.Info("profile-service stopped")
}
