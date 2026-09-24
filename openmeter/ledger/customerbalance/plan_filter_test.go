package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPlanFilteredBalancesAndFundedTransactions(t *testing.T) {
	env := newTestEnv(t)
	// given: shared credits, all four version comparisons, and mixed dimensions.
	grants := []struct {
		amount  int64
		filters ledger.CreditFilters
	}{
		{100, ledger.CreditFilters{}},
		{10, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro"}}}},
		{20, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(1)}}}}},
		{30, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{In: []int{2, 3}}}}}},
		{40, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Gte: lo.ToPtr(3)}}}}},
		{50, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Lte: lo.ToPtr(2)}}}}},
		{60, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "enterprise", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}}}},
		{70, ledger.CreditFilters{Features: []string{"storage"}, Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}}}},
		{80, ledger.CreditFilters{Features: []string{"api"}, Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}}}},
		{90, ledger.CreditFilters{Features: []string{"api"}}},
		{15, ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(1)}}, {Key: "enterprise", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}}}},
	}
	firstIssuedAt := env.Now()
	for _, grant := range grants {
		env.createCreditPurchase(t, alpacadecimal.NewFromInt(grant.amount), env.Currency, nil, grant.filters, creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}))
		env.createCreditPurchase(t, alpacadecimal.NewFromInt(grant.amount), env.Currency, nil, grant.filters, creditpurchase.NewInvoiceSettlement())
		clock.FreezeTime(env.Now().Add(time.Minute))
		defer clock.UnFreeze()
	}

	for _, tc := range []struct {
		name    string
		plan    mo.Option[*ledger.PlanFilter]
		feature mo.Option[creditpurchase.FeatureFilters]
		want    float64
	}{
		{name: "all", want: 565},
		{name: "no plan restriction", plan: mo.Some[*ledger.PlanFilter](nil), want: 190},
		{name: "all pro versions", plan: mo.Some(&ledger.PlanFilter{Key: "pro"}), want: 505},
		{name: "pro v1", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(1)}}), want: 285},
		{name: "pro v2", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}), want: 430},
		{name: "pro v3", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(3)}}), want: 270},
		{name: "future pro version", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(4)}}), want: 240},
		{name: "unknown plan", plan: mo.Some(&ledger.PlanFilter{Key: "unknown"}), want: 190},
		{name: "pro v2 and api", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}), feature: NewFeatureFilter([]string{"api"}), want: 360},
		{name: "pro v2 without feature restriction", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}), feature: NewUnrestrictedFeatureFilter(), want: 190},
		{name: "fully unrestricted", plan: mo.Some[*ledger.PlanFilter](nil), feature: NewUnrestrictedFeatureFilter(), want: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when: the same selection is applied to balances and paginated funding history.
			facade, err := NewFacade(env.Service)
			require.NoError(t, err)
			balances, err := facade.GetBalances(t.Context(), GetBalancesInput{CustomerID: env.CustomerID, PlanFilter: tc.plan, FeatureFilter: tc.feature})
			require.NoError(t, err)
			require.Len(t, balances, 1)
			require.Equal(t, tc.want, balances[0].Balance.Settled().InexactFloat64())
			require.Equal(t, tc.want, balances[0].Balance.Live().InexactFloat64())
			require.Equal(t, tc.want, balances[0].Balance.Pending().InexactFloat64())

			input := ListCreditTransactionsInput{CustomerID: env.CustomerID, Limit: 2, PlanFilter: tc.plan, FeatureFilter: tc.feature}
			var items []CreditTransaction
			for {
				page, err := env.Service.ListCreditTransactions(t.Context(), input)
				require.NoError(t, err)
				items = append(items, page.Items...)
				if page.NextCursor == nil {
					break
				}
				input.After = page.NextCursor
			}
			// then: pending grants never appear, and each page continues the same filtered balance.
			remaining := tc.want
			for _, item := range items {
				require.Equal(t, CreditTransactionTypeFunded, item.Type)
				require.Equal(t, remaining, item.Balance.After.InexactFloat64())
				remaining -= item.Amount.InexactFloat64()
				require.Equal(t, remaining, item.Balance.Before.InexactFloat64())
			}
			require.Zero(t, remaining)

			historical, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
				CustomerID: env.CustomerID, Currency: env.CurrencyReference(), PlanFilter: tc.plan, FeatureFilter: tc.feature,
				BalanceQuery: ledger.BalanceQuery{AsOf: &firstIssuedAt},
			})
			require.NoError(t, err)
			require.Equal(t, float64(100), historical.Settled().InexactFloat64())
			require.Zero(t, historical.Live().InexactFloat64())
		})
	}
}

