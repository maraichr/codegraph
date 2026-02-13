package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting scheduler")

	// TODO: Initialize scheduler with:
	// - Cron-based index run scheduling
	// - Webhook-triggered re-indexing
	// - Job queue management via Valkey

	<-ctx.Done()
	logger.Info("scheduler stopped")
}
