package credit

import (
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/migrate"
)

func NewSchema(driver *sql.Driver) *migrate.Schema {
	return db.NewClient(db.Driver(driver)).Schema
}
