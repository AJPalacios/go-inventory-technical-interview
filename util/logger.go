package util

import (
	"go.uber.org/zap"
)

// NewLogger creates a new zap logger based on the log level and environment
func NewLogger(logLevel, environment string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Set log level
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return nil, err
	}
	config.Level = level

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
