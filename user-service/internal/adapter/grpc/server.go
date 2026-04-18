package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/19parwiz/user-service/config"
	"github.com/19parwiz/user-service/internal/usecase"
	proto "github.com/19parwiz/user-service/protos/gen/golang"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

const serverIPAddress = "0.0.0.0:%d" // for external access

// ServerAPI manages the gRPC server lifecycle
type ServerAPI struct {
	grpcServer  *grpc.Server
	cfg         config.GRPCServer
	address     string
	userHandler *UserGRPCServer
}

// New creates a new gRPC Server instance
func New(cfg config.Server, userUsecase usecase.UserUsecase) *ServerAPI {
	grpcServer := grpc.NewServer()

	userHandler := NewUserGRPCServer(userUsecase)

	// Register the Auth service with the gRPC server
	proto.RegisterAuthServer(grpcServer, userHandler)

	server := &ServerAPI{
		grpcServer:  grpcServer,
		cfg:         cfg.GRPCServer,
		address:     fmt.Sprintf(serverIPAddress, cfg.GRPCServer.Port),
		userHandler: userHandler,
	}

	return server
}

// Run starts the gRPC server
func (s *ServerAPI) Run(errCh chan<- error) {
	go func() {
		lis, err := net.Listen("tcp", s.address)
		if err != nil {
			errCh <- fmt.Errorf("failed to listen on %s: %w", s.address, err)
			return
		}
		log.Printf("gRPC listening on %s", lis.Addr().String())

		if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("failed to run gRPC server: %w", err)
			return
		}
	}()
}

// Stop gracefully shuts down the gRPC server
func (s *ServerAPI) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("gRPC server shutting down gracefully")

	// Gracefully stop the gRPC server
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC server stopped successfully")
	case <-ctx.Done():
		log.Println("gRPC server shutdown timed out, forcing stop")
		s.grpcServer.Stop()
	}

	return nil
}
