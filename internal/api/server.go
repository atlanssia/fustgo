package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/atlanssia/fustgo/internal/logger"
)

// Server represents the HTTP API server
type Server struct {
	router  *gin.Engine
	server  *http.Server
	config  *ServerConfig
	handler *Handler
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string
	Port            int
	Mode            string // "debug", "release", "test"
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxHeaderBytes  int
	EnableCORS      bool
	TrustedProxies  []string
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:           "0.0.0.0",
		Port:           8080,
		Mode:           gin.ReleaseMode,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
		EnableCORS:     true,
		TrustedProxies: []string{"127.0.0.1"},
	}
}

// NewServer creates a new API server
func NewServer(config *ServerConfig, handler *Handler) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Set Gin mode
	gin.SetMode(config.Mode)

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware())
	router.Use(RequestIDMiddleware())

	// CORS configuration
	if config.EnableCORS {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}
		router.Use(cors.New(corsConfig))
	}

	// Set trusted proxies
	if len(config.TrustedProxies) > 0 {
		router.SetTrustedProxies(config.TrustedProxies)
	}

	server := &Server{
		router:  router,
		config:  config,
		handler: handler,
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/", s.home)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Jobs endpoints
		jobs := v1.Group("/jobs")
		{
			jobs.GET("", s.handler.ListJobs)
			jobs.POST("", s.handler.CreateJob)
			jobs.GET("/:id", s.handler.GetJob)
			jobs.PUT("/:id", s.handler.UpdateJob)
			jobs.DELETE("/:id", s.handler.DeleteJob)
			jobs.POST("/:id/start", s.handler.StartJob)
			jobs.POST("/:id/stop", s.handler.StopJob)
			jobs.POST("/:id/pause", s.handler.PauseJob)
			jobs.POST("/:id/resume", s.handler.ResumeJob)
		}

		// Plugins endpoints
		plugins := v1.Group("/plugins")
		{
			plugins.GET("", s.handler.ListPlugins)
			plugins.GET("/:name", s.handler.GetPlugin)
		}

		// Workers endpoints
		workers := v1.Group("/workers")
		{
			workers.GET("", s.handler.ListWorkers)
			workers.GET("/:id", s.handler.GetWorker)
		}

		// Monitoring endpoints
		monitoring := v1.Group("/monitoring")
		{
			monitoring.GET("/stats", s.handler.GetStats)
			monitoring.GET("/metrics", s.handler.GetMetrics)
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	logger.Info("Starting API server on %s", addr)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Shutting down API server...")

	if s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	logger.Info("API server stopped")
	return nil
}

// GetRouter returns the Gin router (for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// Basic handlers (placeholder implementations)

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "FustGo DataX API",
		"version": "0.1.0",
		"message": "Welcome to FustGo DataX REST API",
	})
}


