package api

import (
	"database/sql"

	"github.com/AJPalacios/inventory/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	config config.Config
	router *gin.Engine
	db     *sql.DB
	logger *zap.Logger
}

// NewServer creates a new HTTP server and setup routing
func NewServer(cfg config.Config, db *sql.DB, logger *zap.Logger) *Server {
	server := &Server{
		config: cfg,
		db:     db,
		logger: logger,
	}

	server.setupRouter()
	return server
}

func (server *Server) setupRouter() {
	// Set Gin mode based on environment
	if server.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", server.healthCheck)

	server.router = router
}

// Start runs the HTTP server on the specified port
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
