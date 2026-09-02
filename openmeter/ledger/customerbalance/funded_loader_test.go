package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestListCreditTransactionsFundedBalanceCoalescesSameEffectiveTime(t *testing.T) {
	tests := []struct {
		name          string
		advanceAmount int64
		wantBefore    int64
		wantAfter     int64
	}{
		{
			name:       "ordinary issuance",
			wantBefore: 0,
			wantAfter:  100,
		},
		{
			name:          "partially covered advance",
			advanceAmount: 40,
			wantBefore:    -40,
			wantAfter:     60,
		},
		{
			name:          "fully covered advance without issuance",
			advanceAmount: 100,
			wantBefore:    -100,
			wantAfter:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)

			// given:
			// - the customer may already have settled advance usage
			if tt.advanceAmount > 0 {
				servicePeriod := env.sp()
				charge := env.createFlatFeeCharge(
					t,
					alpacadecimal.NewFromInt(tt.advanceAmount),
					productcatalog.CreditOnlySettlementMode,
					servicePeriod,
				)
				env.passTimeAfterServicePeriod(t, servicePeriod)
				env.advanceFlatFeeCharge(t, charge)
			}

			fundedAt := clock.Now()
			env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, nil)

			// when:
			// - merged transaction history reconstructs the balance around the purchase
			result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         10,
				Currency:      &env.Currency,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)

			// then:
			// - the full purchase is visible and its before/after balance includes both
			//   issued FBO and cleared advance receivable value
			expectedItems := 1
			if tt.advanceAmount > 0 {
				expectedItems++
			}
			require.Len(t, result.Items, expectedItems)
			item := result.Items[0]
			require.Equal(t, CreditTransactionTypeFunded, item.Type)
			require.True(t, fundedAt.Equal(item.BookedAt))
			require.Equal(t, float64(100), item.Amount.InexactFloat64())
			require.Equal(t, float64(tt.wantBefore), item.Balance.Before.InexactFloat64())
			require.Equal(t, float64(tt.wantAfter), item.Balance.After.InexactFloat64())

			if tt.advanceAmount > 0 {
				advance := result.Items[1]
				require.Equal(t, CreditTransactionTypeConsumed, advance.Type)
				require.Equal(t, float64(-tt.advanceAmount), advance.Amount.InexactFloat64())
				require.Equal(t, float64(0), advance.Balance.Before.InexactFloat64())
				require.Equal(t, float64(-tt.advanceAmount), advance.Balance.After.InexactFloat64())
			}
		})
	}
}

