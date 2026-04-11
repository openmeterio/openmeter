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
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func MapChargeBaseFromDB(dbEntity *entdb.ChargeCreditPurchase) creditpurchase.ChargeBase {
	mappedMeta := chargemeta.MapFromDB(dbEntity)

	return creditpurchase.ChargeBase{
		ManagedResource: mappedMeta.ManagedResource,
		Status:          dbEntity.StatusDetailed,
		Intent: creditpurchase.Intent{
			Intent:       mappedMeta.Intent,
			CreditAmount: dbEntity.CreditAmount,
			EffectiveAt:  convert.SafeToUTC(dbEntity.EffectiveAt),
			Priority:     dbEntity.Priority,
			Settlement:   dbEntity.Settlement,
		},
	}
}

func MapCreditPurchaseChargeFromDB(dbEntity *entdb.ChargeCreditPurchase, expands meta.Expands) (creditpurchase.Charge, error) {
	chargeBase := MapChargeBaseFromDB(dbEntity)

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
