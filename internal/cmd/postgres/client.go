package postgres

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Clients struct {
	Driver *sql.Driver
	Client *db.Client
}

func GetClients(config config.PostgresConfig) (*Clients, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid postgres config: %w", err)
	}
	driver, err := entutils.GetPGDriver(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres driver: %w", err)
	}

	// initialize client & run migrations
	dbClient := db.NewClient(db.Driver(driver))

	// TODO: use versioned migrations: https://entgo.io/docs/versioned-migrations
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to migrate credit db: %w", err)
	}

	return &Clients{
		Driver: driver,
		Client: dbClient,
	}, nil
}
