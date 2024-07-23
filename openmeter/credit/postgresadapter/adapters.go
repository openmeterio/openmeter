package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entdb"
)

func NewPostgresBalanceSnapshotDBAdapter(db *entdb.DBClient) credit.BalanceSnapshotRepo {
	return postgresadapter.NewPostgresBalanceSnapshotRepo(db)
}

func NewPostgresGrantDBAdapter(db *entdb.DBClient) credit.GrantRepo {
	return postgresadapter.NewPostgresGrantRepo(db)
}
