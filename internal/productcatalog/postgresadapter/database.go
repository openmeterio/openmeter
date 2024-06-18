package postgresadapter

import (
	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db"
)

func NewClient(drv *entsql.Driver) (*db.Client, error) {
	return db.NewClient(db.Driver(drv)), nil
}
