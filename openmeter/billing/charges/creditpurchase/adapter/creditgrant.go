package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ creditpurchase.CreditGrantAdapter = (*adapter)(nil)

func (a *adapter) CreateCreditGrant(ctx context.Context, chargeID meta.ChargeID, input creditpurchase.CreateCreditGrantInput) (ledgertransaction.TimedGroupReference, error) {
	if err := chargeID.Validate(); err != nil {
		return ledgertransaction.TimedGroupReference{}, err
	}

	if err := input.Validate(); err != nil {
		return ledgertransaction.TimedGroupReference{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (ledgertransaction.TimedGroupReference, error) {
		entity, err := tx.db.ChargeCreditPurchaseCreditGrant.Create().
			SetNamespace(chargeID.Namespace).
			SetChargeID(chargeID.ID).
			SetTransactionGroupID(input.TransactionGroupID).
			SetGrantedAt(input.GrantedAt.In(time.UTC)).
			Save(ctx)
		if err != nil {
			return ledgertransaction.TimedGroupReference{}, fmt.Errorf("creating credit grant for charge [id=%s]: %w", chargeID.ID, err)
		}

		return ledgertransaction.TimedGroupReference{
			GroupReference: ledgertransaction.GroupReference{
				TransactionGroupID: entity.TransactionGroupID,
			},
			Time: entity.GrantedAt.In(time.UTC),
		}, nil
	})
}