func TestPlanFilteredExpiry(t *testing.T) {
	env := newTestEnv(t)
	issuedAt := env.Now()
	expiresAt := issuedAt.Add(time.Hour)
	// given: two plan-restricted grants expire together.
	for _, key := range []string{"pro", "enterprise"} {
		period := timeutil.ClosedPeriod{From: issuedAt, To: issuedAt}
		_, err := env.creditPurchase.Create(t.Context(), creditpurchase.CreateInput{
			Namespace: env.Namespace,
			Intent: creditpurchase.Intent{
				Intent: chargemeta.Intent{
					ManagedBy: billing.SystemManagedLine, CustomerID: env.CustomerID.ID,
					Currency:  currenciestestutils.NewFiatCurrency(t, env.Currency),
					TaxConfig: productcatalog.TaxCodeConfig{TaxCodeID: env.taxCodeID},
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: chargemeta.IntentMutableFields{Name: "Expiring credit", ServicePeriod: period, BillingPeriod: period, FullServicePeriod: period},
					CreditAmount:        alpacadecimal.NewFromInt(100), ExpiresAt: &expiresAt,
					Filters:    ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: key}}},
					Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				},
			},
		})
		require.NoError(t, err)
	}
	// when: expiry is queried by plan, the other plan is excluded before pagination.
	result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID: env.CustomerID, Limit: 1, AsOf: &expiresAt,
		Type:       lo.ToPtr(CreditTransactionTypeExpired),
		PlanFilter: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}),
	})
	require.NoError(t, err)
	// then: only this plan's credit expires and its filtered balance reaches zero.
	require.Len(t, result.Items, 1)
	require.Nil(t, result.NextCursor)
	require.Equal(t, float64(-100), result.Items[0].Amount.InexactFloat64())
	require.Equal(t, float64(100), result.Items[0].Balance.Before.InexactFloat64())
	require.Zero(t, result.Items[0].Balance.After.InexactFloat64())
}

func TestPlanFilteredLiveConsumptionAndVoid(t *testing.T) {
	env := newTestEnv(t)
	// given: two independently restricted grants and charges with snapshotted plans.
	period := timeutil.ClosedPeriod{From: env.Now(), To: env.Now().Add(time.Hour)}
	var created []flatfee.Charge
	var grants []creditpurchase.Charge
	for _, key := range []string{"pro", "enterprise"} {
		grants = append(grants, env.createCreditPurchase(t, alpacadecimal.NewFromInt(100), env.Currency, nil,
			ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: key, Version: &ledger.VersionFilter{Gte: lo.ToPtr(2)}}}},
			creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{})))
		charges, err := env.flatFeeService.Create(t.Context(), flatfee.CreateInput{
			Namespace: env.Namespace,
			Intents: []flatfee.Intent{{
				Intent: chargemeta.Intent{
					ManagedBy: billing.SystemManagedLine, CustomerID: env.CustomerID.ID,
					Currency:         currenciestestutils.NewFiatCurrency(t, env.Currency),
					TaxConfig:        productcatalog.TaxCodeConfig{TaxCodeID: env.taxCodeID},
					SubscriptionPlan: &chargemeta.SubscriptionPlan{Key: key, Version: 2},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: chargemeta.IntentMutableFields{Name: "Plan charge", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
					InvoiceAt:           env.Now(), PaymentTerm: productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(40),
				},
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			}},
		})
		require.NoError(t, err)
		created = append(created, charges[0].Charge)
	}

	selection := mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}})
	// when: a live balance is selected before either charge has been booked.
	balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(), PlanFilter: selection,
	})
	require.NoError(t, err)
	// then: the other plan's pending charge does not reduce this balance.
	require.Equal(t, float64(100), balance.Settled().InexactFloat64())
	require.Equal(t, float64(60), balance.Live().InexactFloat64())

	clock.FreezeTime(period.To.Add(time.Second))
	defer clock.UnFreeze()
	for _, charge := range created {
		env.advanceFlatFeeCharge(t, charge)
	}
	consumed, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID: env.CustomerID, Limit: 10, PlanFilter: selection, Type: lo.ToPtr(CreditTransactionTypeConsumed),
	})
	require.NoError(t, err)
	require.Len(t, consumed.Items, 1)
	require.Equal(t, float64(-40), consumed.Items[0].Amount.InexactFloat64())
	require.Equal(t, float64(100), consumed.Items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(60), consumed.Items[0].Balance.After.InexactFloat64())

	// when: unused credit from both plans is voided, only this plan's movement is listed.
	for _, grant := range grants {
		_, err := env.CreditVoidService.VoidCreditPurchase(t.Context(), creditvoid.VoidCreditPurchaseInput{
			CustomerID: env.CustomerID, ChargeID: grant.ID, Currency: env.Currency,
		})
		require.NoError(t, err)
	}
	voided, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID: env.CustomerID, Limit: 10, PlanFilter: selection, Type: lo.ToPtr(CreditTransactionTypeVoided),
	})
	require.NoError(t, err)
	require.Len(t, voided.Items, 1)
	require.Equal(t, float64(-60), voided.Items[0].Amount.InexactFloat64())
	require.Equal(t, float64(60), voided.Items[0].Balance.Before.InexactFloat64())
	require.Zero(t, voided.Items[0].Balance.After.InexactFloat64())
}

