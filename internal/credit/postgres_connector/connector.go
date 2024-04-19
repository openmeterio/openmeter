package postgres_connector

import (
	"log/slog"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
)

type PostgresConnector struct {
	logger             *slog.Logger
	db                 *db.Client
	streamingConnector streaming.Connector
	meterRepository    meter.Repository
	lockManager        credit_model.LockManager
}

// Implement the Connector interface
var _ credit.Connector = &PostgresConnector{}

func NewPostgresConnector(
	logger *slog.Logger,
	db *db.Client,
	streamingConnector streaming.Connector,
	meterRepository meter.Repository,
	lockManager credit_model.LockManager,
) credit.Connector {
	connector := PostgresConnector{
		logger:             logger,
		db:                 db,
		streamingConnector: streamingConnector,
		meterRepository:    meterRepository,
		lockManager:        lockManager,
	}

	return &connector
}
