package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	dbchargestandardinvoicerealization "github.com/openmeterio/openmeter/openmeter/ent/db/chargestandardinvoicerealization"
	dbchargeusagebased "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) UpsertChargesByChildUniqueReferenceID(ctx context.Context, input charges.UpsertChargesByChildUniqueReferenceIDInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charges, error) {
		existingCharges, err := tx.db.Charge.Query().
			Select(dbcharge.FieldID, dbcharge.FieldUniqueReferenceID).
			Where(dbcharge.Namespace(input.Customer.Namespace)).
			Where(dbcharge.CustomerID(input.Customer.ID)).
			Where(dbcharge.UniqueReferenceIDIn(lo.Map(input.Charges, func(charge charges.Charge, _ int) string {
				return *charge.Intent.UniqueReferenceID
			})...)).
			Where(dbcharge.UniqueReferenceIDNotNil()).
			Where(dbcharge.DeletedAtIsNil()).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query existing charges: %w", err)
		}

		chargesByUniqueReferenceID := lo.SliceToMap(existingCharges, func(charge *db.Charge) (string, *db.Charge) {
			return *charge.UniqueReferenceID, charge
		})

		// Let's add the IDs to the intents
		chargesWithIDs := lo.Map(input.Charges, func(charge charges.Charge, _ int) charges.Charge {
			existingCharge, ok := chargesByUniqueReferenceID[*charge.Intent.UniqueReferenceID]
			if ok {
				charge.ID = existingCharge.ID
			} else {
				charge.ID = ulid.Make().String()
			}

			return charge
		})

		// Let's bulk insert the changes
		creates, err := slicesx.MapWithErr(chargesWithIDs, func(charge charges.Charge) (*db.ChargeCreate, error) {
			intent := charge.Intent

			create := tx.db.Charge.Create().
				// ManagedResource
				SetNamespace(charge.Namespace).
				SetNillableDeletedAt(charge.DeletedAt).
				SetName(charge.Name).
				SetID(charge.ID).

				// Other fields
				SetMetadata(intent.Metadata).
				SetAnnotations(intent.Annotations).
				SetManagedBy(intent.ManagedBy).
				SetCustomerID(intent.CustomerID).
				SetCurrency(intent.Currency).
				SetType(intent.IntentType).
				SetServicePeriodFrom(intent.ServicePeriod.From).
				SetServicePeriodTo(intent.ServicePeriod.To).
				SetFullServicePeriodFrom(intent.FullServicePeriod.From).
				SetFullServicePeriodTo(intent.FullServicePeriod.To).
				SetBillingPeriodFrom(intent.BillingPeriod.From).
				SetBillingPeriodTo(intent.BillingPeriod.To).
				SetInvoiceAt(intent.InvoiceAt).
				SetNillableTaxConfig(intent.TaxConfig).
				SetNillableUniqueReferenceID(intent.UniqueReferenceID)

			if intent.Subscription != nil {
				create = create.
					SetSubscriptionID(intent.Subscription.SubscriptionID).
					SetSubscriptionPhaseID(intent.Subscription.PhaseID).
					SetSubscriptionItemID(intent.Subscription.ItemID)
			}

			return create, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create charges: %w", err)
		}

		if len(creates) > 0 {
			err = tx.db.Charge.
				CreateBulk(creates...).
				OnConflict(
					sql.ConflictColumns(
						dbcharge.FieldNamespace,
						dbcharge.FieldCustomerID,
						dbcharge.FieldUniqueReferenceID,
					),
					sql.ConflictWhere(sql.And(
						sql.NotNull(dbcharge.FieldUniqueReferenceID),
						sql.IsNull(dbcharge.FieldDeletedAt),
					)),
					sql.ResolveWithNewValues(),
					sql.ResolveWith(func(u *sql.UpdateSet) {
						u.SetIgnore(dbcharge.FieldCreatedAt)
					})).
				UpdateAnnotations().
				UpdateMetadata().
				UpdateDeletedAt().
				UpdateTaxConfig().
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert charges: %w", err)
			}
		}
		// Let's bulk insert the flat fees
		flatFees := lo.Filter(chargesWithIDs, func(charge charges.Charge, _ int) bool {
			return charge.Intent.IntentType == charges.IntentTypeFlatFee
		})

		flatFeesCreates, err := slicesx.MapWithErr(flatFees, func(charge charges.Charge) (*db.ChargeFlatFeeCreate, error) {
			flatFeeIntent, err := charge.Intent.GetFlatFeeIntent()
			if err != nil {
				return nil, fmt.Errorf("failed to get flat fee intent: %w", err)
			}

			return tx.db.ChargeFlatFee.Create().
				SetNamespace(charge.Namespace).
				SetID(charge.ID).
				SetChargeID(charge.ID).
				SetPaymentTerm(flatFeeIntent.PaymentTerm).
				SetDiscounts(&productcatalog.Discounts{
					Percentage: flatFeeIntent.PercentageDiscounts,
				}).
				SetProRating(mapProRatingToDB(flatFeeIntent.ProRating)).
				SetNillableFeatureKey(lo.EmptyableToPtr(flatFeeIntent.FeatureKey)).
				SetAmountBeforeProration(flatFeeIntent.AmountBeforeProration).
				SetAmountAfterProration(flatFeeIntent.AmountAfterProration), nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create flat fees: %w", err)
		}

		if len(flatFeesCreates) > 0 {
			err = tx.db.ChargeFlatFee.
				CreateBulk(flatFeesCreates...).
				OnConflict(sql.ConflictColumns(dbchargeflatfee.FieldNamespace, dbchargeflatfee.FieldID),
					sql.ResolveWithNewValues(),
				).
				UpdateDiscounts().
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert flat fees: %w", err)
			}
		}

		// Let's bulk insert the usage baseds
		usageBased := lo.Filter(chargesWithIDs, func(charge charges.Charge, _ int) bool {
			return charge.Intent.IntentType == charges.IntentTypeUsageBased
		})

		usageBasedCreates, err := slicesx.MapWithErr(usageBased, func(charge charges.Charge) (*db.ChargeUsageBasedCreate, error) {
			usageBasedIntent, err := charge.Intent.GetUsageBasedIntent()
			if err != nil {
				return nil, fmt.Errorf("failed to get usage based intent: %w", err)
			}

			return tx.db.ChargeUsageBased.Create().
				SetNamespace(charge.Namespace).
				SetID(charge.ID).
				SetChargeID(charge.ID).
				SetPrice(&usageBasedIntent.Price).
				SetFeatureKey(usageBasedIntent.FeatureKey).
				SetDiscounts(usageBasedIntent.Discounts), nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create usage baseds: %w", err)
		}

		if len(usageBasedCreates) > 0 {
			err = tx.db.ChargeUsageBased.
				CreateBulk(usageBasedCreates...).
				OnConflict(sql.ConflictColumns(dbchargeusagebased.FieldNamespace, dbchargeusagebased.FieldID),
					sql.ResolveWithNewValues(),
				).
				UpdateDiscounts().
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert usage baseds: %w", err)
			}
		}

		// Let's bulk insert the standard invoice realizations
		upsertedCharges, err := tx.upsertStandardInvoiceRealizations(ctx, input.Customer.Namespace, chargesWithIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert standard invoice realizations: %w", err)
		}

		return upsertedCharges, nil
	})
}

