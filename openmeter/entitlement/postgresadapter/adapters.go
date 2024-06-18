package postgresadapter

import (
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

func NewPostgresEntitlementDBAdapter(db *DBClient) entitlement.EntitlementDBConnector {
	return postgresadapter.NewPostgresEntitlementDBAdapter(db)
}

func NewPostgresUsageResetDBAdapter(db *DBClient) entitlement.UsageResetDBConnector {
	return postgresadapter.NewPostgresUsageResetDBAdapter(db)
}
