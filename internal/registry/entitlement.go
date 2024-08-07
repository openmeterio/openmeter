package registry

import (
	"log/slog"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Entitlement struct {
	Feature            productcatalog.FeatureConnector
	FeatureRepo        productcatalog.FeatureRepo
	EntitlementOwner   credit.OwnerConnector
	CreditBalance      credit.BalanceConnector
	Grant              credit.GrantConnector
	GrantRepo          credit.GrantRepo
	MeteredEntitlement meteredentitlement.Connector
	Entitlement        entitlement.EntitlementConnector
	EntitlementRepo    entitlement.EntitlementRepo
}

type EntitlementOptions struct {
	DatabaseClient     *db.Client
	StreamingConnector streaming.Connector
	Logger             *slog.Logger
	MeterRepository    meter.Repository
	Publisher          eventbus.Publisher
}
