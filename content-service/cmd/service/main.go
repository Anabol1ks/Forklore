package main

import (
	"content-service/config"
	"content-service/internal/repository"
	"content-service/internal/repositoryaccess"
	"content-service/internal/service"
	"os"

	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	"github.com/Anabol1ks/Forklore/pkg/utils/database"
	"github.com/Anabol1ks/Forklore/pkg/utils/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	_ = contentService
}
