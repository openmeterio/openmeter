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

func fromDBOverride(dbOverride *entdb.ChargeUsageBasedOverride) *usagebased.IntentMutableFields {
	if dbOverride == nil {
		return nil
	}

	return &usagebased.IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			Name:              dbOverride.Name,
			Description:       dbOverride.Description,
			Metadata:          lo.FromPtr(dbOverride.Metadata),
			ServicePeriod:     fromDBClosedPeriod(dbOverride.ServicePeriodFrom, dbOverride.ServicePeriodTo),
			FullServicePeriod: fromDBClosedPeriod(dbOverride.FullServicePeriodFrom, dbOverride.FullServicePeriodTo),
			BillingPeriod:     fromDBClosedPeriod(dbOverride.BillingPeriodFrom, dbOverride.BillingPeriodTo),
		},
		IntentDeletedAt: convert.TimePtrIn(dbOverride.IntentDeletedAt, time.UTC),
		InvoiceAt:       dbOverride.InvoiceAt.UTC(),
		Price:           lo.FromPtr(dbOverride.Price),
		Discounts:       lo.FromPtr(dbOverride.Discounts),
		UnitConfig:      dbOverride.UnitConfig,
	}
}

func (a *adapter) CreateChargeOverride(ctx context.Context, charge usagebased.ChargeBase, override usagebased.IntentMutableFields) (usagebased.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	if err := override.Validate(); err != nil {
		return usagebased.ChargeBase{}, fmt.Errorf("validating intent override: %w", err)
	}

	if charge.Intent.HasOverrideLayer() {
		return usagebased.ChargeBase{}, errors.New("intent override already exists")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		dbIntentOverride, err := tx.createIntentOverride(ctx, charge.GetChargeID(), override)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		deletedAt := convert.TimePtrIn(dbIntentOverride.IntentDeletedAt, time.UTC)
		dbCharge, err := tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, fmt.Errorf("updating usage based effective deleted at: %w", err)
		}

		dbCharge.Edges.IntentOverride = dbIntentOverride

		return fromDBBaseWithCurrency(dbCharge, charge.Intent.GetBaseIntent().Currency)
	})
}

func (a *adapter) DeleteChargeOverride(ctx context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	if !charge.Intent.HasOverrideLayer() {
		return usagebased.ChargeBase{}, errors.New("intent override is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		affectedRows, err := tx.db.ChargeUsageBasedOverride.Delete().
			Where(dbchargeusagebasedoverride.NamespaceEQ(charge.Namespace)).
			Where(dbchargeusagebasedoverride.ChargeIDEQ(charge.ID)).
			Exec(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, fmt.Errorf("deleting usage based intent override: %w", err)
		}

		if affectedRows == 0 {
			return usagebased.ChargeBase{}, fmt.Errorf("intent override does not exist")
		}

		baseIntent := charge.Intent.GetBaseIntent()
		deletedAt := convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)
		_, err = tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, fmt.Errorf("updating usage based effective deleted at: %w", err)
		}

		charge.Intent = baseIntent.AsOverridableIntent()
		charge.DeletedAt = deletedAt

		return charge, nil
	})
}

func (a *adapter) createIntentOverride(ctx context.Context, chargeID meta.ChargeID, override usagebased.IntentMutableFields) (*entdb.ChargeUsageBasedOverride, error) {
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
		SetNillableIntentDeletedAt(convert.TimePtrIn(normalized.IntentDeletedAt, time.UTC)).
		SetServicePeriodFrom(normalized.ServicePeriod.From.UTC()).
		SetServicePeriodTo(normalized.ServicePeriod.To.UTC()).
		SetFullServicePeriodFrom(normalized.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(normalized.FullServicePeriod.To.UTC()).
		SetBillingPeriodFrom(normalized.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(normalized.BillingPeriod.To.UTC()).
		SetInvoiceAt(normalized.InvoiceAt.UTC()).
		SetPrice(&normalized.Price).
		SetDiscounts(&normalized.Discounts)
	if normalized.UnitConfig != nil {
		create = create.SetUnitConfig(normalized.UnitConfig)
	}
	if normalized.Metadata != nil {
		create = create.SetMetadata(&normalized.Metadata)
	}

	return create.Save(ctx)
}

func (a *adapter) updateIntentOverride(ctx context.Context, chargeID meta.ChargeID, override *usagebased.IntentMutableFields) (*entdb.ChargeUsageBasedOverride, error) {
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
		SetOrClearIntentDeletedAt(convert.TimePtrIn(normalized.IntentDeletedAt, time.UTC)).
		SetServicePeriodFrom(normalized.ServicePeriod.From.UTC()).
		SetServicePeriodTo(normalized.ServicePeriod.To.UTC()).
		SetFullServicePeriodFrom(normalized.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(normalized.FullServicePeriod.To.UTC()).
		SetBillingPeriodFrom(normalized.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(normalized.BillingPeriod.To.UTC()).
		SetInvoiceAt(normalized.InvoiceAt.UTC()).
		SetPrice(&normalized.Price).
		SetDiscounts(&normalized.Discounts)
	if normalized.UnitConfig != nil {
		update = update.SetUnitConfig(normalized.UnitConfig)
	} else {
		update = update.ClearUnitConfig()
	}
	if normalized.Metadata == nil {
		update = update.ClearMetadata()
	} else {
		update = update.SetMetadata(&normalized.Metadata)
	}

	affectedRows, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating intent override for charge[%s]: %w", chargeID.ID, err)
	}

	if affectedRows == 0 {
		return nil, fmt.Errorf("intent override does not exist for charge[%s]", chargeID.ID)
	}

	dbOverride, err := a.db.ChargeUsageBasedOverride.Query().
		Where(dbchargeusagebasedoverride.NamespaceEQ(chargeID.Namespace)).
		Where(dbchargeusagebasedoverride.ChargeIDEQ(chargeID.ID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying updated intent override for charge[%s]: %w", chargeID.ID, err)
	}

	return dbOverride, nil
}

func fromDBClosedPeriod(from, to time.Time) timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: from.UTC(),
		To:   to.UTC(),
	}
}
