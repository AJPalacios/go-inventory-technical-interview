package main

import (
	"database/sql"
	"log"

	"github.com/AJPalacios/inventory/api"
	"github.com/AJPalacios/inventory/util"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Initialize logger
	logger, err := util.NewLogger(cfg.LogLevel, cfg.Environment)
	if err != nil {
		log.Fatal("cannot initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting inventory server",
		zap.String("environment", cfg.Environment),
		zap.String("port", cfg.ServerPort),
	)

	// Initialize database
	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		logger.Fatal("cannot connect to database", zap.Error(err))
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Fatal("cannot ping database", zap.Error(err))
	}

	logger.Info("database connection established",
		zap.String("path", cfg.DBPath),
	)

	// Create and start server
	server := api.NewServer(cfg, db, logger)

	logger.Info("server listening", zap.String("port", cfg.ServerPort))

	if err := server.Start(":" + cfg.ServerPort); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}
}
