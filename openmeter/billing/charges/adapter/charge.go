package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
)

func (a *adapter) updateChargeIntent(ctx context.Context, charge charges.ChargeID, intent charges.IntentMeta, status charges.ChargeStatus) (*entdb.Charge, error) {
	return a.db.Charge.UpdateOneID(charge.ID).
		Where(dbcharge.NamespaceEQ(charge.Namespace)).
		SetName(intent.Name).
		SetNillableDescription(intent.Description).
		SetServicePeriodFrom(intent.ServicePeriod.From.UTC()).
		SetServicePeriodTo(intent.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(intent.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(intent.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(intent.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(intent.FullServicePeriod.To.UTC()).
		SetStatus(status).
		SetManagedBy(intent.ManagedBy).
		SetNillableUniqueReferenceID(intent.UniqueReferenceID).
		SetMetadata(intent.Metadata).
		SetAnnotations(intent.Annotations).
		Save(ctx)
}
