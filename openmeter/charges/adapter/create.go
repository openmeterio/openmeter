package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type createChargeIntentInputWithID struct {
	charges.CreateChargeIntentInput
	ID string
}

func (a *adapter) CreateCharges(ctx context.Context, input charges.CreateChargeInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (charges.Charges, error) {
		chargeCreates := lo.Map(input.Intents, func(intent charges.CreateChargeIntentInput, _ int) *db.ChargeCreate {
			create := repo.db.Charge.Create().
				// ManagedModel
				SetNamespace(input.Customer.Namespace).
				SetName(intent.Name).
				SetNillableDescription(intent.Description).

				// IntentMeta
				SetMetadata(intent.Metadata).
				SetAnnotations(intent.Annotations).
				SetManagedBy(intent.ManagedBy).
				SetCustomerID(input.Customer.ID).
				SetCurrency(input.Currency).
				SetSettlementMode(intent.SettlementMode).
				SetStatus(charges.ChargeStatusActive).
				SetServicePeriodFrom(intent.ServicePeriod.From).
				SetServicePeriodTo(intent.ServicePeriod.To).
				SetBillingPeriodFrom(intent.BillingPeriod.From).
				SetBillingPeriodTo(intent.BillingPeriod.To).
				SetFullServicePeriodFrom(intent.FullServicePeriod.From).
				SetFullServicePeriodTo(intent.FullServicePeriod.To).
				SetInvoiceAt(intent.InvoiceAt).
				SetNillableTaxConfig(intent.TaxConfig).
				SetNillableUniqueReferenceID(intent.UniqueReferenceID).

				// IntentType
				SetType(intent.IntentType)

			if intent.Subscription != nil {
				create.SetSubscriptionID(intent.Subscription.SubscriptionID).
					SetSubscriptionPhaseID(intent.Subscription.PhaseID).
					SetSubscriptionItemID(intent.Subscription.ItemID)
			}

			return create
		})

		createdCharges, err := repo.db.Charge.CreateBulk(chargeCreates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		chargesWithIDs := lo.Map(createdCharges, func(charge *db.Charge, idx int) createChargeIntentInputWithID {
			return createChargeIntentInputWithID{
				CreateChargeIntentInput: input.Intents[idx],
				ID:                      charge.ID,
			}
		})

		flatFeeIntents := lo.Filter(chargesWithIDs, func(intent createChargeIntentInputWithID, _ int) bool {
			return intent.IntentType == charges.IntentTypeFlatFee
		})

		flatFeeCreates, err := slicesx.MapWithErr(flatFeeIntents, func(intent createChargeIntentInputWithID) (*db.ChargeFlatFeeCreate, error) {
			flatFee, err := intent.GetFlatFeeIntent()
			if err != nil {
				return nil, err
			}

			var discounts *productcatalog.Discounts
			if flatFee.PercentageDiscounts != nil {
				discounts = &productcatalog.Discounts{
					Percentage: flatFee.PercentageDiscounts,
				}
			}

			return repo.db.ChargeFlatFee.Create().
				SetChargeID(intent.ID).
				SetID(intent.ID).
				SetNamespace(input.Customer.Namespace).
				SetPaymentTerm(flatFee.PaymentTerm).
				SetNillableFeatureKey(lo.EmptyableToPtr(flatFee.FeatureKey)).
				SetDiscounts(discounts).
				SetProRating(mapProRatingToDB(flatFee.ProRating)).
				SetAmountBeforeProration(flatFee.AmountBeforeProration).
				SetAmountAfterProration(flatFee.AmountAfterProration), nil
		})
		if err != nil {
			return nil, err
		}

		flatFeeEntities, err := repo.db.ChargeFlatFee.CreateBulk(flatFeeCreates...).Save(ctx)
		if err != nil {
			return nil, err
		}
		flatFeeEntitiesByID := lo.SliceToMap(flatFeeEntities, func(entity *db.ChargeFlatFee) (string, *db.ChargeFlatFee) {
			return entity.ID, entity
		})

		usageBasedIntents := lo.Filter(chargesWithIDs, func(intent createChargeIntentInputWithID, _ int) bool {
			return intent.IntentType == charges.IntentTypeUsageBased
		})

		usageBasedCreates, err := slicesx.MapWithErr(usageBasedIntents, func(intent createChargeIntentInputWithID) (*db.ChargeUsageBasedCreate, error) {
			usageBased, err := intent.GetUsageBasedIntent()
			if err != nil {
				return nil, err
			}

			return repo.db.ChargeUsageBased.Create().
				SetChargeID(intent.ID).
				SetID(intent.ID).
				SetNamespace(input.Customer.Namespace).
				SetPrice(lo.ToPtr(usageBased.Price)).
				SetFeatureKey(usageBased.FeatureKey).
				SetDiscounts(usageBased.Discounts), nil
		})
		if err != nil {
			return nil, err
		}

		usageBasedEntities, err := repo.db.ChargeUsageBased.CreateBulk(usageBasedCreates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		usageBasedEntitiesByID := lo.SliceToMap(usageBasedEntities, func(entity *db.ChargeUsageBased) (string, *db.ChargeUsageBased) {
			return entity.ID, entity
		})

		createdCharges, err = slicesx.MapWithErr(createdCharges, func(charge *db.Charge) (*db.Charge, error) {
			switch charge.Type {
			case charges.IntentTypeFlatFee:
				createdFlatFee, ok := flatFeeEntitiesByID[charge.ID]
				if !ok {
					return nil, fmt.Errorf("flat fee entity not found for charge %s", charge.ID)
				}
				charge.Edges.FlatFee = createdFlatFee
			case charges.IntentTypeUsageBased:
				createdUsageBased, ok := usageBasedEntitiesByID[charge.ID]
				if !ok {
					return nil, fmt.Errorf("usage based entity not found for charge %s", charge.ID)
				}
				charge.Edges.UsageBased = createdUsageBased
			default:
				return nil, fmt.Errorf("unknown charge type %s", charge.Type)
			}

			return charge, nil
		})
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(createdCharges, func(charge *db.Charge) (charges.Charge, error) {
			return mapChargeFromDB(charge)
		})
	})
}
