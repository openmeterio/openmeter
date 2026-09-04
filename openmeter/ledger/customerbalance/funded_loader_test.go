package customerbalance

import (
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
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

func TestListCreditTransactionsFundedMarksVoidedGrant(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - one funded grant has been voided and another remains active
	voidedCharge := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(25), env.Currency, nil)
	activeCharge := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(40), env.Currency, nil)
	_, err := env.creditPurchase.MarkVoided(t.Context(), creditpurchase.MarkVoidedInput{
		ChargeID: voidedCharge.GetChargeID(),
		VoidedAt: clock.Now(),
	})
	require.NoError(t, err)

	// when:
	// - funded transaction history is listed after the void
	result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		Type:          lo.ToPtr(CreditTransactionTypeFunded),
		Currency:      &env.Currency,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	// then:
	// - only the funding backed by the voided grant carries the marker
	itemsByID := make(map[string]CreditTransaction, len(result.Items))
	for _, item := range result.Items {
		itemsByID[item.ID.ID] = item
	}

	require.Contains(t, itemsByID, voidedCharge.ID)
	require.True(t, itemsByID[voidedCharge.ID].GrantVoided)
	require.Contains(t, itemsByID, activeCharge.ID)
	require.False(t, itemsByID[activeCharge.ID].GrantVoided)
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
			input := GetBalanceServiceInput{Currency: currencies.NewCurrencyReference("USD")}
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
			Currency:      env.CurrencyReference(),
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

func TestListCreditTransactionsPaginatesFundedEffectiveTimes(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - a future-effective purchase covers an existing advance immediately and
	//   issues its remainder later
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
	splitPurchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, &effectiveAt)

	// and:
	// - another purchase is created and funded between those two balance movements,
	//   so charge order and final effective-time order differ
	middleFundedAt := fundedAt.Add(time.Hour)
	clock.FreezeTime(middleFundedAt)
	defer clock.UnFreeze()
	middlePurchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(25), env.Currency, nil)

	asOf := effectiveAt.Add(time.Second)
	txType := CreditTransactionTypeFunded

	expected := []struct {
		chargeID string
		bookedAt time.Time
		amount   int64
		before   int64
		after    int64
	}{
		{chargeID: splitPurchase.ID, bookedAt: effectiveAt, amount: 60, before: 25, after: 85},
		{chargeID: middlePurchase.ID, bookedAt: middleFundedAt, amount: 25, before: 0, after: 25},
		{chargeID: splitPurchase.ID, bookedAt: fundedAt, amount: 40, before: -40, after: 0},
	}

	for _, tc := range []struct {
		name          string
		limit         int
		wantPageSizes []int
	}{
		{name: "one row pages", limit: 1, wantPageSizes: []int{1, 1, 1}},
		{name: "multi row pages", limit: 2, wantPageSizes: []int{2, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         tc.limit,
				Type:          &txType,
				Currency:      &env.Currency,
				AsOf:          &asOf,
				FeatureFilter: AllFeatureFilter(),
			}

			// when:
			// - the funded history is traversed in both cursor directions
			forward := collectCreditTransactionsForward(t, env.Service, input, len(expected))
			reverse := collectCreditTransactionsBackward(t, env.Service, input, forward.lastPage, len(expected))

			// then:
			// - every effective-time line appears exactly once across page boundaries
			require.Equal(t, tc.wantPageSizes, forward.pageSizes)
			require.Len(t, forward.items, len(expected))
			require.Len(t, reverse, len(expected))
			for i, want := range expected {
				item := forward.items[i]
				require.Equal(t, want.chargeID, item.ID.ID)
				require.True(t, want.bookedAt.Equal(item.BookedAt), "booked at index %d: want %s, got %s", i, want.bookedAt, item.BookedAt)
				require.Equal(t, float64(want.amount), item.Amount.InexactFloat64())
				require.Equal(t, float64(want.before), item.Balance.Before.InexactFloat64())
				require.Equal(t, float64(want.after), item.Balance.After.InexactFloat64())

				reverseItem := reverse[len(expected)-1-i]
				require.Equal(t, want.chargeID, reverseItem.ID.ID)
				require.True(t, want.bookedAt.Equal(reverseItem.BookedAt))
				require.Equal(t, float64(want.amount), reverseItem.Amount.InexactFloat64())
			}
		})
	}

	// when:
	// - a request resumes directly between the split purchase's two impacts
	all, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		Type:          &txType,
		Currency:      &env.Currency,
		AsOf:          &asOf,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Len(t, all.Items, len(expected))
	middleCursor := creditTransactionCursor(all.Items[1])

	older, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		After:         &middleCursor,
		Type:          &txType,
		Currency:      &env.Currency,
		AsOf:          &asOf,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Len(t, older.Items, 1)
	require.Equal(t, splitPurchase.ID, older.Items[0].ID.ID)
	require.True(t, fundedAt.Equal(older.Items[0].BookedAt))

	newer, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		Before:        &middleCursor,
		Type:          &txType,
		Currency:      &env.Currency,
		AsOf:          &asOf,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Len(t, newer.Items, 1)
	require.Equal(t, splitPurchase.ID, newer.Items[0].ID.ID)
	require.True(t, effectiveAt.Equal(newer.Items[0].BookedAt))
}

