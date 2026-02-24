package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type chargeWithOriginal struct {
	entity *db.Charge
	orig   charges.Charge
}

func (a *adapter) CreateCharges(ctx context.Context, input charges.Charges) (charges.Charges, error) {
	if len(input) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charges, error) {
		// Step 1: Create all parent Charge entities in bulk.
		chargeCreates, err := slicesx.MapWithErr(input, func(ch charges.Charge) (*db.ChargeCreate, error) {
			gc, err := ch.AsGenericCharge()
			if err != nil {
				return nil, err
			}

			return tx.buildCreateCharge(ctx, gc), nil
		})
		if err != nil {
			return nil, err
		}

		createdEntities, err := tx.db.Charge.CreateBulk(chargeCreates...).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating charges: %w", err)
		}

		pairs := lo.Map(createdEntities, func(entity *db.Charge, idx int) chargeWithOriginal {
			return chargeWithOriginal{entity: entity, orig: input[idx]}
		})

		// Step 2: Create FlatFee sub-entities in bulk.
		ffPairs := lo.Filter(pairs, func(p chargeWithOriginal, _ int) bool {
			return p.entity.Type == charges.ChargeTypeFlatFee
		})

		ffCreates, err := slicesx.MapWithErr(ffPairs, func(p chargeWithOriginal) (*db.ChargeFlatFeeCreate, error) {
			ff, err := p.orig.AsFlatFeeCharge()
			if err != nil {
				return nil, err
			}

			ff.ID = p.entity.ID
			return tx.buildCreateFlatFeeCharge(ctx, ff)
		})
		if err != nil {
			return nil, err
		}

		if len(ffCreates) > 0 {
			ffEntities, err := tx.db.ChargeFlatFee.CreateBulk(ffCreates...).Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("creating flat fee charges: %w", err)
			}

			ffByID := lo.SliceToMap(ffEntities, func(e *db.ChargeFlatFee) (string, *db.ChargeFlatFee) {
				return e.ID, e
			})

			for _, p := range ffPairs {
				p.entity.Edges.FlatFee = ffByID[p.entity.ID]
			}
		}

		// Step 3: Create UsageBased sub-entities in bulk.
		ubPairs := lo.Filter(pairs, func(p chargeWithOriginal, _ int) bool {
			return p.entity.Type == charges.ChargeTypeUsageBased
		})

		ubCreates, err := slicesx.MapWithErr(ubPairs, func(p chargeWithOriginal) (*db.ChargeUsageBasedCreate, error) {
			ub, err := p.orig.AsUsageBasedCharge()
			if err != nil {
				return nil, err
			}

			ub.ID = p.entity.ID
			return tx.buildCreateUsageBasedCharge(ctx, ub)
		})
		if err != nil {
			return nil, err
		}

		if len(ubCreates) > 0 {
			ubEntities, err := tx.db.ChargeUsageBased.CreateBulk(ubCreates...).Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("creating usage based charges: %w", err)
			}

			ubByID := lo.SliceToMap(ubEntities, func(e *db.ChargeUsageBased) (string, *db.ChargeUsageBased) {
				return e.ID, e
			})

			for _, p := range ubPairs {
				p.entity.Edges.UsageBased = ubByID[p.entity.ID]
			}
		}

		// Step 4: Create CreditPurchase sub-entities in bulk.
		cpPairs := lo.Filter(pairs, func(p chargeWithOriginal, _ int) bool {
			return p.entity.Type == charges.ChargeTypeCreditPurchase
		})

		cpCreates, err := slicesx.MapWithErr(cpPairs, func(p chargeWithOriginal) (*db.ChargeCreditPurchaseCreate, error) {
			cp, err := p.orig.AsCreditPurchase()
			if err != nil {
				return nil, err
			}

			cp.ID = p.entity.ID
			return tx.buildCreateCreditPurchaseCharge(ctx, cp), nil
		})
		if err != nil {
			return nil, err
		}

		if len(cpCreates) > 0 {
			cpEntities, err := tx.db.ChargeCreditPurchase.CreateBulk(cpCreates...).Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("creating credit purchase charges: %w", err)
			}

			cpByID := lo.SliceToMap(cpEntities, func(e *db.ChargeCreditPurchase) (string, *db.ChargeCreditPurchase) {
				return e.ID, e
			})

			for _, p := range cpPairs {
				p.entity.Edges.CreditPurchase = cpByID[p.entity.ID]
			}
		}

		// Step 5: Map all created entities back to domain objects.
		return slicesx.MapWithErr(createdEntities, func(entity *db.Charge) (charges.Charge, error) {
			return MapChargeFromDB(entity)
		})
	})
}

