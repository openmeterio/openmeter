package adapter

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// LoadLegacyBackfillAmounts keeps the compatibility state aligned with the
// actual legacy portion of a purchase. A mixed purchase must not transition
// legacy lineage for value posted against a new collection origin.
func (a *adapter) LoadLegacyBackfillAmounts(ctx context.Context, groupID models.NamespacedID) (map[string]alpacadecimal.Decimal, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (map[string]alpacadecimal.Decimal, error) {
		entries, err := tx.db.LedgerEntry.Query().Where(
			ledgerentry.Namespace(groupID.Namespace), ledgerentry.OriginIDIsNil(), ledgerentry.SourceChargeIDIsNil(),
			ledgerentry.HasTransactionWith(ledgertransaction.GroupID(groupID.ID)),
		).WithTransaction().All(ctx)
		if err != nil {
			return nil, err
		}
		out := make(map[string]alpacadecimal.Decimal)
		for _, entry := range entries {
			code, err := ledger.TransactionTemplateCodeFromAnnotations(entry.Edges.Transaction.Annotations)
			if err != nil {
				return nil, err
			}
			if code != transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}) || !entry.Amount.IsPositive() {
				continue
			}
			key := lo.FromPtr(entry.SpendChargeID)
			out[key] = out[key].Add(entry.Amount)
		}
		return out, nil
	})
}