func TestListCreditTransactionsFundedBalanceProjectsFeatureFilteredImpact(t *testing.T) {
	t.Run("partial projected impact", func(t *testing.T) {
		env := newTestEnv(t)

		// given:
		// - feature-restricted usage has created a 40-credit advance
		// - an unrestricted 100-credit purchase covers that advance and issues the remaining 60
		servicePeriod := env.sp()
		charge := env.createFlatFeeCharge(
			t,
			alpacadecimal.NewFromInt(40),
			productcatalog.CreditOnlySettlementMode,
			servicePeriod,
			testFeatureKey,
		)
		env.passTimeAfterServicePeriod(t, servicePeriod)
		env.advanceFlatFeeCharge(t, charge)
		env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, nil)

		// when:
		// - funded history is projected onto the unrestricted-only balance
		unrestricted, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
			CustomerID:    env.CustomerID,
			Limit:         10,
			Type:          lo.ToPtr(CreditTransactionTypeFunded),
			Currency:      &env.Currency,
			FeatureFilter: NewUnrestrictedFeatureFilter(),
		})
		require.NoError(t, err)

		// then:
		// - only the residual unrestricted issuance is visible; the restricted advance
		//   attribution does not affect this filtered balance
		require.Len(t, unrestricted.Items, 1)
		item := unrestricted.Items[0]
		require.Equal(t, float64(60), item.Amount.InexactFloat64())
		require.Equal(t, float64(0), item.Balance.Before.InexactFloat64())
		require.Equal(t, float64(60), item.Balance.After.InexactFloat64())

		// when:
		// - the same funded history is projected onto the advance's feature balance
		matchingFeature, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
			CustomerID:    env.CustomerID,
			Limit:         10,
			Type:          lo.ToPtr(CreditTransactionTypeFunded),
			Currency:      &env.Currency,
			FeatureFilter: NewFeatureFilter([]string{testFeatureKey}),
		})
		require.NoError(t, err)

		// then:
		// - both the restricted advance attribution and unrestricted issuance affect it
		require.Len(t, matchingFeature.Items, 1)
		item = matchingFeature.Items[0]
		require.Equal(t, float64(100), item.Amount.InexactFloat64())
		require.Equal(t, float64(-40), item.Balance.Before.InexactFloat64())
		require.Equal(t, float64(60), item.Balance.After.InexactFloat64())
	})

	t.Run("zero projected impact", func(t *testing.T) {
		env := newTestEnv(t)

		// given:
		// - an unrestricted purchase is fully attributed to a feature-restricted advance
		servicePeriod := env.sp()
		charge := env.createFlatFeeCharge(
			t,
			alpacadecimal.NewFromInt(40),
			productcatalog.CreditOnlySettlementMode,
			servicePeriod,
			testFeatureKey,
		)
		env.passTimeAfterServicePeriod(t, servicePeriod)
		env.advanceFlatFeeCharge(t, charge)
		env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(40), env.Currency, nil)

		// when:
		// - funded history is projected onto the unrestricted-only balance
		result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
			CustomerID:    env.CustomerID,
			Limit:         10,
			Type:          lo.ToPtr(CreditTransactionTypeFunded),
			Currency:      &env.Currency,
			FeatureFilter: NewUnrestrictedFeatureFilter(),
		})
		require.NoError(t, err)

		// then:
		// - the purchase has no impact on that balance and is omitted
		require.Empty(t, result.Items)
	})
}

func TestFundedCreditTransactionBalanceImpacts(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - a real funded group attributes 40 credits to a restricted advance now
	//   and issues the remaining 60 unrestricted credits later
	servicePeriod := env.sp()
	charge := env.createFlatFeeCharge(
		t,
		alpacadecimal.NewFromInt(40),
		productcatalog.CreditOnlySettlementMode,
		servicePeriod,
		testFeatureKey,
	)
	env.passTimeAfterServicePeriod(t, servicePeriod)
	env.advanceFlatFeeCharge(t, charge)
	fundedAt := clock.Now()
	effectiveAt := fundedAt.Add(time.Hour)
	purchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, &effectiveAt)
	require.NotNil(t, purchase.Realizations.CreditGrantRealization)

	group, err := env.Service.Ledger.GetTransactionGroup(t.Context(), models.NamespacedID{
		Namespace: env.Namespace,
		ID:        purchase.Realizations.CreditGrantRealization.TransactionGroupID,
	})
	require.NoError(t, err)

	type expectedImpact struct {
		bookedAt time.Time
		amount   int64
	}

	tests := []struct {
		name          string
		featureFilter creditpurchase.FeatureFilters
		unrestricted  bool
		want          []expectedImpact
	}{
		{
			name: "all balance routes",
			want: []expectedImpact{
				{bookedAt: effectiveAt, amount: 60},
				{bookedAt: fundedAt, amount: 40},
			},
		},
		{
			name:         "unrestricted balance routes",
			unrestricted: true,
			want: []expectedImpact{
				{bookedAt: effectiveAt, amount: 60},
			},
		},
		{
			name:          "matching feature balance routes",
			featureFilter: creditpurchase.FeatureFilters{testFeatureKey},
			want: []expectedImpact{
				{bookedAt: effectiveAt, amount: 60},
				{bookedAt: fundedAt, amount: 40},
			},
		},
		{
			name:          "non-matching feature still includes unrestricted routes",
			featureFilter: creditpurchase.FeatureFilters{"other-feature"},
			want: []expectedImpact{
				{bookedAt: effectiveAt, amount: 60},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := GetBalanceServiceInput{Currency: currencyx.Code("USD")}
			switch {
			case tt.unrestricted:
				input.FeatureFilter = NewUnrestrictedFeatureFilter()
			case len(tt.featureFilter) > 0:
				input.FeatureFilter = NewFeatureFilter(tt.featureFilter)
			}

			impacts, err := fundedCreditTransactionBalanceImpacts(group, input)
			require.NoError(t, err)
			require.Len(t, impacts, len(tt.want))
			for i, want := range tt.want {
				require.True(t, want.bookedAt.Equal(impacts[i].Cursor.BookedAt))
				require.Equal(t, float64(want.amount), impacts[i].Amount.InexactFloat64())
				require.True(t, env.CurrencyReference().Equal(impacts[i].CurrencyReference))
			}
		})
	}

	t.Run("restricted purchase has zero unrestricted impact", func(t *testing.T) {
		restrictedPurchase := env.createPromotionalCreditGrant(
			t,
			alpacadecimal.NewFromInt(25),
			env.Currency,
			nil,
			testFeatureKey,
		)
		require.NotNil(t, restrictedPurchase.Realizations.CreditGrantRealization)

		restrictedGroup, err := env.Service.Ledger.GetTransactionGroup(t.Context(), models.NamespacedID{
			Namespace: env.Namespace,
			ID:        restrictedPurchase.Realizations.CreditGrantRealization.TransactionGroupID,
		})
		require.NoError(t, err)

		impacts, err := fundedCreditTransactionBalanceImpacts(restrictedGroup, GetBalanceServiceInput{
			Currency:      env.Currency,
			FeatureFilter: NewUnrestrictedFeatureFilter(),
		})
		require.NoError(t, err)
		require.Empty(t, impacts)
	})
}

