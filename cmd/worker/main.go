package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/yadukrishnan2004/antrelay-sdk/client"
)

// processOrder is a real business handler.
// This is the only code the developer writes.
func processOrder(_ context.Context, input []byte) ([]byte, error) {
	slog.Info("processing order", "input", string(input))
	return []byte(`{"status":"charged"}`), nil
}

func sendEmail(_ context.Context, input []byte) ([]byte, error) {
	slog.Info("sending email", "input", string(input))
	return []byte(`{"status":"sent"}`), nil
}

func main() {
	// configure structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// create the SDK client
	c, err := client.New(
		client.Config{
			ServerURL: "http://localhost:8080",
			Queue:     "orders-queue",
		},
		client.WithPollInterval(500*time.Millisecond),
		client.WithHandlerTimeout(30*time.Second),
	)
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(1)
	}

	// register business handlers
	if err := c.Register("processOrder", processOrder); err != nil {
		slog.Error("failed to register handler", "error", err)
		os.Exit(1)
	}

	if err := c.Register("sendEmail", sendEmail); err != nil {
		slog.Error("failed to register handler", "error", err)
		os.Exit(1)
	}

	// start the worker
	c.Start()
	slog.Info("worker started, waiting for tasks")

	// wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutdown signal received")
	c.Stop()
}