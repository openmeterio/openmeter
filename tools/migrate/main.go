package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// driver := testutils.InitPostgresDB(t)
	driver, err := entutils.GetPGDriver("postgres://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		logger.Error("failed to get pg driver", err)
		os.Exit(1)
	}

	// initialize client & run migrations
	dbClient := db.NewClient(db.Driver(driver))

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		logger.Error("failed to create schema", err)
		os.Exit(1)
	}
}
