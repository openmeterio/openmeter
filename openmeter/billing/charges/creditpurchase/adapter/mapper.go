package adapter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func MapCreditPurchaseChargeFromDB(m meta.Charge, dbEntity *entdb.ChargeCreditPurchase, expands meta.Expands) (creditpurchase.Charge, error) {
	if err := m.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	if m.Type != meta.ChargeTypeCreditPurchase {
		return creditpurchase.Charge{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	var grantLedgerTransactionReference *ledgertransaction.TimedGroupReference
	if dbEntity.CreditGrantTransactionGroupID != nil && dbEntity.CreditGrantedAt != nil {
		grantLedgerTransactionReference = &ledgertransaction.TimedGroupReference{
			GroupReference: ledgertransaction.GroupReference{
				TransactionGroupID: *dbEntity.CreditGrantTransactionGroupID,
			},
			Time: dbEntity.CreditGrantedAt.In(time.UTC),
		}
	}

	var externalPaymentSettlement *payment.External
	if expands.Has(meta.ExpandRealizations) {
		dbExternalPaymentSettlement, err := dbEntity.Edges.ExternalPaymentOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("external payment settlement not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbExternalPaymentSettlement != nil {
			externalPaymentSettlement = lo.ToPtr(payment.MapExternalFromDB(dbExternalPaymentSettlement))
		}
	}

	return creditpurchase.Charge{
		ManagedResource: m.ManagedResource,
		Status:          m.Status,
		Intent: creditpurchase.Intent{
			Intent:       m.Intent,
			CreditAmount: dbEntity.CreditAmount,
			Settlement:   dbEntity.Settlement,
		},
		State: creditpurchase.State{
			CreditGrantRealization:    grantLedgerTransactionReference,
			ExternalPaymentSettlement: externalPaymentSettlement,
		},
	}, nil
}
