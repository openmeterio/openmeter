package registry

import (
	"log/slog"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/event/publisher"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Entitlement struct {
	Feature            productcatalog.FeatureConnector
	EntitlementOwner   credit.OwnerConnector
	CreditBalance      credit.BalanceConnector
	Grant              credit.GrantConnector
	MeteredEntitlement meteredentitlement.Connector
	Entitlement        entitlement.EntitlementConnector
	EntitlementRepo    entitlement.EntitlementRepo
}

type EntitlementOptions struct {
	DatabaseClient     *db.Client
	StreamingConnector streaming.Connector
	Logger             *slog.Logger
	MeterRepository    meter.Repository
	Publisher          publisher.TopicPublisher
}
