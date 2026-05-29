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


func processOrder(_ context.Context, input []byte) ([]byte, error){
	slog.Info("processing order", "input", string(input))
	return []byte(`{"status":"charged"}`), nil

}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout,nil)))

	c,err := client.New(
		client.Config{
			ServerURL: "http://localhost:8080",
			Queue: "Orders-queue",
		},
		client.WithPollInterval(500*time.Millisecond),
		client.WithHandlerTimeout(30*time.Second),
	)

	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(1)
	}

	if err := c.Register("processOrder", processOrder); err != nil {
		slog.Error("failed to register handler", "error", err)
		os.Exit(1)
	}

	c.Start()
	slog.Info("worker started")

	result, err := c.Enqueue("processOrder", []byte(`{"orderId":"123","amount":99.99}`))
	if err != nil {
		slog.Error("failed to enqueue task", "error", err)
	} else {
		slog.Info("task enqueued", "task_id", result.TaskID, "queue", result.Queue)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	c.Stop()


}