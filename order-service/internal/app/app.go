package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/19parwiz/order-service/config"
	grpcAPI "github.com/19parwiz/order-service/internal/adapter/grpc"
	gclients "github.com/19parwiz/order-service/internal/adapter/grpc/clients"
	httpRepo "github.com/19parwiz/order-service/internal/adapter/http"
	"github.com/19parwiz/order-service/internal/adapter/kafka"
	postgresRepo "github.com/19parwiz/order-service/internal/adapter/postgres"
	"github.com/19parwiz/order-service/internal/usecase"
	postgresConn "github.com/19parwiz/order-service/pkg/postgres"
)

const serviceName = "order-service"

type App struct {
	httpServer  *httpRepo.API
	grpcServer  *grpcAPI.ServerAPI
	grpcClients *gclients.Clients
	kafkaProd   *kafka.Producer
	pgDB        *postgresConn.DB
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log.Printf("Initializing %s service...", serviceName)

	log.Println("Connecting to DB:", cfg.Postgres.Database)
	pgDB, err := postgresConn.NewDB(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("error connecting to DB: %v", err)
	}

	aiRepo := postgresRepo.NewAutoInc(pgDB.Pool)
	orderRepo := postgresRepo.NewOrderRepo(pgDB.Pool)

	grpcClients, err := gclients.NewClients(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gRPC clients: %w", err)
	}
	inventoryClient := gclients.NewInventoryClient(grpcClients.Inventory)

	producer, err := kafka.NewKafkaProducer(cfg.Brokers, "order.created")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka producer: %w", err)
	}

	orderUsecase := usecase.NewOrder(aiRepo, orderRepo, inventoryClient, producer)

	httpServer := httpRepo.New(cfg.Server, orderUsecase)
	grpcServer := grpcAPI.New(cfg.Server, orderUsecase)

	app := &App{
		httpServer:  httpServer,
		grpcServer:  grpcServer,
		grpcClients: grpcClients,
		kafkaProd:   producer,
		pgDB:        pgDB,
	}

	return app, nil
}

func (app *App) Start() error {
	errCh := make(chan error)

	app.httpServer.Run(errCh)
	app.grpcServer.Run(errCh)

	log.Printf("Starting %s service!", serviceName)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:
		log.Printf("Received %v signal, shutting down!", sig)
		app.Stop()
		log.Println("graceful shutdown completed!")
	}
	return nil
}

func (app *App) Stop() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if app.httpServer != nil {
		if err := app.httpServer.Stop(shutdownCtx); err != nil {
			log.Println("failed to shutdown http service:", err)
		}
	}
	err := app.grpcServer.Stop()
	if err != nil {
		log.Println("failed to shutdown grpc service:", err)
	}

	// Guard clauses keep shutdown safe even if initialization failed halfway.
	if app.grpcClients != nil {
		app.grpcClients.Close()
	}
	if app.kafkaProd != nil {
		if err := app.kafkaProd.Close(); err != nil {
			log.Println("failed to close Kafka producer:", err)
		}
	}
	if app.pgDB != nil {
		app.pgDB.Close()
	}
}
