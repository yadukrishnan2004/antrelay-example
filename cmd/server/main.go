package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yadukrishnan2004/antrelay-example/internal/handler"
	"github.com/yadukrishnan2004/antrelay-example/internal/service"
	"github.com/yadukrishnan2004/antrelay-example/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	orderStore, err := store.New("orders.db")
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	orderService := service.New(orderStore)
	orderHandler := handler.New(orderService)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/orders", orderHandler.CreateOrder)
	r.Get("/orders", orderHandler.ListOrders)
	r.Get("/orders/{id}", orderHandler.GetOrder)
	r.Patch("/orders/{id}", orderHandler.UpdateStatus)
	r.Delete("/orders/{id}", orderHandler.DeleteOrder)

	slog.Info("orders server starting", "addr", ":9090")
	if err := http.ListenAndServe(":9090", r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}