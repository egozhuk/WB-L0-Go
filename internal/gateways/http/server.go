package http

import (
	"WB-L0/internal/service"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Server struct {
	Host       string
	Port       uint16
	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(service service.Service, host string, port uint16, options ...func(*Server)) *Server {
	re := gin.Default()

	server := &Server{
		router: re,
		Host:   host,
		Port:   port,
	}

	for _, option := range options {
		option(server)
	}

	setupRouter(re, service)

	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", server.Host, server.Port),
		Handler: server.router,
	}

	return server
}

func (s *Server) Run() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server ListenAndServe: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	fmt.Println("Server gracefully stopped")
	return nil
}

func setupRouter(r *gin.Engine, service service.Service) {
	handler := NewHandler(service)

	r.StaticFile("/home", "./web/home.html")

	r.POST("/order", handler.SaveOrder)
	r.GET("/orders", handler.GetOrders)
	r.GET("/order/:uid", handler.GetOrderByUID)

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Go to: /home"})
	})
}
