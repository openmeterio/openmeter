package credit

import (
	"log/slog"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/migrate"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewSchema(driver *sql.Driver) *migrate.Schema {
	return db.NewClient(db.Driver(driver)).Schema
}

func NewConnector(
	logger *slog.Logger,
	driver *sql.Driver,
	streamingConnector streaming.Connector,
	meterRepository meter.Repository,
) Connector {
	return postgres_connector.NewPostgresConnector(
		logger, db.NewClient(db.Driver(driver)), streamingConnector, meterRepository,
	)
}
