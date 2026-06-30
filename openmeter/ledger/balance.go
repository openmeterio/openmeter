package ledger

import (
	"context"
	"errors"
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

type BalanceQuerier interface {
	GetAccountBalance(ctx context.Context, account Account, route RouteFilter, query BalanceQuery) (Balance, error)
	GetSubAccountBalance(ctx context.Context, subAccount SubAccount, query BalanceQuery) (Balance, error)
	GetBalanceBuckets(ctx context.Context, query BalanceBucketQuery) ([]BalanceBucket, error)
}
