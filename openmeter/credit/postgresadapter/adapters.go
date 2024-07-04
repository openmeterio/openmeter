package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/credit"
)

func NewPostgresBalanceSnapshotDBAdapter(db *DBClient) credit.BalanceSnapshotRepo {
	return postgresadapter.NewPostgresBalanceSnapshotRepo(db)
}

func NewPostgresGrantDBAdapter(db *DBClient) credit.GrantRepo {
	return postgresadapter.NewPostgresGrantRepo(db)
}