func TestListCreditTransactionsFundedBalanceUsesEffectiveTimes(t *testing.T) {
	// Each scenario issues one future-effective credit purchase, then lists its
	// funded history both before and after the purchase becomes effective.
	type expectedTransaction struct {
		bookedAfter time.Duration
		amount      int64
		before      int64
		after       int64
	}

	tests := []struct {
		name          string
		advanceAmount int64
		before        []expectedTransaction
		after         []expectedTransaction
	}{
		{
			name:   "ordinary future issuance",
			before: []expectedTransaction{},
			after: []expectedTransaction{
				{bookedAfter: time.Hour, amount: 100, before: 0, after: 100},
			},
		},
		{
			name:          "future issuance partially covers advance",
			advanceAmount: 40,
			before: []expectedTransaction{
				{amount: 40, before: -40, after: 0},
			},
			after: []expectedTransaction{
				{bookedAfter: time.Hour, amount: 60, before: 0, after: 60},
				{amount: 40, before: -40, after: 0},
			},
		},
		{
			name:          "future issuance fully covers advance",
			advanceAmount: 100,
			before: []expectedTransaction{
				{amount: 100, before: -100, after: 0},
			},
			after: []expectedTransaction{
				{amount: 100, before: -100, after: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)

			// given:
			// - a customer may have an outstanding advance when a future-effective
			//   credit purchase is funded
			if tt.advanceAmount > 0 {
				servicePeriod := env.sp()
				charge := env.createFlatFeeCharge(
					t,
					alpacadecimal.NewFromInt(tt.advanceAmount),
					productcatalog.CreditOnlySettlementMode,
					servicePeriod,
				)
				env.passTimeAfterServicePeriod(t, servicePeriod)
				env.advanceFlatFeeCharge(t, charge)
			}

			fundedAt := clock.Now()
			effectiveAt := fundedAt.Add(time.Hour)
			charge := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, &effectiveAt)

			// when:
			// - funded history is listed before the scheduled issuance is effective
			before, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         10,
				Type:          lo.ToPtr(CreditTransactionTypeFunded),
				Currency:      &env.Currency,
				AsOf:          &fundedAt,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)

			// then:
			// - only the advance attribution that already affected balance is visible
			require.Len(t, before.Items, len(tt.before))
			for i, want := range tt.before {
				item := before.Items[i]
				require.Equal(t, charge.ID, item.ID.ID)
				require.True(t, fundedAt.Add(want.bookedAfter).Equal(item.BookedAt))
				require.Equal(t, float64(want.amount), item.Amount.InexactFloat64())
				require.Equal(t, float64(want.before), item.Balance.Before.InexactFloat64())
				require.Equal(t, float64(want.after), item.Balance.After.InexactFloat64())
			}

			// when:
			// - funded history is listed after the scheduled issuance is effective
			afterEffectiveAt := effectiveAt.Add(time.Second)
			after, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         10,
				Type:          lo.ToPtr(CreditTransactionTypeFunded),
				Currency:      &env.Currency,
				AsOf:          &afterEffectiveAt,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)

			// then:
			// - each distinct effective time has its own correctly ordered balance line
			require.Len(t, after.Items, len(tt.after))
			for i, want := range tt.after {
				item := after.Items[i]
				require.Equal(t, charge.ID, item.ID.ID)
				require.True(t, fundedAt.Add(want.bookedAfter).Equal(item.BookedAt))
				require.Equal(t, float64(want.amount), item.Amount.InexactFloat64())
				require.Equal(t, float64(want.before), item.Balance.Before.InexactFloat64())
				require.Equal(t, float64(want.after), item.Balance.After.InexactFloat64())
			}
		})
	}
}

