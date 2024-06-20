package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

func NewPostgresEntitlementDBAdapter(db *DBClient) entitlement.EntitlementRepo {
	return postgresadapter.NewPostgresEntitlementRepo(db)
}

func NewPostgresUsageResetDBAdapter(db *DBClient) entitlement.UsageResetRepo {
	return postgresadapter.NewPostgresUsageResetRepo(db)
}
