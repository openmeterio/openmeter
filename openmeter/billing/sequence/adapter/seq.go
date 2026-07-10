package adapter

import (
	"context"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/sequence"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingsequencenumbers"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) NextSequenceNumber(ctx context.Context, input sequence.NextSequenceNumberInput) (alpacadecimal.Decimal, error) {
	if err := input.Validate(); err != nil {
		return alpacadecimal.Zero, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (alpacadecimal.Decimal, error) {
		one := alpacadecimal.NewFromInt(1)

		// Equivalent PostgreSQL:
		// INSERT INTO billing_sequence_numbers (namespace, scope, last)
		// VALUES ($1, $2, 1)
		// ON CONFLICT (namespace, scope)
		// DO UPDATE SET last = billing_sequence_numbers.last + 1
		// RETURNING last;
		insert := entsql.Dialect(tx.db.GetConfig().Driver.Dialect()).
			Insert(billingsequencenumbers.Table).
			Columns(
				billingsequencenumbers.FieldNamespace,
				billingsequencenumbers.FieldScope,
				billingsequencenumbers.FieldLast,
			).
			Values(input.Namespace, input.Scope, one).
			OnConflict(
				entsql.ConflictColumns(
					billingsequencenumbers.FieldNamespace,
					billingsequencenumbers.FieldScope,
				),
				entsql.ResolveWith(func(update *entsql.UpdateSet) {
					update.Add(billingsequencenumbers.FieldLast, one)
				}),
			).
			Returning(billingsequencenumbers.FieldLast)

		query, args, err := insert.QueryErr()
		if err != nil {
			return alpacadecimal.Zero, fmt.Errorf("failed to build sequence allocation query: %w", err)
		}

		rows, err := tx.db.QueryContext(ctx, query, args...)
		if err != nil {
			return alpacadecimal.Zero, fmt.Errorf("failed to allocate sequence number: %w", err)
		}
		defer rows.Close()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return alpacadecimal.Zero, fmt.Errorf("failed to read allocated sequence number: %w", err)
			}

			return alpacadecimal.Zero, errors.New("sequence allocation returned no value")
		}

		var next alpacadecimal.Decimal
		if err := rows.Scan(&next); err != nil {
			return alpacadecimal.Zero, fmt.Errorf("failed to scan allocated sequence number: %w", err)
		}

		return next, nil
	})
}
