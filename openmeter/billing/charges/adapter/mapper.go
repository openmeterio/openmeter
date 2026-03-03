package adapter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MapChargeFromDB dispatches to the type-specific mapper based on the charge's type field.
func MapChargeFromDB(entity *entdb.Charge, expands charges.Expands) (charges.Charge, error) {
	switch entity.Type {
	case charges.ChargeTypeFlatFee:
		ff, err := MapFlatFeeChargeFromDB(entity, expands)
		if err != nil {
			return charges.Charge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
		}

		return ff.AsCharge(), nil
	case charges.ChargeTypeUsageBased:
		ub, err := MapUsageBasedChargeFromDB(entity, expands)
		if err != nil {
			return charges.Charge{}, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
		}

		return ub.AsCharge(), nil
	case charges.ChargeTypeCreditPurchase:
		cp, err := MapCreditPurchaseChargeFromDB(entity, expands)
		if err != nil {
			return charges.Charge{}, fmt.Errorf("mapping credit purchase charge [id=%s]: %w", entity.ID, err)
		}

		return cp.AsCharge(), nil
	default:
		return charges.Charge{}, fmt.Errorf("unknown charge type: %s", entity.Type)
	}
}

// MapFlatFeeChargeFromDB converts a DB Charge entity (with loaded FlatFee edge) to a FlatFeeCharge.
func MapFlatFeeChargeFromDB(entity *entdb.Charge, expands charges.Expands) (charges.FlatFeeCharge, error) {
	if entity.Edges.FlatFee == nil {
		return charges.FlatFeeCharge{}, fmt.Errorf("flat_fee edge not loaded for charge [id=%s]", entity.ID)
	}

	ff := entity.Edges.FlatFee

	var percentageDiscounts *productcatalog.PercentageDiscount
	if ff.Discounts != nil {
		percentageDiscounts = ff.Discounts.Percentage
	}

	charge := charges.FlatFeeCharge{
		ManagedResource: mapManagedResourceFromDB(entity),
		Status:          entity.Status,
		Intent: charges.FlatFeeIntent{
			IntentMeta:            mapIntentMetaFromDB(entity),
			InvoiceAt:             ff.InvoiceAt.UTC(),
			SettlementMode:        ff.SettlementMode,
			PaymentTerm:           ff.PaymentTerm,
			FeatureKey:            lo.FromPtrOr(ff.FeatureKey, ""),
			PercentageDiscounts:   percentageDiscounts,
			ProRating:             proRatingConfigFromDB(ff.ProRating),
			AmountBeforeProration: ff.AmountBeforeProration,
			AmountAfterProration:  ff.AmountAfterProration,
		},
	}

	if expands.Has(charges.ExpandRealizations) {
		dbCreditRealizations, err := ff.Edges.ChargeCreditRealizationsOrErr()
		if err != nil {
			return charges.FlatFeeCharge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
		}

		charge.State.CreditRealizations = lo.Map(dbCreditRealizations, func(entity *entdb.ChargeCreditRealization, _ int) charges.CreditRealization {
			return mapCreditRealizationFromDB(entity)
		})

		dbPaymentState, err := ff.Edges.ChargeStandardInvoicePaymentSettlementOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return charges.FlatFeeCharge{}, fmt.Errorf("payment state not loaded for flat fee charge [id=%s]", entity.ID)
		}

		if dbPaymentState != nil {
			charge.State.Payment = lo.ToPtr(mapStandardInvoicePaymentSettlementFromDB(dbPaymentState))
		}

		dbAccruedUsage, err := ff.Edges.ChargeStandardInvoiceAccruedUsageOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return charges.FlatFeeCharge{}, fmt.Errorf("accrued usage not loaded for flat fee charge [id=%s]", entity.ID)
		}

		if dbAccruedUsage != nil {
			charge.State.AccruedUsage = lo.ToPtr(mapStandardInvoiceAccruedUsageFromDB(dbAccruedUsage))
		}
	}

	return charge, nil
}

func mapLedgerTransactionGroupReferenceFromDB(entity *string) *charges.LedgerTransactionGroupReference {
	if entity == nil {
		return nil
	}

	return &charges.LedgerTransactionGroupReference{
		TransactionGroupID: *entity,
	}
}

