package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

func (a *adapter) GetChargeByID(ctx context.Context, input models.NamespacedID) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charge, error) {
		entity, err := tx.db.Charge.Query().
			Where(dbcharge.Namespace(input.Namespace)).
			Where(dbcharge.ID(input.ID)).
			WithStandardInvoiceRealizations().
			WithUsageBased().
			WithFlatFee().
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return charges.Charge{}, models.NewGenericNotFoundError(
					fmt.Errorf("charge not found [namespace=%s charge.id=%s]", input.Namespace, input.ID),
				)
			}

			return charges.Charge{}, err
		}

		return mapChargeFromDB(entity), nil
	})
}

func mapChargeFromDB(entity *db.Charge) charges.Charge {
	return charges.Charge{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			CreatedAt:   entity.CreatedAt,
			UpdatedAt:   entity.UpdatedAt,
			DeletedAt:   entity.DeletedAt,
			Namespace:   entity.Namespace,
			ID:          entity.ID,
			Name:        entity.Name,
			Description: entity.Description,
		}),
		Intent: mapIntentFromDB(entity),
		Realizations: charges.Realizations{
			StandardInvoice: lo.Map(entity.Edges.StandardInvoiceRealizations, func(realization *db.ChargeStandardInvoiceRealization, _ int) charges.StandardInvoiceRealization {
				return mapStandardInvoiceRealizationFromDB(realization)
			}),
		},
	}
}

func mapIntentFromDB(entity *db.Charge) charges.Intent {
	return charges.Intent{
		IntentMeta: charges.IntentMeta{
			Metadata:    entity.Metadata,
			Annotations: entity.Annotations,
			ManagedBy:   entity.ManagedBy,
			CustomerID:  entity.CustomerID,
			Currency:    entity.Currency,
			ServicePeriod: timeutil.ClosedPeriod{
				From: entity.ServicePeriodFrom,
				To:   entity.ServicePeriodTo,
			},
			FullServicePeriod: timeutil.ClosedPeriod{
				From: entity.FullServicePeriodFrom,
				To:   entity.FullServicePeriodTo,
			},
			BillingPeriod: timeutil.ClosedPeriod{
				From: entity.BillingPeriodFrom,
				To:   entity.BillingPeriodTo,
			},
			InvoiceAt:         entity.InvoiceAt,
			TaxConfig:         lo.EmptyableToPtr(entity.TaxConfig),
			UniqueReferenceID: entity.UniqueReferenceID,
			Subscription:      mapSubscriptionFromDB(entity),
		},
		IntentType: entity.Type,
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
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},

		Annotations: entity.Annotations,
		LineID:      entity.LineID,
		ServicePeriod: timeutil.ClosedPeriod{
			From: entity.ServicePeriodFrom,
			To:   entity.ServicePeriodTo,
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
