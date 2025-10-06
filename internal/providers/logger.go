package providers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
)

// LoggerProviderType defines available logger provider types.
type LoggerProviderType string

const (
	LoggerProviderZap    LoggerProviderType = "zap"
	LoggerProviderLogrus LoggerProviderType = "logrus"
	LoggerProviderStd    LoggerProviderType = "std"
)

// LogLevel defines log levels.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LoggerConfig holds configuration for logger provider.
type LoggerConfig struct {
	Provider LoggerProviderType
	Level    LogLevel
	Format   string // "json" or "text"
	Output   string // "stdout", "stderr", or file path
}

// NewLogger creates a logger provider based on configuration.
//
// This factory function allows switching between different logging
// providers without changing the business logic code.
func NewLogger(config LoggerConfig) domain.Logger {
	switch config.Provider {
	case LoggerProviderZap:
		// return NewZapLogger(config)
		log.Printf("Zap logger not implemented, falling back to standard")
		return NewStandardLogger(config)
	case LoggerProviderLogrus:
		// return NewLogrusLogger(config)
		log.Printf("Logrus logger not implemented, falling back to standard")
		return NewStandardLogger(config)
	case LoggerProviderStd:
		return NewStandardLogger(config)
	default:
		log.Printf("Unknown logger provider %s, falling back to standard", config.Provider)
		return NewStandardLogger(config)
	}
}

// standardLogger provides structured logging using Go's standard library.
//
// This implementation provides basic structured logging functionality
// using the standard log package with JSON formatting support.
type standardLogger struct {
	level      LogLevel
	format     string
	logger     *log.Logger
	baseFields map[string]interface{}
}

// NewStandardLogger creates a standard library-based logger.
func NewStandardLogger(config LoggerConfig) domain.Logger {
	var output *os.File

	switch config.Output {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file %s: %v, falling back to stdout", config.Output, err)
			output = os.Stdout
		} else {
			output = file
		}
	}

	return &standardLogger{
		level:      config.Level,
		format:     config.Format,
		logger:     log.New(output, "", 0), // No default formatting
		baseFields: make(map[string]interface{}),
	}
}

// Debug logs a debug message with optional structured fields.
func (l *standardLogger) Debug(msg string, fields ...map[string]interface{}) {
	if !l.shouldLog(LogLevelDebug) {
		return
	}
	l.logMessage("DEBUG", msg, fields...)
}

// Info logs an info message with optional structured fields.
func (l *standardLogger) Info(msg string, fields ...map[string]interface{}) {
	if !l.shouldLog(LogLevelInfo) {
		return
	}
	l.logMessage("INFO", msg, fields...)
}

// Warn logs a warning message with optional structured fields.
func (l *standardLogger) Warn(msg string, fields ...map[string]interface{}) {
	if !l.shouldLog(LogLevelWarn) {
		return
	}
	l.logMessage("WARN", msg, fields...)
}

// Error logs an error message with optional structured fields.
func (l *standardLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	if !l.shouldLog(LogLevelError) {
		return
	}

	// Add error to fields
	mergedFields := l.mergeFields(fields...)
	if err != nil {
		mergedFields["error"] = err.Error()
	}

	l.logMessage("ERROR", msg, mergedFields)
}

// With creates a new logger with additional base fields.
func (l *standardLogger) With(fields map[string]interface{}) domain.Logger {
	newLogger := &standardLogger{
		level:      l.level,
		format:     l.format,
		logger:     l.logger,
		baseFields: make(map[string]interface{}),
	}

	// Copy existing base fields
	for k, v := range l.baseFields {
		newLogger.baseFields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.baseFields[k] = v
	}

	return newLogger
}

// shouldLog determines if a message should be logged based on level.
func (l *standardLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}

	return levels[level] >= levels[l.level]
}

// logMessage formats and outputs a log message.
func (l *standardLogger) logMessage(level, msg string, fields ...map[string]interface{}) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	if l.format == "json" {
		l.logJSON(level, msg, timestamp, fields...)
	} else {
		l.logText(level, msg, timestamp, fields...)
	}
}

// logJSON outputs a JSON-formatted log message.
func (l *standardLogger) logJSON(level, msg, timestamp string, fields ...map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": timestamp,
		"level":     level,
		"message":   msg,
	}

	// Add base fields
	for k, v := range l.baseFields {
		logEntry[k] = v
	}

	// Add additional fields
	mergedFields := l.mergeFields(fields...)
	for k, v := range mergedFields {
		logEntry[k] = v
	}

	jsonBytes, err := json.Marshal(logEntry)
	if err != nil {
		// Fallback to text format if JSON marshaling fails
		l.logText(level, msg, timestamp, fields...)
		return
	}

	l.logger.Println(string(jsonBytes))
}

// logText outputs a text-formatted log message.
func (l *standardLogger) logText(level, msg, timestamp string, fields ...map[string]interface{}) {
	output := fmt.Sprintf("[%s] %s %s", level, timestamp, msg)

	// Add base fields
	if len(l.baseFields) > 0 {
		output += " |"
		for k, v := range l.baseFields {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	// Add additional fields
	mergedFields := l.mergeFields(fields...)
	if len(mergedFields) > 0 {
		if len(l.baseFields) == 0 {
			output += " |"
		}
		for k, v := range mergedFields {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	l.logger.Println(output)
}

// mergeFields merges multiple field maps into a single map.
func (l *standardLogger) mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			merged[k] = v
		}
	}

	return merged
}
