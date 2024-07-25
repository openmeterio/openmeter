package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func main() {
	PG_URL := os.Getenv("POSTGRES_URL")
	if PG_URL == "" {
		PG_URL = "postgres://postgres:postgres@localhost:5432/postgres"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	driver, err := entutils.GetPGDriver(PG_URL)
	if err != nil {
		logger.Error("failed to get pg driver", slog.Any("error", err))
		os.Exit(1)
	}

	// initialize client & run migrations
	dbClient := db.NewClient(db.Driver(driver))

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		logger.Error("failed to create schema", slog.Any("error", err))
		os.Exit(1)
	}
}
