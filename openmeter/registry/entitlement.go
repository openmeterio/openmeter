package registry

import (
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type Entitlement struct {
	Feature            feature.FeatureConnector
	FeatureRepo        feature.FeatureRepo
	EntitlementOwner   grant.OwnerConnector
	CreditBalance      credit.BalanceConnector
	Grant              credit.GrantConnector
	GrantRepo          grant.Repo
	MeteredEntitlement meteredentitlement.Connector
	Entitlement        entitlement.Service
	EntitlementRepo    entitlement.EntitlementRepo
}