// MapUsageBasedChargeFromDB converts a DB Charge entity (with loaded UsageBased edge) to a UsageBasedCharge.
func MapUsageBasedChargeFromDB(entity *entdb.Charge, expands charges.Expands) (charges.UsageBasedCharge, error) {
	if entity.Edges.UsageBased == nil {
		return charges.UsageBasedCharge{}, fmt.Errorf("usage_based edge not loaded for charge [id=%s]", entity.ID)
	}

	ub := entity.Edges.UsageBased

	if ub.Price == nil {
		return charges.UsageBasedCharge{}, fmt.Errorf("price is nil for usage based charge [id=%s]", entity.ID)
	}

	return charges.UsageBasedCharge{
		ManagedResource: mapManagedResourceFromDB(entity),
		Status:          entity.Status,
		Intent: charges.UsageBasedIntent{
			IntentMeta:     mapIntentMetaFromDB(entity),
			Price:          *ub.Price,
			FeatureKey:     ub.FeatureKey,
			InvoiceAt:      ub.InvoiceAt.UTC(),
			SettlementMode: ub.SettlementMode,
			Discounts:      ub.Discounts,
		},
		State: charges.UsageBasedState{},
	}, nil
}

// MapCreditPurchaseChargeFromDB converts a DB Charge entity (with loaded CreditPurchase edge) to a CreditPurchaseCharge.
func MapCreditPurchaseChargeFromDB(entity *entdb.Charge, expands charges.Expands) (charges.CreditPurchaseCharge, error) {
	if entity.Edges.CreditPurchase == nil {
		return charges.CreditPurchaseCharge{}, fmt.Errorf("credit_purchase edge not loaded for charge [id=%s]", entity.ID)
	}

	cp := entity.Edges.CreditPurchase

	var grantLedgerTransactionReference *charges.TimedLedgerTransactionGroupReference
	if cp.CreditGrantTransactionGroupID != nil {
		grantLedgerTransactionReference = &charges.TimedLedgerTransactionGroupReference{
			LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
				TransactionGroupID: *cp.CreditGrantTransactionGroupID,
			},
			Time: cp.CreditGrantedAt.In(time.UTC),
		}
	}

	return charges.CreditPurchaseCharge{
		ManagedResource: mapManagedResourceFromDB(entity),
		Status:          entity.Status,
		Intent: charges.CreditPurchaseIntent{
			IntentMeta:   mapIntentMetaFromDB(entity),
			CreditAmount: cp.CreditAmount,
			Settlement:   cp.Settlement,
		},
		State: charges.CreditPurchaseState{
			CreditGrantRealization: grantLedgerTransactionReference,
		},
	}, nil
}

