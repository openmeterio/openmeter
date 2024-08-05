package adapter

import (
	postgresadapter "github.com/openmeterio/openmeter/internal/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entdb"
)

func NewPostgresBalanceSnapshotRepo(db *entdb.DBClient) credit.BalanceSnapshotRepo {
	return postgresadapter.NewPostgresBalanceSnapshotRepo(db)
}

func NewPostgresGrantRepo(db *entdb.DBClient) credit.GrantRepo {
	return postgresadapter.NewPostgresGrantRepo(db)
}
