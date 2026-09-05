package adapter

import (
	"context"
	"slices"

	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerunoveragecreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// legacyNilSpendQuery checks original allocation groups, not a charge-wide
// history: separate runs of the same charge can predate spend provenance.
type legacyNilSpendQuery struct {
	namespace      string
	realizationIDs []string
}

func (q legacyNilSpendQuery) Run(ctx context.Context, db *entdb.Client) (map[string]bool, error) {
	if len(q.realizationIDs) == 0 {
		return nil, nil
	}
	type allocation struct {
		ID                       string `json:"id"`
		LedgerTransactionGroupID string `json:"ledger_transaction_group_id"`
	}
	var flatFee, overage, usage []allocation
	if err := db.ChargeFlatFeeRunCreditAllocations.Query().Where(
		chargeflatfeeruncreditallocations.Namespace(q.namespace), chargeflatfeeruncreditallocations.IDIn(q.realizationIDs...),
	).Select(chargeflatfeeruncreditallocations.FieldID, chargeflatfeeruncreditallocations.FieldLedgerTransactionGroupID).Scan(ctx, &flatFee); err != nil {
		return nil, err
	}
	if err := db.ChargeFlatFeeRunOverageCreditAllocations.Query().Where(
		chargeflatfeerunoveragecreditallocations.Namespace(q.namespace), chargeflatfeerunoveragecreditallocations.IDIn(q.realizationIDs...),
	).Select(chargeflatfeerunoveragecreditallocations.FieldID, chargeflatfeerunoveragecreditallocations.FieldLedgerTransactionGroupID).Scan(ctx, &overage); err != nil {
		return nil, err
	}
	if err := db.ChargeUsageBasedRunCreditAllocations.Query().Where(
		chargeusagebasedruncreditallocations.Namespace(q.namespace), chargeusagebasedruncreditallocations.IDIn(q.realizationIDs...),
	).Select(chargeusagebasedruncreditallocations.FieldID, chargeusagebasedruncreditallocations.FieldLedgerTransactionGroupID).Scan(ctx, &usage); err != nil {
		return nil, err
	}
	allocations := slices.Concat(flatFee, overage, usage)
	groupIDs := lo.Map(allocations, func(a allocation, _ int) string { return a.LedgerTransactionGroupID })
	out := make(map[string]bool)
	if len(groupIDs) == 0 {
		return out, nil
	}
	entries, err := db.LedgerEntry.Query().Where(
		ledgerentry.Namespace(q.namespace), ledgerentry.OriginIDIsNil(), ledgerentry.SpendChargeIDIsNil(),
		ledgerentry.HasTransactionWith(ledgertransaction.GroupIDIn(groupIDs...)),
	).WithTransaction().All(ctx)
	if err != nil {
		return nil, err
	}
	nilSpendGroups := make(map[string]bool)
	for _, entry := range entries {
		tx := entry.Edges.Transaction
		if tx.Annotations[ledger.AnnotationTransactionTemplateCode] == transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) &&
			tx.Annotations[ledger.AnnotationTransactionDirection] == string(ledger.TransactionDirectionForward) {
			nilSpendGroups[tx.GroupID] = true
		}
	}
	for _, allocation := range allocations {
		out[allocation.ID] = nilSpendGroups[allocation.LedgerTransactionGroupID]
	}
	return out, nil
}
