package main

import (
	"content-service/config"
	"content-service/internal/repository"
	"content-service/internal/repositoryaccess"
	"content-service/internal/service"
	grpcserver "content-service/internal/transport/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"

	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

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

	repoConn, err := grpc.NewClient(
		cfg.RepoistoryGRPC.Addres,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("Failed to connect to repository service", zap.Error(err))
	}
	defer repoConn.Close()

	repoClient := repositoryv1.NewRepositoryServiceClient(repoConn)

	repoAccess := repositoryaccess.NewGRPCChecker(
		repoClient,
		log,
		cfg.RepoistoryGRPC.RequestTimeout,
	)

	contentService := service.NewContentService(
		repos,
		repoAccess,
	)

	tokenManager := authjwt.NewJWTVerifier(cfg.Auth.JWTSecret)
	authInterceptor := grpcserver.NewAuthInterceptor(tokenManager, log)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor.UnaryServerInterceptor(),
		),
	)

	contentHandler := grpcserver.NewContentHandler(contentService, log)

	contentv1.RegisterContentServiceServer(grpcServer, contentHandler)
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
