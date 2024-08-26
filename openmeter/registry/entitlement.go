package registry

import (
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type Entitlement struct {
	Feature            productcatalog.FeatureConnector
	FeatureRepo        productcatalog.FeatureRepo
	EntitlementOwner   grant.OwnerConnector
	CreditBalance      credit.BalanceConnector
	Grant              credit.GrantConnector
	GrantRepo          grant.Repo
	MeteredEntitlement meteredentitlement.Connector
	Entitlement        entitlement.Connector
	EntitlementRepo    entitlement.EntitlementRepo
}

type EntitlementOptions struct {
	DatabaseClient     *db.Client
	StreamingConnector streaming.Connector
	Logger             *slog.Logger
	MeterRepository    meter.Repository
	Publisher          eventbus.Publisher
}