func TestListCreditTransactionsPaginatesFeatureFilteredFundedEffectiveTimes(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - a future-effective unrestricted purchase covers a feature-restricted
	//   advance immediately, so that first impact is hidden by this projection
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
	effectiveAt := fundedAt.Add(2 * time.Hour)
	splitPurchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, &effectiveAt)

	// and:
	// - another unrestricted purchase is funded between the hidden and visible impacts
	middleFundedAt := fundedAt.Add(time.Hour)
	clock.FreezeTime(middleFundedAt)
	defer clock.UnFreeze()
	middlePurchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(25), env.Currency, nil)

	// when:
	// - unrestricted funded history is traversed one row at a time
	asOf := effectiveAt.Add(time.Second)
	txType := CreditTransactionTypeFunded
	input := ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         1,
		Type:          &txType,
		Currency:      &env.Currency,
		AsOf:          &asOf,
		FeatureFilter: NewUnrestrictedFeatureFilter(),
	}
	forward := collectCreditTransactionsForward(t, env.Service, input, 2)
	reverse := collectCreditTransactionsBackward(t, env.Service, input, forward.lastPage, 2)

	// then:
	// - pagination skips the filtered advance impact without losing either visible row
	require.Equal(t, []int{1, 1}, forward.pageSizes)
	require.Len(t, forward.items, 2)
	require.Len(t, reverse, 2)
	require.Equal(t, splitPurchase.ID, forward.items[0].ID.ID)
	require.True(t, effectiveAt.Equal(forward.items[0].BookedAt))
	require.Equal(t, float64(60), forward.items[0].Amount.InexactFloat64())
	require.Equal(t, middlePurchase.ID, forward.items[1].ID.ID)
	require.True(t, middleFundedAt.Equal(forward.items[1].BookedAt))
	require.Equal(t, float64(25), forward.items[1].Amount.InexactFloat64())
	require.Equal(t, []string{middlePurchase.ID, splitPurchase.ID}, []string{reverse[0].ID.ID, reverse[1].ID.ID})
}

