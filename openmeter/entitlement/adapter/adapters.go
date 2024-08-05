package adapter

import (
	"github.com/openmeterio/openmeter/internal/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entdb"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
)

func NewPostgresEntitlementDBAdapter(db *entdb.DBClient) entitlement.EntitlementRepo {
	return adapter.NewPostgresEntitlementRepo(db)
}

func NewPostgresUsageResetDBAdapter(db *entdb.DBClient) meteredentitlement.UsageResetRepo {
	return adapter.NewPostgresUsageResetRepo(db)
}
