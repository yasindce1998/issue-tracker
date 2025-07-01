// Package logger provides logging functionality for the application
package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger is the global logger instance available across your application
var ZapLogger *zap.Logger

// Key types for context values to avoid collisions
type contextKey string

const (
	// CacheStatsKey is used to store cache statistics in context
	CacheStatsKey contextKey = "cache_stats"
)

// CacheAccessType represents where data was retrieved from
type CacheAccessType string

const (
	// FromCache indicates the data was retrieved from the cache
	FromCache CacheAccessType = "CACHE"
	// FromDatabase indicates the data was retrieved directly from the database
	FromDatabase CacheAccessType = "DATABASE"
)

// CacheEvent represents a single cache access event
type CacheEvent struct {
	Key      string
	Source   CacheAccessType
	EntityID string
	Entity   string
}

// InitializeLogger configures the global logger with the specified log level
// Valid levels: "debug", "info", "warn", "error", "dpanic", "panic", "fatal"
func InitializeLogger(level string) error {
	// Parse the log level
	var zapLevel zapcore.Level
	err := zapLevel.UnmarshalText([]byte(level))
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", level, err)
	}

	// Create the logger configuration
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      zapLevel == zap.DebugLevel,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// For more readable logs during development
	if zapLevel == zap.DebugLevel {
		config.Encoding = "console"
		config.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %w", err)
	}

	// Set the global logger
	ZapLogger = logger
	return nil
}

// LogCacheAccess logs a cache hit or miss with entity information
func LogCacheAccess(ctx context.Context, entity, entityID string, source CacheAccessType) {
	// Extract trace ID if present
	var traceID string
	if val := ctx.Value(contextKey("trace_id")); val != nil {
		traceID = val.(string)
	}

	// Gather fields for the log message
	fields := []zapcore.Field{
		zap.String("entity", entity),
		zap.String("entity_id", entityID),
		zap.String("data_source", string(source)),
	}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// Record the event
	if source == FromCache {
		ZapLogger.Info("Data retrieved from cache", fields...)
	} else {
		ZapLogger.Info("Data retrieved from database", fields...)
	}
}

// WithCacheStats adds cache tracking to a context
func WithCacheStats(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheStatsKey, make(map[string]CacheEvent))
}