func TestListCreditTransactionsPaginatesCoalescedFundedImpacts(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - an immediate purchase settles an advance and issues residual credit at
	//   the same effective time, producing two ledger transactions but one row
	servicePeriod := env.sp()
	charge := env.createFlatFeeCharge(
		t,
		alpacadecimal.NewFromInt(40),
		productcatalog.CreditOnlySettlementMode,
		servicePeriod,
	)
	env.passTimeAfterServicePeriod(t, servicePeriod)
	env.advanceFlatFeeCharge(t, charge)

	coalescedAt := clock.Now()
	coalescedPurchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, nil)

	// and:
	// - a later purchase creates a page boundary above the coalesced row
	laterAt := coalescedAt.Add(time.Hour)
	clock.FreezeTime(laterAt)
	defer clock.UnFreeze()
	laterPurchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(25), env.Currency, nil)

	// when:
	// - funded history is traversed one row at a time in both directions
	asOf := laterAt.Add(time.Second)
	txType := CreditTransactionTypeFunded
	input := ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         1,
		Type:          &txType,
		Currency:      &env.Currency,
		AsOf:          &asOf,
		FeatureFilter: AllFeatureFilter(),
	}
	forward := collectCreditTransactionsForward(t, env.Service, input, 2)
	reverse := collectCreditTransactionsBackward(t, env.Service, input, forward.lastPage, 2)

	// then:
	// - the same-time ledger transactions remain one 100-credit row on every page
	require.Equal(t, []int{1, 1}, forward.pageSizes)
	require.Len(t, forward.items, 2)
	require.Len(t, reverse, 2)
	require.Equal(t, laterPurchase.ID, forward.items[0].ID.ID)
	require.Equal(t, float64(25), forward.items[0].Amount.InexactFloat64())
	require.Equal(t, coalescedPurchase.ID, forward.items[1].ID.ID)
	require.True(t, coalescedAt.Equal(forward.items[1].BookedAt))
	require.Equal(t, float64(100), forward.items[1].Amount.InexactFloat64())
	require.Equal(t, []string{coalescedPurchase.ID, laterPurchase.ID}, []string{reverse[0].ID.ID, reverse[1].ID.ID})
}

func TestListCreditTransactionsPaginatesPastUnrelatedLedgerHistory(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - a funded purchase is followed by more unrelated customer-balance
	//   transactions than fit in one candidate batch
	fundedAt := clock.Now()
	purchase := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, nil)

	templates := make([]transactions.TransactionTemplate, chargeListPageSize+1)
	for i := range templates {
		templates[i] = transactions.IssueCustomerReceivableTemplate{
			At:       fundedAt.Add(time.Hour + time.Duration(i)*time.Microsecond),
			Amount:   alpacadecimal.NewFromInt(1),
			Currency: env.currencyReference(env.Currency),
		}
	}
	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		templates...,
	)
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	// when:
	// - funded history requests one row from before that unrelated history
	asOf := fundedAt.Add(2 * time.Hour)
	txType := CreditTransactionTypeFunded
	result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         1,
		Type:          &txType,
		Currency:      &env.Currency,
		AsOf:          &asOf,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	// then:
	// - the loader refills candidates and still returns the older funded row
	require.Len(t, result.Items, 1)
	require.Equal(t, purchase.ID, result.Items[0].ID.ID)
	require.True(t, fundedAt.Equal(result.Items[0].BookedAt))
	require.Nil(t, result.NextCursor)
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

type collectedCreditTransactionPages struct {
	items     []CreditTransaction
	pageSizes []int
	lastPage  ListCreditTransactionsResult
}

func collectCreditTransactionsForward(
	t *testing.T,
	service Service,
	input ListCreditTransactionsInput,
	maxItems int,
) collectedCreditTransactionPages {
	t.Helper()

	var collected collectedCreditTransactionPages
	for {
		page, err := service.ListCreditTransactions(t.Context(), input)
		require.NoError(t, err)
		require.NotEmpty(t, page.Items)

		collected.items = append(collected.items, page.Items...)
		collected.pageSizes = append(collected.pageSizes, len(page.Items))
		collected.lastPage = page
		require.LessOrEqual(t, len(collected.items), maxItems, "forward pagination did not terminate")

		if page.NextCursor == nil {
			return collected
		}

		input.After = page.NextCursor
		input.Before = nil
	}
}

func collectCreditTransactionsBackward(
	t *testing.T,
	service Service,
	input ListCreditTransactionsInput,
	lastForwardPage ListCreditTransactionsResult,
	maxItems int,
) []CreditTransaction {
	t.Helper()

	items := slices.Clone(lastForwardPage.Items)
	slices.Reverse(items)
	before := lastForwardPage.PreviousCursor

	for before != nil {
		input.After = nil
		input.Before = before
		page, err := service.ListCreditTransactions(t.Context(), input)
		require.NoError(t, err)
		require.NotEmpty(t, page.Items)

		pageItems := slices.Clone(page.Items)
		slices.Reverse(pageItems)
		items = append(items, pageItems...)
		require.LessOrEqual(t, len(items), maxItems, "backward pagination did not terminate")
		before = page.PreviousCursor
	}

	return items
}
