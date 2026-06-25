package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	dbchargeflatfeeoverride "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeoverride"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mapIntentOverrideFromDB(dbOverride *entdb.ChargeFlatFeeOverride) *flatfee.IntentOverride {
	if dbOverride == nil {
		return nil
	}

	return &flatfee.IntentOverride{
		Name:                  dbOverride.Name,
		Description:           dbOverride.Description,
		Metadata:              lo.FromPtr(dbOverride.Metadata),
		TaxBehavior:           dbOverride.TaxBehavior,
		TaxCodeID:             dbOverride.TaxCodeID,
		IntentDeletedAt:       convert.TimePtrIn(dbOverride.IntentDeletedAt, time.UTC),
		ServicePeriod:         closedPeriodFromDB(dbOverride.ServicePeriodFrom, dbOverride.ServicePeriodTo),
		FullServicePeriod:     closedPeriodFromDB(dbOverride.FullServicePeriodFrom, dbOverride.FullServicePeriodTo),
		BillingPeriod:         closedPeriodFromDB(dbOverride.BillingPeriodFrom, dbOverride.BillingPeriodTo),
		InvoiceAt:             dbOverride.InvoiceAt.UTC(),
		FeatureKey:            lo.FromPtr(dbOverride.FeatureKey),
		PaymentTerm:           dbOverride.PaymentTerm,
		ProRating:             lo.FromPtr(dbOverride.ProRating),
		AmountBeforeProration: dbOverride.AmountBeforeProration,
		PercentageDiscounts:   dbOverride.PercentageDiscounts,
	}
}

func (a *adapter) CreateChargeOverride(ctx context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if charge.IntentOverride == nil {
		return flatfee.ChargeBase{}, errors.New("intent override is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.ChargeBase, error) {
		intentOverride, err := tx.createIntentOverride(ctx, charge.GetChargeID(), charge.IntentOverride)
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		deletedAt := convert.TimePtrIn(intentOverride.IntentDeletedAt, time.UTC)
		_, err = tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, fmt.Errorf("updating flat fee effective deleted at: %w", err)
		}

		charge.IntentOverride = intentOverride
		charge.DeletedAt = deletedAt

		return charge, nil
	})
}

func (a *adapter) DeleteChargeOverride(ctx context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	charge.IntentOverride = nil
	if err := charge.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.ChargeBase, error) {
		affectedRows, err := tx.db.ChargeFlatFeeOverride.Delete().
			Where(dbchargeflatfeeoverride.NamespaceEQ(charge.Namespace)).
			Where(dbchargeflatfeeoverride.ChargeIDEQ(charge.ID)).
			Exec(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, fmt.Errorf("deleting flat fee intent override: %w", err)
		}

		if affectedRows == 0 {
			return flatfee.ChargeBase{}, fmt.Errorf("intent override does not exist")
		}

		deletedAt := convert.TimePtrIn(charge.Intent.IntentDeletedAt, time.UTC)
		_, err = tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, fmt.Errorf("updating flat fee effective deleted at: %w", err)
		}

		charge.DeletedAt = deletedAt

		return charge, nil
	})
}

func (a *adapter) createIntentOverride(ctx context.Context, chargeID meta.ChargeID, override *flatfee.IntentOverride) (*flatfee.IntentOverride, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge id: %w", err)
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validating intent override: %w", err)
	}

	create := a.db.ChargeFlatFeeOverride.Create().
		SetNamespace(chargeID.Namespace).
		SetChargeID(chargeID.ID).
		SetFlatFeeID(chargeID.ID).
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
		SetInvoiceAt(normalized.InvoiceAt.UTC()).
		SetNillableFeatureKey(lo.EmptyableToPtr(normalized.FeatureKey)).
		SetPaymentTerm(normalized.PaymentTerm).
		SetProRating(&normalized.ProRating).
		SetAmountBeforeProration(normalized.AmountBeforeProration)
	if normalized.Metadata != nil {
		create = create.SetMetadata(&normalized.Metadata)
	}
	if normalized.PercentageDiscounts != nil {
		create = create.SetPercentageDiscounts(normalized.PercentageDiscounts)
	}

	dbOverride, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return mapIntentOverrideFromDB(dbOverride), nil
}

func (a *adapter) updateIntentOverride(ctx context.Context, chargeID meta.ChargeID, override *flatfee.IntentOverride) (*entdb.ChargeFlatFeeOverride, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge id: %w", err)
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validating intent override: %w", err)
	}

	update := a.db.ChargeFlatFeeOverride.Update().
		Where(dbchargeflatfeeoverride.NamespaceEQ(chargeID.Namespace)).
		Where(dbchargeflatfeeoverride.ChargeIDEQ(chargeID.ID)).
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
		SetInvoiceAt(normalized.InvoiceAt.UTC()).
		SetOrClearFeatureKey(lo.EmptyableToPtr(normalized.FeatureKey)).
		SetPaymentTerm(normalized.PaymentTerm).
		SetProRating(&normalized.ProRating).
		SetAmountBeforeProration(normalized.AmountBeforeProration)
	if normalized.Metadata == nil {
		update = update.ClearMetadata()
	} else {
		update = update.SetMetadata(&normalized.Metadata)
	}
	if normalized.PercentageDiscounts == nil {
		update = update.ClearPercentageDiscounts()
	} else {
		update = update.SetPercentageDiscounts(normalized.PercentageDiscounts)
	}

	affectedRows, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating intent override for charge[%s]: %w", chargeID.ID, err)
	}

	if affectedRows == 0 {
		return nil, fmt.Errorf("intent override does not exist for charge[%s]", chargeID.ID)
	}

	dbOverride, err := a.db.ChargeFlatFeeOverride.Query().
		Where(dbchargeflatfeeoverride.NamespaceEQ(chargeID.Namespace)).
		Where(dbchargeflatfeeoverride.ChargeIDEQ(chargeID.ID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying updated intent override for charge[%s]: %w", chargeID.ID, err)
	}

	return dbOverride, nil
}

func closedPeriodFromDB(from, to time.Time) timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: from.UTC(),
		To:   to.UTC(),
	}
}
