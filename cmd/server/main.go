// Package main provides the entry point for the inventory management server.
//
// This application provides a complete inventory management system with
// ACID-compliant operations, comprehensive monitoring, and RESTful APIs using Gin.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AJPalacios/inventory/internal/api"
	"github.com/AJPalacios/inventory/internal/api/handlers"
	"github.com/AJPalacios/inventory/internal/config"
	"github.com/AJPalacios/inventory/internal/domain"
	"github.com/AJPalacios/inventory/internal/providers"
	"github.com/AJPalacios/inventory/internal/repository"
	"github.com/AJPalacios/inventory/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// Server holds the main application components.
type Server struct {
	config     *config.Config
	db         *sql.DB
	logger     domain.Logger
	metrics    domain.MetricsProvider
	httpServer *http.Server
	ginEngine  *gin.Engine
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create and start server
	server, err := NewServer(&cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Start server with graceful shutdown
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// NewServer creates a new server instance with all dependencies.
func NewServer(cfg *config.Config) (*Server, error) {
	server := &Server{
		config: cfg,
	}

	// Initialize core dependencies
	if err := server.initializeCore(); err != nil {
		return nil, fmt.Errorf("failed to initialize core: %w", err)
	}

	// Initialize business layer
	if err := server.initializeBusiness(); err != nil {
		return nil, fmt.Errorf("failed to initialize business layer: %w", err)
	}

	return server, nil
}

// initializeCore sets up database, logger, and metrics.
func (s *Server) initializeCore() error {
	// Initialize logger
	loggerConfig := providers.LoggerConfig{
		Provider: providers.LoggerProviderStd,
		Level:    providers.LogLevel("info"),
		Format:   "json",
		Output:   "stdout",
	}
	s.logger = providers.NewLogger(loggerConfig)

	s.logger.Info("Logger initialized", map[string]interface{}{
		"provider": string(loggerConfig.Provider),
		"level":    string(loggerConfig.Level),
	})

	// Initialize metrics
	metricsConfig := providers.MetricsConfig{
		Provider:  providers.MetricsProviderMemory,
		Namespace: "inventory",
		Labels: map[string]string{
			"service": "inventory-api",
			"version": "1.0.0",
		},
	}
	s.metrics = providers.NewMetricsProvider(metricsConfig)

	s.logger.Info("Metrics provider initialized", map[string]interface{}{
		"provider":  string(metricsConfig.Provider),
		"namespace": metricsConfig.Namespace,
	})

	// Initialize database - adjust field names based on Config struct
	dbPath := s.config.DBPath
	if dbPath == "" {
		dbPath = "inventory.db" // default fallback
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	s.db = db

	s.logger.Info("Database connected", map[string]interface{}{
		"driver": "sqlite3",
		"source": dbPath,
	})

	return nil
}

// initializeBusiness sets up services and HTTP server.
func (s *Server) initializeBusiness() error {
	// Initialize repository
	repo := repository.New(s.db)

	// Initialize services
	validationService := domain.NewValidationService()

	idempotencyService := service.NewIdempotencyService(service.IdempotencyServiceConfig{
		MaxSize:    100000,
		DefaultTTL: 24 * time.Hour,
	})

	// Initialize inventory service
	inventoryService := service.NewInventoryServiceImpl(
		repo,
		validationService,
		idempotencyService,
		s.logger,
		s.metrics,
		service.InventoryServiceConfig{
			OperationTimeout:   30 * time.Second,
			MaxRetryAttempts:   3,
			MaxBatchSize:       100,
			LowStockThreshold:  10,
			MaxStockCapacity:   100000,
			ConcurrentOpsLimit: 1000,
		},
	) // Initialize handlers
	inventoryHandler := handlers.NewInventoryHandler(inventoryService, s.logger)

	// Set Gin mode based on environment
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin engine
	s.ginEngine = gin.New()

	// Setup routes with configuration
	routerConfig := api.RouterConfig{
		InventoryHandler: inventoryHandler,
		Logger:           s.logger,
		MetricsProvider:  s.metrics,
	}
	api.SetupRoutes(s.ginEngine, routerConfig)

	// Create HTTP server
	serverPort := s.config.ServerPort
	if serverPort == "" {
		serverPort = "8080" // default fallback
	}

	s.httpServer = &http.Server{
		Addr:           ":" + serverPort,
		Handler:        s.ginEngine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	s.logger.Info("Business layer initialized", map[string]interface{}{
		"server_address": s.httpServer.Addr,
		"gin_mode":       gin.Mode(),
		"services": []string{
			"inventory_service",
			"validation_service",
			"idempotency_service",
		},
	})

	return nil
}

// Start starts the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	s.logger.Info("Starting inventory server", map[string]interface{}{
		"environment": s.config.Environment,
		"address":     s.httpServer.Addr,
		"routes":      len(s.ginEngine.Routes()),
	})

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server failed to start", err, nil)
			os.Exit(1)
		}
	}()

	s.logger.Info("Server started successfully", map[string]interface{}{
		"address": s.httpServer.Addr,
		"ready":   true,
	})

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...", nil)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", err, nil)
		return err
	}

	s.logger.Info("Server shutdown complete", nil)
	return nil
}

// Close closes all resources.
func (s *Server) Close() {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Failed to close database", err, nil)
		} else {
			s.logger.Info("Database connection closed", nil)
		}
	}
}
