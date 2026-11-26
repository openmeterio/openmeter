package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ ledger.JournalReader = (*repo)(nil)

func accountEntries(scope ledger.AccountEntryScope) []predicate.LedgerEntry {
	return []predicate.LedgerEntry{
		ledgerentry.Namespace(scope.Namespace),
		ledgerentry.HasSubAccountWith(
			ledgersubaccount.Namespace(scope.Namespace),
			ledgersubaccount.AccountIDIn(scope.AccountIDs...),
		),
	}
}

func (r *repo) CountAccountEntries(ctx context.Context, scope ledger.AccountEntryScope) (int, error) {
	if err := scope.Validate(); err != nil {
		return 0, err
	}

	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (int, error) {
		return tx.db.LedgerEntry.Query().Where(accountEntries(scope)...).Unique(false).Count(ctx)
	})
}

func (r *repo) GetAccountBookingCoverage(ctx context.Context, input ledger.AccountBookingCoverageInput) (ledger.AccountBookingCoverage, error) {
	if err := input.Validate(); err != nil {
		return ledger.AccountBookingCoverage{}, err
	}

	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (ledger.AccountBookingCoverage, error) {
		query, args := accountBookingCoverageQuery{input: input}.SQL()
		rows, err := tx.db.QueryContext(ctx, query, args...)
		if err != nil {
			return ledger.AccountBookingCoverage{}, fmt.Errorf("query booking coverage: %w", err)
		}
		defer rows.Close()

		result := ledger.AccountBookingCoverage{EffectiveEarlier: make([]int, len(input.Earlier))}
		var next sql.NullTime
		dest := []any{&result.EntryCount, &next, &result.EffectiveAtAsOf}
		for i := range result.EffectiveEarlier {
			dest = append(dest, &result.EffectiveEarlier[i])
		}

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return ledger.AccountBookingCoverage{}, err
			}

			return ledger.AccountBookingCoverage{}, sql.ErrNoRows
		}

		if err := rows.Scan(dest...); err != nil {
			return ledger.AccountBookingCoverage{}, err
		}

		if next.Valid {
			result.NextBooking = &next.Time
		}

		return result, rows.Err()
	})
}

func (r *repo) ListSubAccountsBookedAfter(ctx context.Context, input ledger.SubAccountsBookedAfterInput) ([]string, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]string, error) {
		return tx.db.LedgerEntry.Query().
			Where(accountEntries(input.AccountEntryScope)...).
			Where(ledgerentry.HasTransactionWith(ledgertransaction.BookedAtGT(input.After))).
			Unique(true).Limit(input.Limit).Select(ledgerentry.FieldSubAccountID).Strings(ctx)
	})
}
