package http

import (
	"context"
	"errors"
	"fmt"
	"log"
	stdhttp "net/http"

	"github.com/19parwiz/order-service/config"
	"github.com/19parwiz/order-service/internal/adapter/http/handler"
	"github.com/gin-gonic/gin"
)

const serverIPAddress = "0.0.0.0:%d"

type API struct {
	router       *gin.Engine
	cfg          config.HTTPServer
	address      string
	orderHandler *handler.OrderHandler
	httpSrv      *stdhttp.Server
}

func New(cfg config.Server, useCase handler.OrderUsecase) *API {
	gin.SetMode(cfg.HTTPServer.Mode)

	server := gin.New()
	server.Use(gin.Recovery())

	orderHandler := handler.NewOrderHandler(useCase)

	api := &API{
		router:       server,
		cfg:          cfg.HTTPServer,
		address:      fmt.Sprintf(serverIPAddress, cfg.HTTPServer.Port),
		orderHandler: orderHandler,
	}

	api.setupRoutes()

	return api
}

func (api *API) setupRoutes() {
	v1 := api.router.Group("api/v1")
	{
		orders := v1.Group("/orders")
		{
			orders.POST("/", api.orderHandler.Create)
			orders.GET("/:id", api.orderHandler.GetByID)
			orders.PATCH("/:id", api.orderHandler.Update)
			orders.GET("", api.orderHandler.GetAll)
		}
	}
}

func (api *API) Run(errCh chan<- error) {
	api.httpSrv = &stdhttp.Server{
		Addr:    api.address,
		Handler: api.router,
	}
	go func() {
		log.Printf("HTTP server running on: %v", api.address)
		if err := api.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to run HTTP server: %w", err)
		}
	}()
}

func (api *API) Stop(ctx context.Context) error {
	if api.httpSrv == nil {
		return nil
	}
	if err := api.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	log.Println("HTTP server stopped")
	return nil
}