func TestPlanFilteredFundingAfterAdvance(t *testing.T) {
	env := newTestEnv(t)

	// given: a plan-attributed charge creates an advance before shared credit arrives.
	period := env.sp()
	charges, err := env.flatFeeService.Create(t.Context(), flatfee.CreateInput{
		Namespace: env.Namespace,
		Intents: []flatfee.Intent{{
			Intent: chargemeta.Intent{
				ManagedBy: billing.SystemManagedLine, CustomerID: env.CustomerID.ID,
				Currency:         currenciestestutils.NewFiatCurrency(t, env.Currency),
				TaxConfig:        productcatalog.TaxCodeConfig{TaxCodeID: env.taxCodeID},
				SubscriptionPlan: &chargemeta.SubscriptionPlan{Key: "pro", Version: 2},
			},
			IntentMutableFields: flatfee.IntentMutableFields{
				IntentMutableFields: chargemeta.IntentMutableFields{Name: "Plan charge", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
				InvoiceAt:           env.Now(), PaymentTerm: productcatalog.InAdvancePaymentTerm,
				AmountBeforeProration: alpacadecimal.NewFromInt(40),
			},
			SettlementMode: productcatalog.CreditOnlySettlementMode,
		}},
	})
	require.NoError(t, err)
	clock.FreezeTime(period.To.Add(time.Second))
	defer clock.UnFreeze()
	env.advanceFlatFeeCharge(t, charges[0].Charge)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, nil)

	for _, tc := range []struct {
		name           string
		plan           mo.Option[*ledger.PlanFilter]
		amount, before float64
	}{
		{name: "matching plan", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}), amount: 100, before: -40},
		{name: "other plan", plan: mo.Some(&ledger.PlanFilter{Key: "enterprise"}), amount: 60},
		{name: "other version", plan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(1)}}), amount: 60},
		{name: "no plan restriction", plan: mo.Some[*ledger.PlanFilter](nil), amount: 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when: funding is viewed through one plan selection.
			result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID: env.CustomerID, Limit: 10, Type: lo.ToPtr(CreditTransactionTypeFunded), PlanFilter: tc.plan,
			})
			require.NoError(t, err)

			// then: only the matching plan includes the cleared advance; shared issuance appears in every view.
			require.Len(t, result.Items, 1)
			require.Equal(t, tc.amount, result.Items[0].Amount.InexactFloat64())
			require.Equal(t, tc.before, result.Items[0].Balance.Before.InexactFloat64())
			require.Equal(t, float64(60), result.Items[0].Balance.After.InexactFloat64())
		})
	}
}
