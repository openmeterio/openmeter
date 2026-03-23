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

	Filters Filters
}

func (p Query) Validate() error {
	if p.Namespace == "" {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"reason":    "namespace_required",
			"namespace": p.Namespace,
		})
	}

	if p.Cursor != nil {
		if err := p.Cursor.Validate(); err != nil {
			return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
				"reason": "cursor_invalid",
				"cursor": p.Cursor,
				"error":  err,
			})
		}
	}

	if p.Filters.TransactionID != nil && *p.Filters.TransactionID == "" {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"reason":         "transaction_id_required",
			"transaction_id": *p.Filters.TransactionID,
		})
	}

	if p.Filters.AccountID != nil && *p.Filters.AccountID == "" {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"reason":     "account_id_required",
			"account_id": *p.Filters.AccountID,
		})
	}

	if p.Filters.BookedAtPeriod != nil {
		if err := p.Filters.BookedAtPeriod.Validate(); err != nil {
			return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
				"reason":           "booked_at_period_invalid",
				"booked_at_period": p.Filters.BookedAtPeriod,
				"error":            err,
			})
		}
	}

	if _, err := p.Filters.Route.Normalize(); err != nil {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"reason": "route_invalid",
			"route":  p.Filters.Route,
			"error":  err,
		})
	}

	return nil
}

type Filters struct {
	// BookedAtPeriod is inclusive-exclusive... should it be? Maybe finally add period inclusivity params?
	BookedAtPeriod *timeutil.OpenPeriod
	TransactionID  *string
	// AccountID narrows the query to a single account via its sub-accounts.
	AccountID *string
	Route     RouteFilter
}

type QuerySummedResult struct {
	SettledSum alpacadecimal.Decimal
	PendingSum alpacadecimal.Decimal
}
