package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/19parwiz/inventory-service/config"
	grpcAPI "github.com/19parwiz/inventory-service/internal/adapter/grpc"
	httpRepo "github.com/19parwiz/inventory-service/internal/adapter/http"
	"github.com/19parwiz/inventory-service/internal/adapter/kafka"
	postgresRepo "github.com/19parwiz/inventory-service/internal/adapter/postgres"
	"github.com/19parwiz/inventory-service/internal/usecase"
	postgresConn "github.com/19parwiz/inventory-service/pkg/postgres"
	"github.com/IBM/sarama"
)

const serviceName = "inventory-service"
const consumerGroupName = "inventory-consumer-group"

func nonEmptyBrokers(in []string) []string {
	var out []string
	for _, b := range in {
		b = strings.TrimSpace(b)
		if b != "" {
			out = append(out, b)
		}
	}
	return out
}

type App struct {
	httpServer    *httpRepo.API
	grpcServer    *grpcAPI.ServerAPI
	consumerGroup sarama.ConsumerGroup
	kafkaHandler  *kafka.Consumer
	pgDB          *postgresConn.DB
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log.Printf(fmt.Sprintf("Initializing %s service!", serviceName))

	log.Println("Connecting to DB:", cfg.Postgres.Database)
	pgDB, err := postgresConn.NewDB(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("error connecting to DB: %v", err)
	}

	aiRepo := postgresRepo.NewAutoInc(pgDB.Pool)
	pRepo := postgresRepo.NewProductRepo(pgDB.Pool)

	pUsecase := usecase.NewProduct(aiRepo, pRepo)

	httpServer := httpRepo.New(cfg.Server, pUsecase)
	grpcServer := grpcAPI.New(cfg.Server, pUsecase)

	brokers := nonEmptyBrokers(cfg.Brokers)
	var consumerGroup sarama.ConsumerGroup
	var kafkaHandler *kafka.Consumer
	if len(brokers) > 0 {
		kafkaConfig := sarama.NewConfig()
		kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
		var err error
		consumerGroup, err = sarama.NewConsumerGroup(brokers, consumerGroupName, kafkaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer group: %w", err)
		}
		kafkaHandler = kafka.NewConsumer(pUsecase, "order.created")
	} else {
		log.Println("Kafka consumer disabled (BROKERS unset or empty); HTTP/gRPC still run. Set BROKERS=127.0.0.1:9092 when Docker Kafka is up.")
	}

	app := &App{
		httpServer:    httpServer,
		grpcServer:    grpcServer,
		consumerGroup: consumerGroup,
		kafkaHandler:  kafkaHandler,
		pgDB:          pgDB,
	}

	return app, nil
}

func (app *App) Start() error {
	errCh := make(chan error)

	app.httpServer.Run(errCh)
	app.grpcServer.Run(errCh)

	if app.consumerGroup != nil && app.kafkaHandler != nil {
		go func() {
			for {
				if err := app.consumerGroup.Consume(context.Background(), []string{app.kafkaHandler.Topic}, app.kafkaHandler); err != nil {
					log.Printf("Consumer error: %v", err)
				}
			}
		}()
	}

	log.Printf(fmt.Sprintf("Starting %s service!", serviceName))

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:

		log.Printf(fmt.Sprintf("Received %v signal, shutting down! Thank you Parwiz ", sig))
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

	if app.consumerGroup != nil {
		if err := app.consumerGroup.Close(); err != nil {
			log.Println("failed to close consumer group:", err)
		}
	}
	if app.pgDB != nil {
		app.pgDB.Close()
	}
}
