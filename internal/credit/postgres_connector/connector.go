package postgres_connector

import (
	"log/slog"
	"time"

	"entgo.io/ent"
	"github.com/openmeterio/openmeter/internal/cdc"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming"
)

type PostgresConnector struct {
	logger             *slog.Logger
	db                 *db.Client
	streamingConnector streaming.Connector
	meterRepository    meter.Repository
	config             PostgresConnectorConfig
}

// Implement the Connector interface
var _ credit.Connector = &PostgresConnector{}

type PostgresConnectorConfig struct {
	WindowSize time.Duration
}

func NewPostgresConnector(
	logger *slog.Logger,
	db *db.Client,
	streamingConnector streaming.Connector,
	meterRepository meter.Repository,
	config PostgresConnectorConfig,
) credit.Connector {

	db.Use(ent.Hook(cdc.NewMutationSink().MutatorFunc()))
	connector := PostgresConnector{
		logger:             logger,
		db:                 db,
		streamingConnector: streamingConnector,
		meterRepository:    meterRepository,
		config:             config,
	}

	return &connector
}
