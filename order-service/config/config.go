package config

import (
	"github.com/19parwiz/order-service/pkg/postgres"
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"log"

	"time"
)

type (
	Config struct {
		Postgres postgres.Config
		Server   Server
		Version  string `env:"VERSION"`
		Services Microservices
		Brokers  []string `env:"BROKERS"`
	}

	Server struct {
		HTTPServer HTTPServer
		GRPCServer GRPCServer
	}

	HTTPServer struct {
		Port           int           `env:"HTTP_PORT,required"`
		ReadTimeout    time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"30s"`
		WriteTimeout   time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`
		IdleTimeout    time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
		MaxHeaderBytes int           `env:"HTTP_MAX_HEADER_BYTES" envDefault:"1048576"` // 1 MB
		Mode           string        `env:"GIN_MODE" envDefault:"release"`              // Can be: release, debug, test
	}

	GRPCServer struct {
		Port    int           `env:"GRPC_PORT,required"`
		Timeout time.Duration `env:"GRPC_TIMEOUT" envDefault:"30s"`
	}

	Microservices struct {
		InventoryService ServiceConfig `envPrefix:"INVENTORY_SERVICE_"`
		//if you need other clients...
	}

	ServiceConfig struct {
		Host string `env:"HOST,required"`
		Port int    `env:"PORT,required"`
	}
)

func New() (*Config, error) {
	//Loading local .env file for private configuration
	if err := godotenv.Load("local.env"); err != nil {
		log.Printf("Error loading local.env file")
	}

	var cfg Config
	err := env.Parse(&cfg)

	return &cfg, err
}
