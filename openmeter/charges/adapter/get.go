package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (a *adapter) GetChargeByID(ctx context.Context, input charges.ChargeID) (charges.Charge, error) {
	res, err := a.GetChargesByIDs(ctx, input.Namespace, []string{input.ID})
	if err != nil {
		// Note: not found is handled by the GetChargesByIDs function
		return charges.Charge{}, err
	}

	return res[0], nil
}

func (a *adapter) GetChargesByIDs(ctx context.Context, ns string, ids []string) (charges.Charges, error) {
	if ns == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charges, error) {
		entities, err := tx.db.Charge.Query().
			Where(dbcharge.Namespace(ns)).
			Where(dbcharge.IDIn(ids...)).
			WithStandardInvoiceRealizations().
			WithUsageBased().
			WithFlatFee().
			All(ctx)
		if err != nil {
			return nil, err
		}

		entriesById := lo.SliceToMap(entities, func(entity *db.Charge) (string, *db.Charge) {
			return entity.ID, entity
		})

		dbChargesInInputOrder := make([]*db.Charge, len(ids))

		errs := []error{}
		for idx, id := range ids {
			dbCharge, ok := entriesById[id]
			if !ok {
				errs = append(errs, models.NewGenericNotFoundError(
					fmt.Errorf("charge not found [namespace=%s charge.id=%s]", ns, id),
				))
				continue
			}

			dbChargesInInputOrder[idx] = dbCharge
		}
		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}

		return slicesx.MapWithErr(entities, mapChargeFromDB)
	})
}

func mapChargeFromDB(entity *db.Charge) (charges.Charge, error) {
	intent, err := mapIntentFromDB(entity)
	if err != nil {
		return charges.Charge{}, fmt.Errorf("failed to map intent: %w", err)
	}

	return charges.Charge{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			CreatedAt:   entity.CreatedAt.In(time.UTC),
			UpdatedAt:   entity.UpdatedAt.In(time.UTC),
			DeletedAt:   convert.TimePtrIn(entity.DeletedAt, time.UTC),
			Namespace:   entity.Namespace,
			ID:          entity.ID,
			Name:        entity.Name,
			Description: entity.Description,
		}),
		Intent: intent,
		Realizations: charges.Realizations{
			StandardInvoice: lo.Map(entity.Edges.StandardInvoiceRealizations, func(realization *db.ChargeStandardInvoiceRealization, _ int) charges.StandardInvoiceRealization {
				return mapStandardInvoiceRealizationFromDB(realization)
			}),
		},
		Status: entity.Status,
	}, nil
}

func mapIntentFromDB(entity *db.Charge) (charges.Intent, error) {
	intentMeta := charges.IntentMeta{
		Metadata:       entity.Metadata,
		Annotations:    entity.Annotations,
		ManagedBy:      entity.ManagedBy,
		CustomerID:     entity.CustomerID,
		Currency:       entity.Currency,
		SettlementMode: entity.SettlementMode,
		ServicePeriod: timeutil.ClosedPeriod{
			From: entity.ServicePeriodFrom.In(time.UTC),
			To:   entity.ServicePeriodTo.In(time.UTC),
		},
		FullServicePeriod: timeutil.ClosedPeriod{
			From: entity.FullServicePeriodFrom.In(time.UTC),
			To:   entity.FullServicePeriodTo.In(time.UTC),
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: entity.BillingPeriodFrom.In(time.UTC),
			To:   entity.BillingPeriodTo.In(time.UTC),
		},
		InvoiceAt:         entity.InvoiceAt.In(time.UTC),
		TaxConfig:         lo.EmptyableToPtr(entity.TaxConfig),
		UniqueReferenceID: entity.UniqueReferenceID,
		Subscription:      mapSubscriptionFromDB(entity),
	}

	switch entity.Type {
	case charges.IntentTypeFlatFee:
		if entity.Edges.FlatFee == nil {
			return charges.Intent{}, fmt.Errorf("flat fee entity not found for charge %s", entity.ID)
		}

		feeEntity := entity.Edges.FlatFee

		var percentageDiscounts *productcatalog.PercentageDiscount
		if feeEntity.Discounts != nil {
			percentageDiscounts = feeEntity.Discounts.Percentage
		}

		proRating, err := mapProRatingFromDB(feeEntity.ProRating)
		if err != nil {
			return charges.Intent{}, err
		}

		flatFeeIntent := charges.FlatFeeIntent{
			PaymentTerm:         feeEntity.PaymentTerm,
			FeatureKey:          lo.FromPtr(feeEntity.FeatureKey),
			PercentageDiscounts: percentageDiscounts,

			ProRating:             proRating,
			AmountBeforeProration: feeEntity.AmountBeforeProration,
			AmountAfterProration:  feeEntity.AmountAfterProration,
		}

		return charges.NewIntent(intentMeta, flatFeeIntent), nil
	case charges.IntentTypeUsageBased:
		if entity.Edges.UsageBased == nil {
			return charges.Intent{}, fmt.Errorf("usage based entity not found for charge %s", entity.ID)
		}

		usageBasedEntity := entity.Edges.UsageBased

		usageBasedIntent := charges.UsageBasedIntent{
			Price:      lo.FromPtr(usageBasedEntity.Price),
			FeatureKey: usageBasedEntity.FeatureKey,
			Discounts:  usageBasedEntity.Discounts,
		}

		return charges.NewIntent(intentMeta, usageBasedIntent), nil
	default:
		return charges.Intent{}, fmt.Errorf("invalid intent type %s", entity.Type)
	}
}

func mapSubscriptionFromDB(entity *db.Charge) *charges.SubscriptionReference {
	if entity.SubscriptionID == nil || entity.SubscriptionPhaseID == nil || entity.SubscriptionItemID == nil {
		return nil
	}

	return &charges.SubscriptionReference{
		SubscriptionID: *entity.SubscriptionID,
		PhaseID:        *entity.SubscriptionPhaseID,
		ItemID:         *entity.SubscriptionItemID,
	}
}

func mapStandardInvoiceRealizationFromDB(entity *db.ChargeStandardInvoiceRealization) charges.StandardInvoiceRealization {
	return charges.StandardInvoiceRealization{
		NamespacedID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt.In(time.UTC),
			UpdatedAt: entity.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(entity.DeletedAt, time.UTC),
		},

		Annotations: entity.Annotations,
		LineID:      entity.LineID,
		ServicePeriod: timeutil.ClosedPeriod{
			From: entity.ServicePeriodFrom.In(time.UTC),
			To:   entity.ServicePeriodTo.In(time.UTC),
		},
		Status:                          entity.Status,
		MeteredServicePeriodQuantity:    entity.MeteredServicePeriodQuantity,
		MeteredPreServicePeriodQuantity: entity.MeteredPreServicePeriodQuantity,
		Totals: billing.Totals{
			Amount:              entity.Amount,
			TaxesTotal:          entity.TaxesTotal,
			TaxesInclusiveTotal: entity.TaxesInclusiveTotal,
			TaxesExclusiveTotal: entity.TaxesExclusiveTotal,
			ChargesTotal:        entity.ChargesTotal,
			DiscountsTotal:      entity.DiscountsTotal,
			Total:               entity.Total,
		},
	}
}
