package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/credit"
)

func NewPostgresBalanceSnapshotDBAdapter(db *DBClient) credit.BalanceSnapshotDBConnector {
	return postgresadapter.NewPostgresBalanceSnapshotDBAdapter(db)
}

func NewPostgresGrantDBAdapter(db *DBClient) credit.GrantDBConnector {
	return postgresadapter.NewPostgresGrantDBAdapter(db)
}