func (a *adapter) buildCreateCharge(ctx context.Context, in charges.GenericCharge) *db.ChargeCreate {
	mr := in.GetManagedResource()
	meta := in.GetIntentMeta()
	chargeType := in.Type()
	status := in.GetStatus()

	create := a.db.Charge.Create().
		SetNamespace(mr.Namespace).
		SetName(mr.Name).
		SetNillableDescription(mr.Description).
		SetCustomerID(meta.CustomerID).
		SetServicePeriodFrom(meta.ServicePeriod.From.UTC()).
		SetServicePeriodTo(meta.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(meta.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(meta.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(meta.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(meta.FullServicePeriod.To.UTC()).
		SetType(chargeType).
		SetStatus(status).
		SetCurrency(meta.Currency).
		SetManagedBy(meta.ManagedBy).
		SetNillableUniqueReferenceID(meta.UniqueReferenceID).
		SetNillableDeletedAt(convert.SafeToUTC(mr.DeletedAt))

	if meta.Metadata != nil {
		create = create.SetMetadata(meta.Metadata)
	}

	if meta.Annotations != nil {
		create = create.SetAnnotations(meta.Annotations)
	}

	if meta.Subscription != nil {
		create = create.
			SetNillableSubscriptionID(&meta.Subscription.SubscriptionID).
			SetNillableSubscriptionPhaseID(&meta.Subscription.PhaseID).
			SetNillableSubscriptionItemID(&meta.Subscription.ItemID)
	}

	return create
}

func (a *adapter) buildCreateFlatFeeCharge(ctx context.Context, in charges.FlatFeeCharge) (*db.ChargeFlatFeeCreate, error) {
	var discounts *productcatalog.Discounts
	if in.Intent.PercentageDiscounts != nil {
		discounts = &productcatalog.Discounts{Percentage: in.Intent.PercentageDiscounts}
	}

	proRating, err := proRatingConfigToDB(in.Intent.ProRating)
	if err != nil {
		return nil, err
	}

	create := a.db.ChargeFlatFee.Create().
		SetID(in.ID).
		SetChargeID(in.ID).
		SetNamespace(in.Namespace).
		SetPaymentTerm(in.Intent.PaymentTerm).
		SetInvoiceAt(in.Intent.InvoiceAt).
		SetSettlementMode(in.Intent.SettlementMode).
		SetNillableFeatureKey(lo.EmptyableToPtr(in.Intent.FeatureKey)).
		SetProRating(proRating).
		SetAmountBeforeProration(in.Intent.AmountBeforeProration).
		SetAmountAfterProration(in.Intent.AmountAfterProration)

	if discounts != nil {
		create = create.SetDiscounts(discounts)
	}

	return create, nil
}

func (a *adapter) buildCreateUsageBasedCharge(ctx context.Context, in charges.UsageBasedCharge) (*db.ChargeUsageBasedCreate, error) {
	create := a.db.ChargeUsageBased.Create().
		SetID(in.ID).
		SetChargeID(in.ID).
		SetNamespace(in.Namespace).
		SetPrice(&in.Intent.Price).
		SetFeatureKey(in.Intent.FeatureKey).
		SetInvoiceAt(in.Intent.InvoiceAt).
		SetSettlementMode(in.Intent.SettlementMode)

	if in.Intent.Discounts != nil {
		create = create.SetDiscounts(in.Intent.Discounts)
	}

	return create, nil
}

func (a *adapter) buildCreateCreditPurchaseCharge(_ context.Context, in charges.CreditPurchaseCharge) *db.ChargeCreditPurchaseCreate {
	return a.db.ChargeCreditPurchase.Create().
		SetID(in.ID).
		SetChargeID(in.ID).
		SetNamespace(in.Namespace).
		SetCreditAmount(in.Intent.CreditAmount).
		SetSettlement(in.Intent.Settlement).
		SetStatus(in.State.Status)
}
