// Package api provides HTTP routing configuration for the inventory service.
//
// This package contains route definitions and middleware configuration
// using Gin framework with proper grouping and organization.
package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AJPalacios/inventory/internal/api/handlers"
	"github.com/AJPalacios/inventory/internal/domain"
	"github.com/gin-gonic/gin"
)

// RouterConfig holds configuration for the API router.
type RouterConfig struct {
	InventoryHandler *handlers.InventoryHandler
	Logger           domain.Logger
	MetricsProvider  domain.MetricsProvider
}

// SetupRoutes configures all routes and middleware for the Gin engine.
//
// This function sets up the complete routing structure with proper
// middleware chain and API versioning using Gin groups.
func SetupRoutes(engine *gin.Engine, config RouterConfig) {
	// Add global middleware
	engine.Use(ginLogger(config.Logger))
	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())
	engine.Use(requestIDMiddleware())
	engine.Use(metricsMiddleware(config.MetricsProvider))

	// Root health check endpoint (simple service health)
	engine.GET("/health", rootHealthCheck)

	// Root documentation endpoints for easy access
	docsHandler := handlers.NewDocsHandler()
	engine.GET("/docs", docsHandler.GetSwaggerUI)
	engine.GET("/openapi.json", docsHandler.GetOpenAPISpec)

	// API v1 group
	v1 := engine.Group("/api/v1")
	{
		// Inventory management group
		setupInventoryRoutes(v1, config.InventoryHandler)
	}

	config.Logger.Info("Routes configured successfully", map[string]interface{}{
		"api_version":  "v1",
		"total_routes": len(engine.Routes()),
	})
}

// setupInventoryRoutes configures inventory-specific routes.
func setupInventoryRoutes(v1 *gin.RouterGroup, handler *handlers.InventoryHandler) {
	inventory := v1.Group("/inventory")
	{
		// Core stock operations
		inventory.POST("/reserve", handler.ReserveStock)
		inventory.POST("/release", handler.ReleaseStock)
		inventory.PUT("/stock", handler.UpdateStock)
		inventory.GET("/stock/:productId", handler.GetStock)

		// Batch operations
		batch := inventory.Group("/batch")
		{
			batch.POST("/reserve", handler.BatchReserveStock)
		}
	}

	// Documentation endpoints
	docsHandler := handlers.NewDocsHandler()
	docs := v1.Group("/docs")
	{
		docs.GET("", docsHandler.GetAPIDocumentation)
		docs.GET("/openapi.json", docsHandler.GetOpenAPISpec)
		docs.GET("/swagger", docsHandler.GetSwaggerUI)
	}
}

// Middleware functions

// ginLogger provides custom logging middleware for Gin.
func ginLogger(logger domain.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log fields
		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		if raw != "" {
			fields["query"] = raw
		}

		if requestID := c.GetString("request_id"); requestID != "" {
			fields["request_id"] = requestID
		}

		// Log based on status code
		if c.Writer.Status() >= 500 {
			logger.Error("HTTP request completed with server error", nil, fields)
		} else if c.Writer.Status() >= 400 {
			logger.Warn("HTTP request completed with client error", fields)
		} else {
			logger.Info("HTTP request completed", fields)
		}
	}
}

// corsMiddleware adds CORS headers for cross-origin requests.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestIDMiddleware adds request ID tracking to all requests.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// metricsMiddleware records metrics for HTTP requests.
func metricsMiddleware(metricsProvider domain.MetricsProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		if metricsProvider == nil {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		labels := map[string]string{
			"method": c.Request.Method,
			"path":   c.FullPath(),
			"status": fmt.Sprintf("%d", c.Writer.Status()),
		}

		metricsProvider.IncrementCounter("http_requests_total", labels)
		metricsProvider.RecordDuration("http_request_duration", duration, labels)

		// Record status code metrics
		if c.Writer.Status() >= 500 {
			metricsProvider.IncrementCounter("http_server_errors_total", labels)
		} else if c.Writer.Status() >= 400 {
			metricsProvider.IncrementCounter("http_client_errors_total", labels)
		}
	}
}

// rootHealthCheck provides a simple health check endpoint.
func rootHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"service":   "inventory-api",
	})
}
