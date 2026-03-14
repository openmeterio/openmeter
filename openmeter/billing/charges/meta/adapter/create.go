package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) Create(ctx context.Context, in meta.CreateInput) (meta.Charges, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (meta.Charges, error) {
		metas, err := slicesx.MapWithErr(in.Intents, func(intent meta.IntentCreate) (*entdb.ChargeCreate, error) {
			return tx.buildCreateMeta(ctx, in.Namespace, intent)
		})
		if err != nil {
			return nil, err
		}

		createdEntities, err := tx.db.Charge.CreateBulk(metas...).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating charge metas: %w", err)
		}

		return lo.Map(createdEntities, func(entity *entdb.Charge, idx int) meta.Charge {
			return MapChargeFromDB(entity)
		}), nil
	})
}

func (a *adapter) buildCreateMeta(ctx context.Context, ns string, intent meta.IntentCreate) (*entdb.ChargeCreate, error) {
	create := a.db.Charge.Create().
		SetNamespace(ns).
		SetName(intent.Name).
		SetNillableDescription(intent.Description).
		SetCustomerID(intent.CustomerID).
		SetServicePeriodFrom(intent.ServicePeriod.From.UTC()).
		SetServicePeriodTo(intent.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(intent.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(intent.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(intent.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(intent.FullServicePeriod.To.UTC()).
		SetType(intent.Type).
		SetStatus(lo.CoalesceOrEmpty(intent.InitialStatus, meta.ChargeStatusCreated)).
		SetCurrency(intent.Currency).
		SetManagedBy(intent.ManagedBy).
		SetNillableUniqueReferenceID(intent.UniqueReferenceID)

	if intent.Metadata != nil {
		create = create.SetMetadata(intent.Metadata)
	}

	if intent.Annotations != nil {
		create = create.SetAnnotations(intent.Annotations)
	}

	if intent.Subscription != nil {
		create = create.
			SetNillableSubscriptionID(&intent.Subscription.SubscriptionID).
			SetNillableSubscriptionPhaseID(&intent.Subscription.PhaseID).
			SetNillableSubscriptionItemID(&intent.Subscription.ItemID)
	}

	return create, nil
}
