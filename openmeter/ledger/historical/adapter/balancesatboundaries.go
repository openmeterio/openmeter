package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (r *repo) GetBalancesAtBoundaries(ctx context.Context, input ledger.GetBalancesAtBoundariesInput) ([]ledger.Balance, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]ledger.Balance, error) {
		query, args, err := (balancesAtBoundariesQuery{input: input}).SQL()
		if err != nil {
			return nil, err
		}
		rows, err := tx.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("query balance boundaries: %w", err)
		}
		defer rows.Close()

		balances := make([]ledger.Balance, len(input.Queries))
		for rows.Next() {
			var amount sql.NullString
			var idx int
			if err := rows.Scan(&amount, &idx); err != nil {
				return nil, fmt.Errorf("scan balance boundary: %w", err)
			}
			balance, err := decimalFromNullString(amount)
			if err != nil {
				return nil, fmt.Errorf("parse balance boundary %d: %w", idx, err)
			}
			balances[idx] = balance
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("read balance boundaries: %w", err)
		}
		return balances, nil
	})
}
