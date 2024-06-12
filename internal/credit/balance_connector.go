package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/streaming"
)

// Generic connector for balance related operations.
type BalanceConnector interface {
	GetBalanceOfOwner(owner NamespacedGrantOwner, at time.Time) (float64, error)
	GetBalanceHistoryOfOwner(owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error)
	ResetUsageForOwner(owner NamespacedGrantOwner) error
}

type GetOwnerQueryParamsFn func(owner NamespacedGrantOwner) (namespace string, defaultParams streaming.QueryParams, err error)

type BalanceHistoryParams struct {
	From time.Time
	To   time.Time
}

func NewBalanceConnector(gc GrantConnector, sc streaming.Connector, getOwnerQuery GetOwnerQueryParamsFn) BalanceConnector {
	return &balanceConnector{gc: gc, sc: sc, getOwnerQuery: getOwnerQuery}
}

type balanceConnector struct {
	gc            GrantConnector
	sc            streaming.Connector
	getOwnerQuery GetOwnerQueryParamsFn
}

var _ BalanceConnector = &balanceConnector{}

func (m *balanceConnector) GetBalanceOfOwner(owner NamespacedGrantOwner, at time.Time) (float64, error) {
	// get last valid grantbalances
	// get all relevant grants
	// run engine and calculate grantbalance
	// store new grantbalance (& history)
	// return balance
	return 0, nil
}

func (m *balanceConnector) GetBalanceHistoryOfOwner(owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error) {
	return GrantBurnDownHistory{}, nil
}

func (m *balanceConnector) ResetUsageForOwner(owner NamespacedGrantOwner) error {
	return nil
}

// returns owner specific QueryUsageFn
func (m *balanceConnector) getQueryUsageFn(ctx context.Context, owner NamespacedGrantOwner) (QueryUsageFn, error) {
	meterSlug, ownerParams, err := m.getOwnerQuery(owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get query params for owner %v: %w", owner, err)
	}
	return func(from, to time.Time) (float64, error) {
		// copy
		params := ownerParams
		params.From = &from
		params.To = &to
		rows, err := m.sc.QueryMeter(context.TODO(), owner.Namespace, meterSlug, &params)
		if err != nil {
			return 0.0, fmt.Errorf("failed to query meter %s: %w", meterSlug, err)
		}
		if len(rows) > 1 {
			return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
		}
		if len(rows) == 0 {
			return 0.0, nil
		}
		return rows[0].Value, nil
	}, nil
}
