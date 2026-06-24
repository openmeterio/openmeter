package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebased "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	dbchargeusagebasedoverride "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedoverride"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mapIntentOverrideFromDB(dbOverride *entdb.ChargeUsageBasedOverride) *usagebased.IntentOverride {
	if dbOverride == nil {
		return nil
	}

	return &usagebased.IntentOverride{
		Name:              dbOverride.Name,
		Description:       dbOverride.Description,
		Metadata:          lo.FromPtr(dbOverride.Metadata),
		TaxBehavior:       dbOverride.TaxBehavior,
		TaxCodeID:         dbOverride.TaxCodeID,
		IntentDeletedAt:   convert.TimePtrIn(dbOverride.IntentDeletedAt, time.UTC),
		ServicePeriod:     closedPeriodFromDB(dbOverride.ServicePeriodFrom, dbOverride.ServicePeriodTo),
		FullServicePeriod: closedPeriodFromDB(dbOverride.FullServicePeriodFrom, dbOverride.FullServicePeriodTo),
		BillingPeriod:     closedPeriodFromDB(dbOverride.BillingPeriodFrom, dbOverride.BillingPeriodTo),
		FeatureKey:        dbOverride.FeatureKey,
		Price:             *dbOverride.Price,
		Discounts:         *dbOverride.Discounts,
	}
}

func (a *adapter) CreateChargeOverride(ctx context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	if charge.IntentOverride == nil {
		return usagebased.ChargeBase{}, errors.New("intent override is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		intentOverride, err := tx.createIntentOverride(ctx, charge.GetChargeID(), charge.IntentOverride)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		_, err = tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(convert.TimePtrIn(intentOverride.IntentDeletedAt, time.UTC)).
			Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, fmt.Errorf("updating usage based effective deleted at: %w", err)
		}

		charge.IntentOverride = intentOverride
		charge.DeletedAt = intentOverride.IntentDeletedAt

		return charge, nil
	})
}

func (a *adapter) DeleteChargeOverride(ctx context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	charge.IntentOverride = nil
	if err := charge.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		_, err := tx.db.ChargeUsageBasedOverride.Delete().
			Where(dbchargeusagebasedoverride.NamespaceEQ(charge.Namespace)).
			Where(dbchargeusagebasedoverride.ChargeIDEQ(charge.ID)).
			Exec(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, fmt.Errorf("deleting usage based intent override: %w", err)
		}

		_, err = tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(convert.TimePtrIn(charge.Intent.IntentDeletedAt, time.UTC)).
			Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, fmt.Errorf("updating usage based effective deleted at: %w", err)
		}

		charge.DeletedAt = charge.Intent.IntentDeletedAt

		return charge, nil
	})
}

func (a *adapter) createIntentOverride(ctx context.Context, chargeID meta.ChargeID, override *usagebased.IntentOverride) (*usagebased.IntentOverride, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge id: %w", err)
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validating intent override: %w", err)
	}

	create := a.db.ChargeUsageBasedOverride.Create().
		SetNamespace(chargeID.Namespace).
		SetChargeID(chargeID.ID).
		SetUsageBasedID(chargeID.ID).
		SetName(normalized.Name).
		SetNillableDescription(normalized.Description).
		SetNillableTaxBehavior(normalized.TaxBehavior).
		SetNillableTaxCodeID(normalized.TaxCodeID).
		SetNillableIntentDeletedAt(convert.TimePtrIn(normalized.IntentDeletedAt, time.UTC)).
		SetServicePeriodFrom(normalized.ServicePeriod.From.UTC()).
		SetServicePeriodTo(normalized.ServicePeriod.To.UTC()).
		SetFullServicePeriodFrom(normalized.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(normalized.FullServicePeriod.To.UTC()).
		SetBillingPeriodFrom(normalized.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(normalized.BillingPeriod.To.UTC()).
		SetFeatureKey(normalized.FeatureKey).
		SetPrice(&normalized.Price).
		SetDiscounts(&normalized.Discounts)
	if normalized.Metadata != nil {
		create = create.SetMetadata(&normalized.Metadata)
	}

	dbOverride, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return mapIntentOverrideFromDB(dbOverride), nil
}

func (a *adapter) updateIntentOverride(ctx context.Context, chargeID meta.ChargeID, override *usagebased.IntentOverride) (*entdb.ChargeUsageBasedOverride, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge id: %w", err)
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validating intent override: %w", err)
	}

	update := a.db.ChargeUsageBasedOverride.Update().
		Where(dbchargeusagebasedoverride.NamespaceEQ(chargeID.Namespace)).
		Where(dbchargeusagebasedoverride.ChargeIDEQ(chargeID.ID)).
		SetName(normalized.Name).
		SetOrClearDescription(normalized.Description).
		SetOrClearTaxBehavior(normalized.TaxBehavior).
		SetOrClearTaxCodeID(normalized.TaxCodeID).
		SetOrClearIntentDeletedAt(convert.TimePtrIn(normalized.IntentDeletedAt, time.UTC)).
		SetServicePeriodFrom(normalized.ServicePeriod.From.UTC()).
		SetServicePeriodTo(normalized.ServicePeriod.To.UTC()).
		SetFullServicePeriodFrom(normalized.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(normalized.FullServicePeriod.To.UTC()).
		SetBillingPeriodFrom(normalized.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(normalized.BillingPeriod.To.UTC()).
		SetFeatureKey(normalized.FeatureKey).
		SetPrice(&normalized.Price).
		SetDiscounts(&normalized.Discounts)
	if normalized.Metadata == nil {
		update = update.ClearMetadata()
	} else {
		update = update.SetMetadata(&normalized.Metadata)
	}

	affectedRows, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	if affectedRows == 0 {
		return nil, fmt.Errorf("intent override is not created")
	}

	dbOverride, err := a.db.ChargeUsageBasedOverride.Query().
		Where(dbchargeusagebasedoverride.NamespaceEQ(chargeID.Namespace)).
		Where(dbchargeusagebasedoverride.ChargeIDEQ(chargeID.ID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying updated intent override: %w", err)
	}

	return dbOverride, nil
}

func closedPeriodFromDB(from, to time.Time) timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: from.UTC(),
		To:   to.UTC(),
	}
}
