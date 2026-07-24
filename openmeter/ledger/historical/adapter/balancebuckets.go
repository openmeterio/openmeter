package adapter

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type balanceBucketRow struct {
	SubAccountID                   string
	SourceChargeID                 stdsql.NullString
	SpendChargeID                  stdsql.NullString
	SumAmount                      stdsql.NullString
	RouteID                        string
	AccountType                    string
	RoutingKeyVersion              string
	RoutingKey                     string
	Currency                       string
	ExchangeSourceCurrency         stdsql.NullString
	TaxCode                        stdsql.NullString
	TaxBehavior                    stdsql.NullString
	Features                       pq.StringArray
	CostBasis                      stdsql.NullString
	CreditPriority                 stdsql.NullInt64
	TransactionAuthorizationStatus stdsql.NullString
}

func (r *repo) GetBalanceBuckets(ctx context.Context, query ledger.BalanceBucketQuery) ([]ledger.BalanceBucket, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]ledger.BalanceBucket, error) {
		sqlQuery, args, err := balanceBucketsQuery{query: query}.SQL()
		if err != nil {
			return nil, err
		}

		rows, err := tx.db.QueryContext(ctx, sqlQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to query ledger balance buckets: %w", err)
		}
		defer rows.Close()

		buckets := make([]ledger.BalanceBucket, 0)
		for rows.Next() {
			row := balanceBucketRow{}
			if err := rows.Scan(row.destinations()...); err != nil {
				return nil, fmt.Errorf("failed to scan ledger balance bucket: %w", err)
			}

			bucket, err := row.toBalanceBucket(query.GroupBy)
			if err != nil {
				return nil, err
			}
			buckets = append(buckets, bucket)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("ledger balance bucket rows: %w", err)
		}

		return buckets, nil
	})
}

func (r *balanceBucketRow) destinations() []any {
	return []any{
		&r.SubAccountID,
		&r.SourceChargeID,
		&r.SpendChargeID,
		&r.SumAmount,
		&r.RouteID,
		&r.AccountType,
		&r.RoutingKeyVersion,
		&r.RoutingKey,
		&r.Currency,
		&r.ExchangeSourceCurrency,
		&r.TaxCode,
		&r.TaxBehavior,
		&r.Features,
		&r.CostBasis,
		&r.CreditPriority,
		&r.TransactionAuthorizationStatus,
	}
}

func (r balanceBucketRow) toBalanceBucket(groupBy []string) (ledger.BalanceBucket, error) {
	amount, err := decimalFromNullString(r.SumAmount)
	if err != nil {
		return ledger.BalanceBucket{}, fmt.Errorf("failed to parse balance bucket amount for sub-account %s: %w", r.SubAccountID, err)
	}

	routingKey, err := ledger.NewRoutingKey(ledger.RoutingKeyVersion(r.RoutingKeyVersion), r.RoutingKey)
	if err != nil {
		return ledger.BalanceBucket{}, fmt.Errorf("sub-account %s routing key: %w", r.SubAccountID, err)
	}

	costBasis, err := nullableDecimalValue(r.CostBasis)
	if err != nil {
		return ledger.BalanceBucket{}, fmt.Errorf("sub-account %s cost basis: %w", r.SubAccountID, err)
	}

	address, err := ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID: r.SubAccountID,
		AccountType:  ledger.AccountType(r.AccountType),
		Route: ledger.Route{
			Currency:                       currencyx.Code(r.Currency),
			ExchangeSourceCurrency:         nullableCurrencyCode(r.ExchangeSourceCurrency),
			TaxCode:                        nullableStringValue(r.TaxCode),
			TaxBehavior:                    nullableTaxBehavior(r.TaxBehavior),
			Features:                       []string(r.Features),
			CostBasis:                      costBasis,
			CreditPriority:                 nullableIntValue(r.CreditPriority),
			TransactionAuthorizationStatus: nullableTransactionAuthorizationStatus(r.TransactionAuthorizationStatus),
		},
		RouteID:    r.RouteID,
		RoutingKey: routingKey,
	})
	if err != nil {
		return ledger.BalanceBucket{}, fmt.Errorf("sub-account %s address: %w", r.SubAccountID, err)
	}

	return ledger.BalanceBucket{
		Address:       address,
		GroupByValues: balanceBucketGroupByValues(groupBy, r),
		SettledAmount: amount,
		PendingAmount: amount,
	}, nil
}

func balanceBucketGroupByValues(groupBy []string, row balanceBucketRow) map[string]*string {
	values := make(map[string]*string, len(groupBy))

	for _, dimension := range groupBy {
		switch dimension {
		case ledger.BalanceBucketGroupBySourceChargeID:
			values[dimension] = nullableStringValue(row.SourceChargeID)
		case ledger.BalanceBucketGroupBySpendChargeID:
			values[dimension] = nullableStringValue(row.SpendChargeID)
		}
	}

	return values
}

func nullableStringValue(value stdsql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return lo.ToPtr(value.String)
}

func nullableCurrencyCode(value stdsql.NullString) *currencyx.Code {
	if !value.Valid {
		return nil
	}

	return lo.ToPtr(currencyx.Code(value.String))
}

func nullableTaxBehavior(value stdsql.NullString) *ledger.TaxBehavior {
	if !value.Valid {
		return nil
	}

	return lo.ToPtr(ledger.TaxBehavior(value.String))
}

func nullableTransactionAuthorizationStatus(value stdsql.NullString) *ledger.TransactionAuthorizationStatus {
	if !value.Valid {
		return nil
	}

	return lo.ToPtr(ledger.TransactionAuthorizationStatus(value.String))
}

func nullableDecimalValue(value stdsql.NullString) (*alpacadecimal.Decimal, error) {
	if !value.Valid {
		return nil, nil
	}

	decimal, err := alpacadecimal.NewFromString(value.String)
	if err != nil {
		return nil, err
	}

	return &decimal, nil
}

func nullableIntValue(value stdsql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	return lo.ToPtr(int(value.Int64))
}

func decimalFromNullString(value stdsql.NullString) (alpacadecimal.Decimal, error) {
	if !value.Valid {
		return alpacadecimal.Zero, nil
	}

	return alpacadecimal.NewFromString(value.String)
}
