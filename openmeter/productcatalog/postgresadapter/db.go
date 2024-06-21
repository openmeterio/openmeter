package postgresadapter

import (
	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db"
)

type DBClient = db.Client

func NewClient(drv *entsql.Driver) (*DBClient, error) {
	return db.NewClient(db.Driver(drv)), nil
}