func (a *adapter) upsertStandardInvoiceRealizations(ctx context.Context, namespace string, toUpsert charges.Charges) (charges.Charges, error) {
	existingRealizations, err := a.db.ChargeStandardInvoiceRealization.Query().
		Select(
			dbchargestandardinvoicerealization.FieldID,
			dbchargestandardinvoicerealization.FieldChargeID,
			dbchargestandardinvoicerealization.FieldLineID,
			dbchargestandardinvoicerealization.FieldDeletedAt,
		).
		Where(dbchargestandardinvoicerealization.Namespace(namespace)).
		Where(dbchargestandardinvoicerealization.ChargeIDIn(lo.Map(toUpsert, func(charge charges.Charge, _ int) string {
			return charge.ID
		})...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing standard invoice realizations: %w", err)
	}

	existingRealizationsByLineID := lo.SliceToMap(existingRealizations, func(realization *db.ChargeStandardInvoiceRealization) (string, *db.ChargeStandardInvoiceRealization) {
		return realization.LineID, realization
	})

	allLineIDs := []string{}
	for chargeIdx := range toUpsert {
		for idx, stdRealization := range toUpsert[chargeIdx].Realizations.StandardInvoice {
			existingRealization, ok := existingRealizationsByLineID[stdRealization.LineID]
			if ok {
				stdRealization.ID = existingRealization.ID
			} else {
				stdRealization.ID = ulid.Make().String()
			}

			allLineIDs = append(allLineIDs, stdRealization.LineID)

			toUpsert[chargeIdx].Realizations.StandardInvoice[idx] = stdRealization
		}
	}

	// Let's mark any realization deleted that is not in the new list
	err = a.db.ChargeStandardInvoiceRealization.Update().
		Where(dbchargestandardinvoicerealization.Namespace(namespace)).
		Where(dbchargestandardinvoicerealization.ChargeIDIn(lo.Map(toUpsert, func(charge charges.Charge, _ int) string {
			return charge.ID
		})...)).
		Where(dbchargestandardinvoicerealization.LineIDNotIn(allLineIDs...)).
		SetDeletedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing standard invoice realizations: %w", err)
	}

	// Let's bulk insert the new realizations
	creates := []*db.ChargeStandardInvoiceRealizationCreate{}
	for _, charge := range toUpsert {
		for _, stdRealization := range charge.Realizations.StandardInvoice {
			create := a.db.ChargeStandardInvoiceRealization.Create().
				SetNamespace(namespace).
				SetID(stdRealization.ID).
				SetChargeID(charge.ID).
				SetLineID(stdRealization.LineID).
				SetBillingInvoiceLineID(stdRealization.LineID).
				SetServicePeriodFrom(stdRealization.ServicePeriod.From).
				SetServicePeriodTo(stdRealization.ServicePeriod.To).
				SetStatus(stdRealization.Status).
				SetMeteredServicePeriodQuantity(stdRealization.MeteredServicePeriodQuantity).
				SetMeteredPreServicePeriodQuantity(stdRealization.MeteredPreServicePeriodQuantity).
				SetAmount(stdRealization.Totals.Amount).
				SetTaxesTotal(stdRealization.Totals.TaxesTotal).
				SetTaxesInclusiveTotal(stdRealization.Totals.TaxesInclusiveTotal).
				SetTaxesExclusiveTotal(stdRealization.Totals.TaxesExclusiveTotal).
				SetChargesTotal(stdRealization.Totals.ChargesTotal).
				SetDiscountsTotal(stdRealization.Totals.DiscountsTotal).
				SetTotal(stdRealization.Totals.Total).
				SetAnnotations(stdRealization.Annotations)

			creates = append(creates, create)
		}
	}

	if len(creates) > 0 {
		err = a.db.ChargeStandardInvoiceRealization.
			CreateBulk(creates...).
			OnConflict(sql.ConflictColumns(dbchargestandardinvoicerealization.FieldNamespace,
				dbchargestandardinvoicerealization.FieldChargeID,
				dbchargestandardinvoicerealization.FieldLineID),
				sql.ResolveWithNewValues(),
			).
			UpdateAnnotations().
			UpdateDeletedAt().
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert standard invoice realizations: %w", err)
		}
	}

	return toUpsert, nil
}

func mapProRatingToDB(proRating productcatalog.ProRatingConfig) charges.ProRatingModeAdapterEnum {
	if proRating.Enabled && proRating.Mode == productcatalog.ProRatingModeProratePrices {
		return charges.ProRatingAdapterModeEnumProratePrices
	}

	return charges.ProRatingAdapterModeEnumNoProrate
}
