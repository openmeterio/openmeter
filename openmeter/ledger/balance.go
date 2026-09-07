package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	BalanceBucketGroupBySourceChargeID = "source_charge_id"
	BalanceBucketGroupBySpendChargeID  = "spend_charge_id"
)

type BalanceQuery struct {
	After *TransactionCursor
	AsOf  *time.Time
}

type BalanceBucketQuery struct {
	Namespace string
	Filters   Filters
	GroupBy   []string

	// ExcludeAnnotationFilters excludes entries whose transaction annotations
	// contain any given key-value pair.
	ExcludeAnnotationFilters map[string]string
}

func (q BalanceBucketQuery) Validate() error {
	if err := (Query{
		Namespace: q.Namespace,
		Filters:   q.Filters,
	}).Validate(); err != nil {
		return err
	}

	for _, groupBy := range q.GroupBy {
		switch groupBy {
		case BalanceBucketGroupBySourceChargeID, BalanceBucketGroupBySpendChargeID:
		default:
			return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
				"reason":   "group_by_invalid",
				"group_by": groupBy,
				"error":    errors.New("unsupported balance bucket group by dimension"),
			})
		}
	}

	if len(lo.Uniq(q.GroupBy)) != len(q.GroupBy) {
		return ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"reason": "group_by_duplicate",
			"error":  errors.New("duplicate balance bucket group by dimension"),
		})
	}

	return nil
}

type BalanceBucket struct {
	Address       PostingAddress
	GroupByValues map[string]*string
	SettledAmount alpacadecimal.Decimal
	PendingAmount alpacadecimal.Decimal
}

// GetBalancesAtBoundariesInput describes independent persisted balance boundaries.
// Results follow Queries order, including zero balances for empty scopes.
type GetBalancesAtBoundariesInput struct {
	Queries []Query
}

func (i GetBalancesAtBoundariesInput) Validate() error {
	var errs []error
	if len(i.Queries) == 0 {
		errs = append(errs, errors.New("at least one balance boundary is required"))
	}
	for idx, query := range i.Queries {
		if err := query.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("boundary %d: %w", idx, err))
		}
		if query.Filters.After == nil && query.Filters.AsOf == nil {
			errs = append(errs, fmt.Errorf("boundary %d: cursor or asOf is required", idx))
		}
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type BalanceQuerier interface {
	GetBalancesAtBoundaries(ctx context.Context, input GetBalancesAtBoundariesInput) ([]Balance, error)
	GetAccountBalance(ctx context.Context, account Account, route RouteFilter, query BalanceQuery) (Balance, error)
	GetSubAccountBalance(ctx context.Context, subAccount SubAccount, query BalanceQuery) (Balance, error)
	GetBalanceBuckets(ctx context.Context, query BalanceBucketQuery) ([]BalanceBucket, error)
}
