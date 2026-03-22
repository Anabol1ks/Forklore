package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"ranking-service/config"
	"ranking-service/internal/kafka"
	"ranking-service/internal/repository"
	"ranking-service/internal/service"
	grpcserver "ranking-service/internal/transport/grpc"
	"syscall"
	"time"

	rankingv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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
	rankingService := service.NewRankingService(repos, log)
	rankingHandler := grpcserver.NewRankingHandler(rankingService, log)

	rankingKafkaHandler := kafka.NewRankingHandler(rankingService, log)
	authKafkaHandler := kafka.NewAuthHandler(rankingService, log)

	rankingConsumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:         cfg.Kafka.Brokers,
		Topic:           cfg.Kafka.Topic,
		GroupID:         cfg.Kafka.GroupID,
		MinBytes:        cfg.Kafka.MinBytes,
		MaxBytes:        cfg.Kafka.MaxBytes,
		MaxWait:         2 * time.Second,
		CommitInterval:  time.Second,
		ReadLagInterval: -1,
		StartOffset:     0,
	}, rankingKafkaHandler, log)

	authConsumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:         cfg.Kafka.Brokers,
		Topic:           cfg.Kafka.AuthTopic,
		GroupID:         cfg.Kafka.AuthGroupID,
		MinBytes:        cfg.Kafka.MinBytes,
		MaxBytes:        cfg.Kafka.MaxBytes,
		MaxWait:         2 * time.Second,
		CommitInterval:  time.Second,
		ReadLagInterval: -1,
		StartOffset:     0,
	}, authKafkaHandler, log)

	grpcServer := grpc.NewServer()
	rankingv1.RegisterRankingServiceServer(grpcServer, rankingHandler)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	addr := ":" + cfg.Port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("failed to listen", zap.String("addr", addr), zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 3)

	go func() {
		log.Info("ranking gRPC server started", zap.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	go func() {
		if err := rankingConsumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	go func() {
		if err := authConsumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
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
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	if err := rankingConsumer.Close(); err != nil {
		log.Error("failed to close ranking kafka consumer", zap.Error(err))
	}
	if err := authConsumer.Close(); err != nil {
		log.Error("failed to close auth kafka consumer", zap.Error(err))
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

	log.Info("ranking-service stopped")
}
