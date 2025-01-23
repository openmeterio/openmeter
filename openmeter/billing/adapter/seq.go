package billingadapter

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingsequencenumbers"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.SequenceAdapter = (*adapter)(nil)

func (a *adapter) NextSequenceNumber(ctx context.Context, input billing.NextSequenceNumberInput) (alpacadecimal.Decimal, error) {
	if err := input.Validate(); err != nil {
		return alpacadecimal.Zero, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (alpacadecimal.Decimal, error) {
		existingRecord, err := tx.db.BillingSequenceNumbers.Query().
			Where(
				billingsequencenumbers.Namespace(input.Namespace),
				billingsequencenumbers.Scope(input.Scope),
			).
			ForUpdate().
			First(ctx)
		if err != nil {
			if !entdb.IsNotFound(err) {
				return alpacadecimal.Zero, err
			}

			err := tx.db.BillingSequenceNumbers.Create().
				SetNamespace(input.Namespace).
				SetScope(input.Scope).
				SetLast(alpacadecimal.NewFromInt(0)).
				OnConflict().
				DoNothing().
				Exec(ctx)
			if err != nil {
				return alpacadecimal.Zero, err
			}

			existingRecord, err = tx.db.BillingSequenceNumbers.Query().
				Where(
					billingsequencenumbers.Namespace(input.Namespace),
					billingsequencenumbers.Scope(input.Scope),
				).
				ForUpdate().
				First(ctx)
			if err != nil {
				return alpacadecimal.Zero, err
			}
		}

		next := existingRecord.Last.Add(alpacadecimal.NewFromInt(1))
		err = tx.db.BillingSequenceNumbers.Update().
			SetLast(next).
			Where(
				billingsequencenumbers.Namespace(input.Namespace),
				billingsequencenumbers.Scope(input.Scope),
			).Exec(ctx)
		if err != nil {
			return alpacadecimal.Zero, err
		}

		return next, nil
	})
}
