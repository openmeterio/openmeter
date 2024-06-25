package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
)

func NewPostgresEntitlementDBAdapter(db *DBClient) entitlement.EntitlementRepo {
	return postgresadapter.NewPostgresEntitlementRepo(db)
}

func NewPostgresUsageResetDBAdapter(db *DBClient) meteredentitlement.UsageResetRepo {
	return postgresadapter.NewPostgresUsageResetRepo(db)
}
