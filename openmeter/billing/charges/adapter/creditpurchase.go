package adapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ charges.CreditPurchaseAdapter = (*adapter)(nil)

func (a *adapter) UpdateCreditPurchaseCharge(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	if err := charge.Validate(); err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.CreditPurchaseCharge, error) {
		intent := charge.Intent

		dbEntity, err := tx.updateChargeIntent(ctx, charge.GetChargeID(), intent.IntentMeta, charge.Status)
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}

		creditPurchaseUpdate := tx.db.ChargeCreditPurchase.UpdateOneID(charge.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(charge.Namespace)).
			SetCreditAmount(intent.CreditAmount).
			SetSettlement(intent.Settlement)

		if charge.State.CreditGrantRealization != nil {
			creditPurchaseUpdate = creditPurchaseUpdate.
				SetCreditGrantTransactionGroupID(charge.State.CreditGrantRealization.LedgerTransactionGroupReference.TransactionGroupID).
				SetCreditGrantedAt(charge.State.CreditGrantRealization.Time.In(time.UTC))
		}

		if charge.State.ExternalPaymentSettlement != nil {
			creditPurchaseUpdate = creditPurchaseUpdate.
				SetChargeExternalPaymentSettlementID(charge.State.ExternalPaymentSettlement.ID)
		} else {
			creditPurchaseUpdate = creditPurchaseUpdate.
				ClearExternalPaymentSettlementID()
		}

		dbCreditPurchase, err := creditPurchaseUpdate.Save(ctx)
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}

		dbEntity.Edges.CreditPurchase = dbCreditPurchase

		mapped, err := MapCreditPurchaseChargeFromDB(dbEntity, charges.ExpandNone)
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}

		mapped.State.ExternalPaymentSettlement = charge.State.ExternalPaymentSettlement

		return mapped, nil
	})
}