// mapManagedResourceFromDB extracts the ManagedResource from a DB Charge entity.
func mapManagedResourceFromDB(entity *entdb.Charge) charges.ManagedResource {
	return charges.ManagedResource{
		NamespacedModel: models.NamespacedModel{
			Namespace: entity.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		ID: entity.ID,
	}
}

// mapIntentMetaFromDB extracts the IntentMeta from a DB Charge entity.
func mapIntentMetaFromDB(entity *entdb.Charge) charges.IntentMeta {
	return charges.IntentMeta{
		Name:        entity.Name,
		Metadata:    entity.Metadata,
		Annotations: entity.Annotations,
		ManagedBy:   entity.ManagedBy,
		CustomerID:  entity.CustomerID,
		Currency:    entity.Currency,
		ServicePeriod: timeutil.ClosedPeriod{
			From: entity.ServicePeriodFrom.UTC(),
			To:   entity.ServicePeriodTo.UTC(),
		},
		FullServicePeriod: timeutil.ClosedPeriod{
			From: entity.FullServicePeriodFrom.UTC(),
			To:   entity.FullServicePeriodTo.UTC(),
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: entity.BillingPeriodFrom.UTC(),
			To:   entity.BillingPeriodTo.UTC(),
		},
		UniqueReferenceID: entity.UniqueReferenceID,
		Subscription:      mapSubscriptionRefFromDB(entity),
	}
}

// mapSubscriptionRefFromDB extracts a SubscriptionReference from a DB Charge entity, returning nil if any ID is missing.
func mapSubscriptionRefFromDB(entity *entdb.Charge) *charges.SubscriptionReference {
	if entity.SubscriptionID == nil || entity.SubscriptionPhaseID == nil || entity.SubscriptionItemID == nil {
		return nil
	}

	return &charges.SubscriptionReference{
		SubscriptionID: *entity.SubscriptionID,
		PhaseID:        *entity.SubscriptionPhaseID,
		ItemID:         *entity.SubscriptionItemID,
	}
}

// proRatingConfigFromDB converts a DB ProRatingModeAdapterEnum to a ProRatingConfig.
func proRatingConfigFromDB(pr charges.ProRatingModeAdapterEnum) productcatalog.ProRatingConfig {
	switch pr {
	case charges.ProratePricesProratingAdapterMode:
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	default:
		return productcatalog.ProRatingConfig{
			Enabled: false,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}
}

// proRatingConfigToDB converts a ProRatingConfig to a DB ProRatingModeAdapterEnum.
func proRatingConfigToDB(pc productcatalog.ProRatingConfig) (charges.ProRatingModeAdapterEnum, error) {
	if !pc.Enabled {
		return charges.NoProratingAdapterMode, nil
	}

	if pc.Mode == productcatalog.ProRatingModeProratePrices {
		return charges.ProratePricesProratingAdapterMode, nil
	}

	return "", fmt.Errorf("invalid pro rating mode: %s", pc.Mode)
}

func mapCreditRealizationFromDB(entity *entdb.ChargeCreditRealization) charges.CreditRealization {
	return charges.CreditRealization{
		NamespacedID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt.In(time.UTC),
			UpdatedAt: entity.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(entity.DeletedAt, time.UTC),
		},
		CreditRealizationCreateInput: charges.CreditRealizationCreateInput{
			Annotations: entity.Annotations,
			ServicePeriod: timeutil.ClosedPeriod{
				From: entity.ServicePeriodFrom.In(time.UTC),
				To:   entity.ServicePeriodTo.In(time.UTC),
			},
			Amount: entity.Amount,
			LedgerTransaction: charges.LedgerTransactionGroupReference{
				TransactionGroupID: entity.LedgerTransactionGroupID,
			},
		},
		LineID: entity.LineID,
	}
}

func mapStandardInvoicePaymentSettlementFromDB(entity *entdb.ChargeStandardInvoicePaymentSettlement) charges.StandardInvoicePaymentSettlement {
	return charges.StandardInvoicePaymentSettlement{
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
		Status:     entity.Status,
		Amount:     entity.Amount,
		Authorized: mapTimedLedgerTransactionGroupReferenceFromDB(entity.AuthorizedTransactionGroupID, entity.AuthorizedAt),
		Settled:    mapTimedLedgerTransactionGroupReferenceFromDB(entity.SettledTransactionGroupID, entity.SettledAt),
	}
}

func mapTimedLedgerTransactionGroupReferenceFromDB(reference *string, at *time.Time) *charges.TimedLedgerTransactionGroupReference {
	if reference == nil || at == nil {
		return nil
	}

	return &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
			TransactionGroupID: *reference,
		},
		Time: at.In(time.UTC),
	}
}

func mapStandardInvoiceAccruedUsageFromDB(entity *entdb.ChargeStandardInvoiceAccruedUsage) charges.StandardInvoiceAccruedUsage {
	var ledgerTransaction *charges.LedgerTransactionGroupReference
	if entity.LedgerTransactionGroupID != nil {
		ledgerTransaction = &charges.LedgerTransactionGroupReference{
			TransactionGroupID: *entity.LedgerTransactionGroupID,
		}
	}

	return charges.StandardInvoiceAccruedUsage{
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
		Mutable:           entity.Mutable,
		LedgerTransaction: ledgerTransaction,
		Totals: billing.Totals{
			Amount:              entity.Amount,
			TaxesTotal:          entity.TaxesTotal,
			TaxesInclusiveTotal: entity.TaxesInclusiveTotal,
			TaxesExclusiveTotal: entity.TaxesExclusiveTotal,
			ChargesTotal:        entity.ChargesTotal,
			DiscountsTotal:      entity.DiscountsTotal,
			CreditsTotal:        entity.CreditsTotal,
			Total:               entity.Total,
		},
	}
}
