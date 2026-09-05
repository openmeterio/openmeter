package customerbalance

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type resolveCreditTransactionBalancesInput struct {
	CustomerID    customer.CustomerID
	Accounts      customerBalanceAccounts
	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
	Items         []CreditTransaction
}

func (i resolveCreditTransactionBalancesInput) Validate() error {
	var errs []error
	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}
	if i.Accounts.FBO == "" || i.Accounts.Receivable == "" {
		errs = append(errs, errors.New("customer balance accounts are required"))
	}
	if err := ValidateFeatureFilter(i.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Resolve each row independently so its balance does not depend on type filters
// or page boundaries. Terminal projections use their booked-time boundary and
// sibling offset; other rows use their last contributing ledger transaction.
func (s *service) resolveCreditTransactionBalances(ctx context.Context, input resolveCreditTransactionBalancesInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	if len(input.Items) == 0 {
		return nil
	}

	queries := make([]ledger.Query, 0, 2*len(input.Items))
	for _, item := range input.Items {
		balanceInput := GetBalanceServiceInput{
			CustomerID:    input.CustomerID,
			Currency:      item.CurrencyReference(),
			FeatureFilter: input.FeatureFilter,
			BalanceQuery:  item.balanceQuery(),
		}
		for _, scope := range []struct {
			accountID string
			route     ledger.RouteFilter
		}{
			{accountID: input.Accounts.FBO, route: balanceInput.bookedRoute()},
			{accountID: input.Accounts.Receivable, route: balanceInput.advanceRoute()},
		} {
			queries = append(queries, ledger.Query{
				Namespace: input.CustomerID.Namespace,
				Filters: ledger.Filters{
					AccountID: lo.ToPtr(scope.accountID),
					Route:     scope.route,
					After:     balanceInput.BalanceQuery.After,
					AsOf:      balanceInput.BalanceQuery.AsOf,
				},
			})
		}
	}
	balances, err := s.BalanceQuerier.GetBalancesAtBoundaries(ctx, ledger.GetBalancesAtBoundariesInput{Queries: queries})
	if err != nil {
		return fmt.Errorf("get credit transaction balances: %w", err)
	}
	for idx := range input.Items {
		item := &input.Items[idx]
		item.Balance.After = balances[2*idx].Add(balances[2*idx+1]).Add(item.balanceOffset)
		item.Balance.Before = item.Balance.After.Sub(lo.FromPtrOr(item.balanceImpact, item.Amount))
	}
	return nil
}
