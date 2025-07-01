// Package main provides starts the http and gRPC servers and log the status
package main

import (
	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/server"
	"go.uber.org/zap"
)

func main() {
	// Create and initialize the application
	app, err := server.NewApplication()
	if err != nil {
		panic("Failed to initialize application: " + err.Error())
	}

	// Ensure logger is properly flushed on exit
	defer func() {
		// Ignoring the error as we can't do anything about it during shutdown
		_ = logger.ZapLogger.Sync()
	}()

	// Start the application
	if err := app.Start(); err != nil {
		logger.ZapLogger.Fatal("Failed to serve", zap.Error(err))
	}
}
