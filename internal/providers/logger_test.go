package providers

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoggerProvider tests the logger functionality
func TestLoggerProvider(t *testing.T) {
	config := LoggerConfig{
		Provider: LoggerProviderStd,
		Level:    LogLevelDebug,
		Format:   "text",
		Output:   "stdout",
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	// Test basic logging methods exist and don't panic
	logger.Debug("test debug message")
	logger.Info("test info message")
	logger.Warn("test warn message")
	logger.Error("test error message", errors.New("test error"))
}

func TestLoggerWithFields(t *testing.T) {
	config := LoggerConfig{
		Provider: LoggerProviderStd,
		Level:    LogLevelDebug,
		Format:   "text",
		Output:   "stdout",
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	// Test With method
	loggerWithFields := logger.With(map[string]interface{}{
		"component": "test",
		"version":   "1.0",
	})
	require.NotNil(t, loggerWithFields)

	// Test logging with additional fields
	loggerWithFields.Info("test message", map[string]interface{}{
		"extra_field": "extra_value",
	})
}

func TestLoggerLevels(t *testing.T) {
	// Test different log levels
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}

	for _, level := range levels {
		config := LoggerConfig{
			Provider: LoggerProviderStd,
			Level:    level,
			Format:   "text",
			Output:   "stdout",
		}

		logger := NewLogger(config)
		require.NotNil(t, logger)

		// All these should not panic
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message", errors.New("test error"))
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	config := LoggerConfig{
		Provider: LoggerProviderStd,
		Level:    LogLevelInfo,
		Format:   "json",
		Output:   "stdout",
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	logger.Info("json test message", map[string]interface{}{
		"key": "value",
	})
}

func TestLoggerProviderFallback(t *testing.T) {
	// Test fallback to standard logger for unimplemented providers
	config := LoggerConfig{
		Provider: LoggerProviderZap,
		Level:    LogLevelInfo,
		Format:   "text",
		Output:   "stdout",
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	config.Provider = LoggerProviderLogrus
	logger = NewLogger(config)
	require.NotNil(t, logger)

	config.Provider = "unknown"
	logger = NewLogger(config)
	require.NotNil(t, logger)
}

func TestLoggerFileOutput(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_log_*.log")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := LoggerConfig{
		Provider: LoggerProviderStd,
		Level:    LogLevelInfo,
		Format:   "text",
		Output:   tmpFile.Name(),
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	logger.Info("test file output")

	// Verify file was written to
	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(content), "test file output")
}

func TestStandardLoggerConfig(t *testing.T) {
	// Test different configurations
	configs := []LoggerConfig{
		{Provider: LoggerProviderStd, Level: LogLevelDebug, Format: "text", Output: "stdout"},
		{Provider: LoggerProviderStd, Level: LogLevelInfo, Format: "json", Output: "stderr"},
		{Provider: LoggerProviderStd, Level: LogLevelWarn, Format: "text", Output: ""},
	}

	for _, config := range configs {
		logger := NewStandardLogger(config)
		require.NotNil(t, logger)

		// Test that all methods work
		logger.Debug("debug")
		logger.Info("info")
		logger.Warn("warn")
		logger.Error("error", errors.New("test error"))
	}
}

func TestLoggerShouldLog(t *testing.T) {
	config := LoggerConfig{
		Provider: LoggerProviderStd,
		Level:    LogLevelWarn, // Only warn and error should log
		Format:   "text",
		Output:   "stdout",
	}

	stdLogger := NewStandardLogger(config).(*standardLogger)

	// Test shouldLog method
	assert.False(t, stdLogger.shouldLog(LogLevelDebug))
	assert.False(t, stdLogger.shouldLog(LogLevelInfo))
	assert.True(t, stdLogger.shouldLog(LogLevelWarn))
	assert.True(t, stdLogger.shouldLog(LogLevelError))
}

func TestLoggerMergeFields(t *testing.T) {
	config := LoggerConfig{
		Provider: LoggerProviderStd,
		Level:    LogLevelInfo,
		Format:   "text",
		Output:   "stdout",
	}

	stdLogger := NewStandardLogger(config).(*standardLogger)

	// Test mergeFields method
	fields1 := map[string]interface{}{"key1": "value1", "key2": "value2"}
	fields2 := map[string]interface{}{"key2": "overwrite", "key3": "value3"}

	merged := stdLogger.mergeFields(fields1, fields2)

	assert.Equal(t, "value1", merged["key1"])
	assert.Equal(t, "overwrite", merged["key2"]) // Should be overwritten
	assert.Equal(t, "value3", merged["key3"])
}

func TestLoggerConstants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, LoggerProviderType("zap"), LoggerProviderZap)
	assert.Equal(t, LoggerProviderType("logrus"), LoggerProviderLogrus)
	assert.Equal(t, LoggerProviderType("std"), LoggerProviderStd)

	assert.Equal(t, LogLevel("debug"), LogLevelDebug)
	assert.Equal(t, LogLevel("info"), LogLevelInfo)
	assert.Equal(t, LogLevel("warn"), LogLevelWarn)
	assert.Equal(t, LogLevel("error"), LogLevelError)
}
