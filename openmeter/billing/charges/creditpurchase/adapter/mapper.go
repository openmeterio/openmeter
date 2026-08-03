package adapter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func fromDBBaseWithCurrency(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency) (creditpurchase.ChargeBase, error) {
	mappedMeta, err := chargemeta.FromDBWithCurrency(dbEntity, currency)
	if err != nil {
		return creditpurchase.ChargeBase{}, fmt.Errorf("failed to map charge base: %w", err)
	}

	return fromDBBase(dbEntity, mappedMeta), nil
}

func fromDBBase(dbEntity *entdb.ChargeCreditPurchase, mappedMeta meta.Charge) creditpurchase.ChargeBase {
	return creditpurchase.ChargeBase{
		ManagedResource: mappedMeta.ManagedResource,
		Status:          dbEntity.StatusDetailed,
		Intent: creditpurchase.Intent{
			Intent: mappedMeta.Intent,
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: mappedMeta.IntentMutableFields,
				CreditAmount:        dbEntity.CreditAmount,
				EffectiveAt:         convert.SafeToUTC(dbEntity.EffectiveAt),
				ExpiresAt:           convert.SafeToUTC(dbEntity.ExpiresAt),
				Priority:            dbEntity.Priority,
				FeatureFilters:      creditpurchase.FeatureFilters(dbEntity.FeatureFilters).Normalize(),
				Settlement:          dbEntity.Settlement,
			},
			Key: dbEntity.Key,
		},
		State: creditpurchase.State{
			VoidedAt: convert.SafeToUTC(dbEntity.VoidedAt),
		},
	}
}

func FromDB(dbEntity *entdb.ChargeCreditPurchase, expands meta.Expands) (creditpurchase.Charge, error) {
	mappedMeta, err := chargemeta.FromDB(dbEntity, dbEntity.Edges)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("failed to map charge meta: %w", err)
	}

	return fromDBWithMeta(dbEntity, mappedMeta, expands)
}

func FromDBWithCurrency(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency, expands meta.Expands) (creditpurchase.Charge, error) {
	mappedMeta, err := chargemeta.FromDBWithCurrency(dbEntity, currency)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("failed to map charge meta: %w", err)
	}

	return fromDBWithMeta(dbEntity, mappedMeta, expands)
}

func fromDBWithMeta(dbEntity *entdb.ChargeCreditPurchase, mappedMeta meta.Charge, expands meta.Expands) (creditpurchase.Charge, error) {
	chargeBase := fromDBBase(dbEntity, mappedMeta)

	var creditGrantRealization *ledgertransaction.TimedGroupReference
	var externalPaymentSettlement *payment.External
	var invoiceSettlement *payment.Invoiced
	if expands.Has(meta.ExpandRealizations) {
		dbCreditGrant, err := dbEntity.Edges.CreditGrantOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("credit grant not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbCreditGrant != nil {
			creditGrantRealization = &ledgertransaction.TimedGroupReference{
				GroupReference: ledgertransaction.GroupReference{
					TransactionGroupID: dbCreditGrant.TransactionGroupID,
				},
				Time: dbCreditGrant.GrantedAt.In(time.UTC),
			}
		}

		dbExternalPaymentSettlement, err := dbEntity.Edges.ExternalPaymentOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("external payment settlement not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbExternalPaymentSettlement != nil {
			externalPaymentSettlement = lo.ToPtr(payment.MapExternalFromDB(dbExternalPaymentSettlement))
		}

		dbInvoicedPaymentSettlement, err := dbEntity.Edges.InvoicedPaymentOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("invoiced payment settlement not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbInvoicedPaymentSettlement != nil {
			invoiceSettlement = lo.ToPtr(payment.MapInvoicedFromDB(dbInvoicedPaymentSettlement))
		}
	}

	return creditpurchase.Charge{
		ChargeBase: chargeBase,
		Realizations: creditpurchase.Realizations{
			CreditGrantRealization:    creditGrantRealization,
			ExternalPaymentSettlement: externalPaymentSettlement,
			InvoiceSettlement:         invoiceSettlement,
		},
	}, nil
}
