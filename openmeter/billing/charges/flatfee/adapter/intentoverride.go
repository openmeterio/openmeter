package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	dbchargeflatfeeoverride "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeoverride"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mapIntentOverrideFromDB(dbOverride *entdb.ChargeFlatFeeOverride) *flatfee.IntentMutableFields {
	if dbOverride == nil {
		return nil
	}

	var percentageDiscounts *billing.PercentageDiscount
	if dbOverride.Discounts != nil {
		percentageDiscounts = dbOverride.Discounts.Percentage
	}

	return &flatfee.IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			Name:        dbOverride.Name,
			Description: dbOverride.Description,
			Metadata:    lo.FromPtr(dbOverride.Metadata),
			TaxConfig: productcatalog.TaxCodeConfig{
				Behavior:  dbOverride.TaxBehavior,
				TaxCodeID: lo.FromPtrOr(dbOverride.TaxCodeID, ""),
			},
			ServicePeriod:     closedPeriodFromDB(dbOverride.ServicePeriodFrom, dbOverride.ServicePeriodTo),
			FullServicePeriod: closedPeriodFromDB(dbOverride.FullServicePeriodFrom, dbOverride.FullServicePeriodTo),
			BillingPeriod:     closedPeriodFromDB(dbOverride.BillingPeriodFrom, dbOverride.BillingPeriodTo),
		},
		IntentDeletedAt:       convert.TimePtrIn(dbOverride.IntentDeletedAt, time.UTC),
		InvoiceAt:             dbOverride.InvoiceAt.UTC(),
		FeatureKey:            lo.FromPtr(dbOverride.FeatureKey),
		PaymentTerm:           dbOverride.PaymentTerm,
		ProRating:             lo.FromPtr(dbOverride.ProRating),
		AmountBeforeProration: dbOverride.AmountBeforeProration,
		PercentageDiscounts:   percentageDiscounts,
	}
}

func (a *adapter) CreateChargeOverride(ctx context.Context, charge flatfee.ChargeBase, override flatfee.IntentMutableFields) (flatfee.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if err := override.Validate(); err != nil {
		return flatfee.ChargeBase{}, fmt.Errorf("validating intent override: %w", err)
	}

	if charge.Intent.HasOverrideLayer() {
		return flatfee.ChargeBase{}, errors.New("intent override already exists")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.ChargeBase, error) {
		dbIntentOverride, err := tx.createIntentOverride(ctx, charge.GetChargeID(), override)
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		deletedAt := convert.TimePtrIn(dbIntentOverride.IntentDeletedAt, time.UTC)
		dbCharge, err := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, fmt.Errorf("updating flat fee effective deleted at: %w", err)
		}

		dbCharge.Edges.IntentOverride = dbIntentOverride

		return MapChargeBaseFromDB(dbCharge), nil
	})
}

func (a *adapter) DeleteChargeOverride(ctx context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if !charge.Intent.HasOverrideLayer() {
		return flatfee.ChargeBase{}, errors.New("intent override is required")
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

		baseIntent := charge.Intent.GetBaseIntent()
		deletedAt := convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)
		_, err = tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetOrClearDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, fmt.Errorf("updating flat fee effective deleted at: %w", err)
		}

		charge.Intent = baseIntent.AsOverridableIntent()
		charge.DeletedAt = deletedAt

		return charge, nil
	})
}

func (a *adapter) createIntentOverride(ctx context.Context, chargeID meta.ChargeID, override flatfee.IntentMutableFields) (*entdb.ChargeFlatFeeOverride, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge id: %w", err)
	}

	normalized := override.Normalized("")
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validating intent override: %w", err)
	}

	create := a.db.ChargeFlatFeeOverride.Create().
		SetNamespace(chargeID.Namespace).
		SetChargeID(chargeID.ID).
		SetFlatFeeID(chargeID.ID).
		SetName(normalized.Name).
		SetNillableDescription(normalized.Description).
		SetNillableTaxBehavior(normalized.TaxConfig.Behavior).
		SetNillableTaxCodeID(lo.EmptyableToPtr(normalized.TaxConfig.TaxCodeID)).
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
		create = create.SetDiscounts(&billing.Discounts{Percentage: normalized.PercentageDiscounts})
	}

	return create.Save(ctx)
}

func (a *adapter) updateIntentOverride(ctx context.Context, chargeID meta.ChargeID, override *flatfee.IntentMutableFields) (*entdb.ChargeFlatFeeOverride, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge id: %w", err)
	}

	normalized := override.Normalized("")
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("validating intent override: %w", err)
	}

	update := a.db.ChargeFlatFeeOverride.Update().
		Where(dbchargeflatfeeoverride.NamespaceEQ(chargeID.Namespace)).
		Where(dbchargeflatfeeoverride.ChargeIDEQ(chargeID.ID)).
		SetName(normalized.Name).
		SetOrClearDescription(normalized.Description).
		SetOrClearTaxBehavior(normalized.TaxConfig.Behavior).
		SetOrClearTaxCodeID(lo.EmptyableToPtr(normalized.TaxConfig.TaxCodeID)).
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
		update = update.ClearDiscounts()
	} else {
		update = update.SetDiscounts(&billing.Discounts{Percentage: normalized.PercentageDiscounts})
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