func TestListCreditTransactionsFundedEffectiveTimesAcrossCurrencies(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - USD has an outstanding advance and a future-effective purchase
	servicePeriod := env.sp()
	charge := env.createFlatFeeCharge(
		t,
		alpacadecimal.NewFromInt(40),
		productcatalog.CreditOnlySettlementMode,
		servicePeriod,
	)
	env.passTimeAfterServicePeriod(t, servicePeriod)
	env.advanceFlatFeeCharge(t, charge)

	fundedAt := clock.Now()
	effectiveAt := fundedAt.Add(2 * time.Hour)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, &effectiveAt)

	// and:
	// - EUR funding becomes effective between the two USD balance movements
	eurFundedAt := fundedAt.Add(time.Hour)
	clock.FreezeTime(eurFundedAt)
	t.Cleanup(clock.UnFreeze)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(200), "EUR", nil)

	// when:
	// - the unfiltered history is listed after the scheduled USD issuance
	afterEffectiveAt := effectiveAt.Add(time.Second)
	result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		AsOf:          &afterEffectiveAt,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	// then:
	// - the global order follows effective time while each currency keeps its own balance chain
	require.Len(t, result.Items, 4)
	require.Equal(t, currencyx.Code("USD"), result.Items[0].Currency)
	require.True(t, effectiveAt.Equal(result.Items[0].BookedAt))
	require.Equal(t, float64(60), result.Items[0].Amount.InexactFloat64())
	require.Equal(t, float64(0), result.Items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(60), result.Items[0].Balance.After.InexactFloat64())

	require.Equal(t, currencyx.Code("EUR"), result.Items[1].Currency)
	require.True(t, eurFundedAt.Equal(result.Items[1].BookedAt))
	require.Equal(t, float64(200), result.Items[1].Amount.InexactFloat64())
	require.Equal(t, float64(0), result.Items[1].Balance.Before.InexactFloat64())
	require.Equal(t, float64(200), result.Items[1].Balance.After.InexactFloat64())

	require.Equal(t, currencyx.Code("USD"), result.Items[2].Currency)
	require.True(t, fundedAt.Equal(result.Items[2].BookedAt))
	require.Equal(t, float64(40), result.Items[2].Amount.InexactFloat64())
	require.Equal(t, float64(-40), result.Items[2].Balance.Before.InexactFloat64())
	require.Equal(t, float64(0), result.Items[2].Balance.After.InexactFloat64())

	require.Equal(t, currencyx.Code("USD"), result.Items[3].Currency)
	require.Equal(t, CreditTransactionTypeConsumed, result.Items[3].Type)
	require.Equal(t, float64(-40), result.Items[3].Amount.InexactFloat64())
	require.Equal(t, float64(0), result.Items[3].Balance.Before.InexactFloat64())
	require.Equal(t, float64(-40), result.Items[3].Balance.After.InexactFloat64())
}
