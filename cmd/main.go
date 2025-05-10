package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/user_protos"
	"github.com/ruziba3vich/mm_article_service/internal/service"
	"github.com/ruziba3vich/mm_article_service/internal/storage"
	"github.com/ruziba3vich/mm_article_service/pkg/config"
	logger "github.com/ruziba3vich/prodonik_lgger"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.LoadConfig,
			newLogger,
			newUserServiceClient,
			storage.NewGORM,
			storage.NewArticleRepository,
			storage.NewFileDbStorage,
			storage.NewMinIOStorage,
			service.NewArticleService,
			newGrpcServer,
		),
		fx.Invoke(registerHooks),
	)

	app.Run()
}

// Create a new gRPC server and register the logging service
func newGrpcServer(srv *service.ArticleService) *grpc.Server {
	server := grpc.NewServer()
	article_protos.RegisterArticleServiceServer(server, srv)
	return server
}

// Register application lifecycle hooks
func registerHooks(
	lc fx.Lifecycle,
	db *gorm.DB,
	grpcServer *grpc.Server,
	cfg *config.Config,
) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Println("Starting article service...")

			listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
			if err != nil {
				return fmt.Errorf("failed to listen on port %s: %s", cfg.GRPCPort, err.Error())
			}

			log.Printf("gRPC server listening on port %s", cfg.GRPCPort)

			go func() {
				if err := grpcServer.Serve(listener); err != nil {
					log.Fatalf("Failed to serve gRPC: %v", err)
				}
			}()

			log.Println("Article service started")
			return nil
		},
		OnStop: func(context.Context) error {
			log.Println("Stopping article service...")

			grpcServer.GracefulStop()
			sqlDB, err := db.DB()
			if err != nil {
				log.Printf("Error getting raw db connection: %v", err)
				return err
			}
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}

			log.Println("Article service stopped")
			return nil
		},
	})

	// Setup signal handling for graceful shutdown
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals

		log.Println("Received shutdown signal")
	}()
}

func newUserServiceClient(cfg *config.Config, logger *logger.Logger) (user_protos.UserServiceClient, error) {
	conn, err := grpc.NewClient(cfg.UserService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to User Service", map[string]any{"error": err})
		return nil, err
	}
	logger.Info("Connected to gRPC service", map[string]any{"address": cfg.UserService})
	return user_protos.NewUserServiceClient(conn), nil
}

func newLogger() (*logger.Logger, error) {
	return logger.NewLogger("/app/article_service.log")
}
