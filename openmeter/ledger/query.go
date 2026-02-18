package ledger

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Querier interface {
	// QueryEntries queries ledger entries for the given parameters.
	// QueryEntries(ctx context.Context, query Query) (pagination.Result[LedgerEntry], error)

	// SumEntries sums the resultset of a Query. Account is mandatory.
	SumEntries(ctx context.Context, query Query) (QuerySummedResult, error)
}

type Query struct {
	Namespace string

	Cursor *pagination.Cursor
	Limit  int

	Filters Filters
}

func (p Query) Validate() error {
	if p.Namespace == "" {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"namespace": p.Namespace,
		})
	}

	if p.Limit < 1 {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"limit": p.Limit,
		})
	}

	if p.Cursor != nil {
		if err := p.Cursor.Validate(); err != nil {
			return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
				"cursor": p.Cursor,
				"error":  err,
			})
		}
	}

	return nil
}

type Filters struct {
	// BookedAtPeriod is inclusive-exclusive... should it be? Maybe finally add period inclusivity params?
	BookedAtPeriod *timeutil.OpenPeriod
	TransactionID  *string
	Dimensions     QueryDimensions
}

type QuerySummedResult struct {
	SettledSum alpacadecimal.Decimal
	PendingSum alpacadecimal.Decimal
}
