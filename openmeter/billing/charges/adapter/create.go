package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type chargeWithOriginalIntent struct {
	entity *db.Charge
	orig   charges.ChargeIntent
}

func (a *adapter) CreateCharges(ctx context.Context, in charges.CreateChargeInputs) (charges.Charges, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	if len(in.Intents) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charges, error) {
		// Step 1: Create all parent Charge entities in bulk.
		chargeCreates, err := slicesx.MapWithErr(in.Intents, func(ch charges.ChargeIntent) (*db.ChargeCreate, error) {
			return tx.buildCreateCharge(ctx, in.Namespace, ch)
		})
		if err != nil {
			return nil, err
		}

		createdEntities, err := tx.db.Charge.CreateBulk(chargeCreates...).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating charges: %w", err)
		}

		pairs := lo.Map(createdEntities, func(entity *db.Charge, idx int) chargeWithOriginalIntent {
			return chargeWithOriginalIntent{entity: entity, orig: in.Intents[idx]}
		})

		// Step 2: Create FlatFee sub-entities in bulk.
		ffPairs := lo.Filter(pairs, func(p chargeWithOriginalIntent, _ int) bool {
			return p.entity.Type == charges.ChargeTypeFlatFee
		})

		ffCreates, err := slicesx.MapWithErr(ffPairs, func(p chargeWithOriginalIntent) (*db.ChargeFlatFeeCreate, error) {
			ff, err := p.orig.AsFlatFeeIntent()
			if err != nil {
				return nil, err
			}

			return tx.buildCreateFlatFeeCharge(ctx, in.Namespace, p.entity.ID, ff)
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
		ubPairs := lo.Filter(pairs, func(p chargeWithOriginalIntent, _ int) bool {
			return p.entity.Type == charges.ChargeTypeUsageBased
		})

		ubCreates, err := slicesx.MapWithErr(ubPairs, func(p chargeWithOriginalIntent) (*db.ChargeUsageBasedCreate, error) {
			ub, err := p.orig.AsUsageBasedIntent()
			if err != nil {
				return nil, err
			}

			return tx.buildCreateUsageBasedCharge(ctx, in.Namespace, p.entity.ID, ub)
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
		cpPairs := lo.Filter(pairs, func(p chargeWithOriginalIntent, _ int) bool {
			return p.entity.Type == charges.ChargeTypeCreditPurchase
		})

		cpCreates, err := slicesx.MapWithErr(cpPairs, func(p chargeWithOriginalIntent) (*db.ChargeCreditPurchaseCreate, error) {
			cp, err := p.orig.AsCreditPurchaseIntent()
			if err != nil {
				return nil, err
			}

			return tx.buildCreateCreditPurchaseCharge(in.Namespace, p.entity.ID, cp), nil
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

func (a *adapter) buildCreateCharge(ctx context.Context, ns string, in charges.ChargeIntent) (*db.ChargeCreate, error) {
	var meta charges.IntentMeta

	switch in.Type() {
	case charges.ChargeTypeFlatFee:
		ff, err := in.AsFlatFeeIntent()
		if err != nil {
			return nil, err
		}
		meta = ff.IntentMeta

	case charges.ChargeTypeUsageBased:
		ub, err := in.AsUsageBasedIntent()
		if err != nil {
			return nil, err
		}
		meta = ub.IntentMeta

	case charges.ChargeTypeCreditPurchase:
		cp, err := in.AsCreditPurchaseIntent()
		if err != nil {
			return nil, err
		}
		meta = cp.IntentMeta
	}

	create := a.db.Charge.Create().
		SetNamespace(ns).
		SetName(meta.Name).
		SetNillableDescription(meta.Description).
		SetCustomerID(meta.CustomerID).
		SetServicePeriodFrom(meta.ServicePeriod.From.UTC()).
		SetServicePeriodTo(meta.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(meta.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(meta.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(meta.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(meta.FullServicePeriod.To.UTC()).
		SetType(in.Type()).
		SetStatus(charges.ChargeStatusCreated).
		SetCurrency(meta.Currency).
		SetManagedBy(meta.ManagedBy).
		SetNillableUniqueReferenceID(meta.UniqueReferenceID)

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

	return create, nil
}

func (a *adapter) buildCreateFlatFeeCharge(ctx context.Context, ns string, chargeID string, in charges.FlatFeeIntent) (*db.ChargeFlatFeeCreate, error) {
	var discounts *productcatalog.Discounts
	if in.PercentageDiscounts != nil {
		discounts = &productcatalog.Discounts{Percentage: in.PercentageDiscounts}
	}

	proRating, err := proRatingConfigToDB(in.ProRating)
	if err != nil {
		return nil, err
	}

	create := a.db.ChargeFlatFee.Create().
		SetID(chargeID).
		SetChargeID(chargeID).
		SetNamespace(ns).
		SetPaymentTerm(in.PaymentTerm).
		SetInvoiceAt(in.InvoiceAt).
		SetSettlementMode(in.SettlementMode).
		SetNillableFeatureKey(lo.EmptyableToPtr(in.FeatureKey)).
		SetProRating(proRating).
		SetAmountBeforeProration(in.AmountBeforeProration).
		SetAmountAfterProration(in.AmountAfterProration)

	if discounts != nil {
		create = create.SetDiscounts(discounts)
	}

	return create, nil
}

func (a *adapter) buildCreateUsageBasedCharge(ctx context.Context, ns string, chargeID string, in charges.UsageBasedIntent) (*db.ChargeUsageBasedCreate, error) {
	create := a.db.ChargeUsageBased.Create().
		SetID(chargeID).
		SetChargeID(chargeID).
		SetNamespace(ns).
		SetPrice(&in.Price).
		SetFeatureKey(in.FeatureKey).
		SetInvoiceAt(in.InvoiceAt).
		SetSettlementMode(in.SettlementMode)

	if in.Discounts != nil {
		create = create.SetDiscounts(in.Discounts)
	}

	return create, nil
}

func (a *adapter) buildCreateCreditPurchaseCharge(ns string, chargeID string, in charges.CreditPurchaseIntent) *db.ChargeCreditPurchaseCreate {
	return a.db.ChargeCreditPurchase.Create().
		SetID(chargeID).
		SetChargeID(chargeID).
		SetNamespace(ns).
		SetCreditAmount(in.CreditAmount).
		SetSettlement(in.Settlement).
		SetStatus(charges.InitiatedPaymentSettlementStatus)
}
