package http

import (
	"context"
	"errors"
	"fmt"
	"log"
	stdhttp "net/http"

	"github.com/19parwiz/inventory-service/config"
	"github.com/19parwiz/inventory-service/internal/adapter/http/handler"
	"github.com/gin-gonic/gin"
)

const serverIPAddress = "0.0.0.0:%d"

type API struct {
	router         *gin.Engine
	cfg            config.HTTPServer
	address        string
	productHandler *handler.ProductHandler
	httpSrv        *stdhttp.Server
}

func New(cfg config.Server, useCase handler.ProductUseCase) *API {
	gin.SetMode(cfg.HTTPServer.Mode)

	server := gin.New()
	server.Use(gin.Recovery())

	productHandler := handler.NewProductHandler(useCase)

	api := &API{
		router:         server,
		cfg:            cfg.HTTPServer,
		address:        fmt.Sprintf(serverIPAddress, cfg.HTTPServer.Port),
		productHandler: productHandler,
	}

	api.setupRoutes()

	return api
}

func (api *API) setupRoutes() {
	v1 := api.router.Group("api/v1")
	{
		products := v1.Group("/products")
		{
			products.GET("", api.productHandler.GetAll)
			products.GET("/:id", api.productHandler.GetByID)
			products.PUT("/:id", api.productHandler.Update)
			products.POST("/", api.productHandler.Create)
			products.DELETE("/:id", api.productHandler.Delete)
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
