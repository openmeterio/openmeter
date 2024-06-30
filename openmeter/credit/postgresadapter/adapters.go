package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/credit"
)

type BalanceSnapshotConfig = postgresadapter.BalanceSnapshotConfig

func NewPostgresBalanceSnapshotDBAdapter(db *DBClient, config BalanceSnapshotConfig) credit.BalanceSnapshotConnector {
	return postgresadapter.NewPostgresBalanceSnapshotRepo(db, config)
}

func NewPostgresGrantDBAdapter(db *DBClient) credit.GrantRepo {
	return postgresadapter.NewPostgresGrantRepo(db)
}
