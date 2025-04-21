package adapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingledger"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingsubledger"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingsubledgertransaction"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

var _ ledger.LedgerAdapter = (*adapter)(nil)

func (a *adapter) WithLockedLedger(ctx context.Context, input ledger.WithLockedLedgerAdapterInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		dbLedger, err := tx.db.BillingLedger.Query().
			Where(
				billingledger.Namespace(input.Customer.Namespace),
				billingledger.CustomerID(input.Customer.ID),
				billingledger.Currency(input.Currency),
				billingledger.DeletedAtIsNil(),
			).
			First(ctx)
		if err != nil {
			if !entdb.IsNotFound(err) {
				return fmt.Errorf("failed to get ledger: %w", err)
			}

			// Create the ledger
			dbLedger, err = tx.db.BillingLedger.Create().
				SetNamespace(input.Customer.Namespace).
				SetCustomerID(input.Customer.ID).
				SetCurrency(input.Currency).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("failed to create ledger: %w", err)
			}
		}

		ledger := mapLedgerFromDB(dbLedger)

		return input.Callback(ctx, ledger)
	})
}

func (a *adapter) GetLedger(ctx context.Context, input ledger.LedgerRef) (ledger.Ledger, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (ledger.Ledger, error) {
		dbLedger, err := tx.db.BillingLedger.Query().
			Where(
				billingledger.Namespace(input.Customer.Namespace),
				billingledger.CustomerID(input.Customer.ID),
				billingledger.Currency(input.Currency),
				billingledger.DeletedAtIsNil(),
			).
			First(ctx)
		if err != nil {
			return ledger.Ledger{}, models.NewGenericNotFoundError(fmt.Errorf("ledger not found for customer %s and currency %s", input.Customer.ID, input.Currency))
		}

		return mapLedgerFromDB(dbLedger), nil
	})
}

type subledgerBalance struct {
	SubledgerID string                `json:"subledger_id"`
	Balance     alpacadecimal.Decimal `json:"sum"`
}

func (a *adapter) GetBalance(ctx context.Context, input ledger.LedgerID) (ledger.GetBalanceAdapterResult, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (ledger.GetBalanceAdapterResult, error) {
		subledgers, err := tx.db.BillingSubledger.Query().
			Where(
				billingsubledger.Namespace(input.Namespace),
				billingsubledger.DeletedAtIsNil(),
				billingsubledger.LedgerID(input.ID),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get subledgers: %w", err)
		}

		subledgerIDs := lo.Map(subledgers, func(subledger *entdb.BillingSubledger, _ int) string {
			return subledger.ID
		})

		var subledgerBalances []subledgerBalance

		err = tx.db.BillingSubledgerTransaction.Query().
			Where(
				billingsubledgertransaction.Namespace(input.Namespace),
				billingsubledgertransaction.SubledgerIDIn(subledgerIDs...),
			).
			GroupBy(billingsubledgertransaction.FieldSubledgerID).
			Aggregate(
				entdb.Sum(billingsubledgertransaction.FieldAmount),
			).
			Scan(ctx, &subledgerBalances)
		if err != nil {
			return nil, fmt.Errorf("failed to get subledger balances: %w", err)
		}

		balanceByID := lo.SliceToMap(subledgerBalances, func(item subledgerBalance) (string, alpacadecimal.Decimal) {
			return item.SubledgerID, item.Balance
		})

		return lo.Map(subledgers, func(subledger *entdb.BillingSubledger, _ int) ledger.SubledgerBalance {
			return ledger.SubledgerBalance{
				Subledger: mapSubledgerFromDB(subledger),
				Balance:   balanceByID[subledger.ID],
			}
		}), nil
	})
}

func mapLedgerFromDB(dbLedger *entdb.BillingLedger) ledger.Ledger {
	return ledger.Ledger{
		NamespacedModel: models.NamespacedModel{
			Namespace: dbLedger.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: dbLedger.CreatedAt,
			UpdatedAt: dbLedger.UpdatedAt,
			DeletedAt: dbLedger.DeletedAt,
		},
		ID:         dbLedger.ID,
		CustomerID: dbLedger.CustomerID,
		Currency:   dbLedger.Currency,
	}
}
