package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"search-service/config"
	"search-service/internal/repository"
	"search-service/internal/service"
	grpcserver "search-service/internal/transport/grpc"
	"syscall"
	"time"

	"search-service/internal/kafka"

	searchv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repos := repository.New(db)

	searchService := service.NewSearchService(repos)

	kafkaHandler := kafka.NewHandler(searchService, log)

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:         cfg.Kafka.Brokers,
		Topic:           cfg.Kafka.SearchIndexTopic,
		GroupID:         cfg.Kafka.GroupID,
		MinBytes:        1,
		MaxBytes:        10e6,
		MaxWait:         2 * time.Second,
		CommitInterval:  time.Second,
		ReadLagInterval: -1,
		StartOffset:     0,
	}, kafkaHandler, log)

	searchHandler := grpcserver.NewSearchHandler(searchService, log)

	grpcServer := grpc.NewServer()

	searchv1.RegisterSearchServiceServer(grpcServer, searchHandler)
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
			errCh <- err
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

	log.Info("search-service stopped")
}
