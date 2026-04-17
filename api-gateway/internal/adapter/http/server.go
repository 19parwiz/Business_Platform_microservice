package http

import (
	"context"
	"errors"
	"fmt"
	"log"
	stdhttp "net/http"

	"github.com/19parwiz/api-gateway/config"
	"github.com/19parwiz/api-gateway/internal/adapter/http/handler"
	"github.com/19parwiz/api-gateway/internal/adapter/http/middleware"
	"github.com/gin-gonic/gin"
)

const serverIPAddress = "0.0.0.0:%d"

// corsMiddleware allows browser clients (e.g. static frontend on another port) to call the gateway.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type, X-Email, X-Password")
		if c.Request.Method == stdhttp.MethodOptions {
			c.AbortWithStatus(stdhttp.StatusNoContent)
			return
		}
		c.Next()
	}
}

type Server struct {
	router  *gin.Engine
	cfg     config.HTTPServer
	address string
	handler *handler.Handler
	httpSrv *stdhttp.Server
}

func NewServer(cfg config.Config, handler *handler.Handler) *Server {
	gin.SetMode(cfg.HTTPServer.Mode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	s := &Server{
		router:  r,
		cfg:     cfg.HTTPServer,
		address: fmt.Sprintf(serverIPAddress, cfg.HTTPServer.Port),
		handler: handler,
	}

	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	v1 := s.router.Group("/api/v1")

	v1.POST("/users/register", s.handler.RegisterUser)
	v1.GET("/users/profile", middleware.AuthMiddleware(s.handler.Clients.User), s.handler.GetUserProfile)

	v1.GET("/products", s.handler.ListProducts)

	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware(s.handler.Clients.User))
	{
		protected.POST("/products", s.handler.CreateProduct)
		protected.GET("/products/:id", s.handler.GetProduct)
		protected.PUT("/products/:id", s.handler.UpdateProduct)
		protected.DELETE("/products/:id", s.handler.DeleteProduct)

		protected.POST("/orders", s.handler.CreateOrder)
		protected.GET("/orders", s.handler.GetOrders)
		protected.GET("/orders/:id", s.handler.GetOrder)
		protected.PUT("/orders/:id", s.handler.UpdateOrder)
	}
	s.router.NoRoute(func(c *gin.Context) {
		log.Printf("No route matched: %s %s", c.Request.Method, c.Request.URL.String())
		c.JSON(stdhttp.StatusNotFound, gin.H{"error": "Service not found"})
	})
}

func (s *Server) Run(errCh chan<- error) {
	s.httpSrv = &stdhttp.Server{
		Addr:    s.address,
		Handler: s.router,
	}
	go func() {
		log.Printf("HTTP server running on: %v", s.address)
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to run HTTP server: %w", err)
		}
	}()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	log.Println("HTTP server stopped")
	return nil
}
