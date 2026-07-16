package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetBalanceServiceInputValidate(t *testing.T) {
	valid := GetBalanceServiceInput{
		CustomerID: customer.CustomerID{
			Namespace: "ns",
			ID:        "customer-id",
		},
		Currency:      currencyx.Code("USD"),
		FeatureFilter: AllFeatureFilter(),
	}
	now := clock.Now()
	validCursor := ledger.TransactionCursor{
		BookedAt:  now,
		CreatedAt: now,
		ID: models.NamespacedID{
			Namespace: "ns",
			ID:        "transaction-id",
		},
	}

	tests := []struct {
		name    string
		input   GetBalanceServiceInput
		wantErr bool
	}{
		{
			name:  "valid",
			input: valid,
		},
		{
			name:    "missing customer",
			input:   GetBalanceServiceInput{Currency: currencyx.Code("USD")},
			wantErr: true,
		},
		{
			name: "invalid currency",
			input: GetBalanceServiceInput{
				CustomerID: valid.CustomerID,
				Currency:   currencyx.Code("INVALID|CURRENCY"),
			},
			wantErr: true,
		},
		{
			name: "custom currency",
			input: GetBalanceServiceInput{
				CustomerID: valid.CustomerID,
				Currency:   currencyx.Code("CREDITS"),
			},
			wantErr: true,
		},
		{
			name: "multiple feature filters",
			input: GetBalanceServiceInput{
				CustomerID:    valid.CustomerID,
				Currency:      valid.Currency,
				FeatureFilter: NewFeatureFilter([]string{"feature-a", "feature-b"}),
			},
			wantErr: true,
		},
		{
			name: "invalid after cursor",
			input: GetBalanceServiceInput{
				CustomerID:   valid.CustomerID,
				Currency:     valid.Currency,
				BalanceQuery: ledger.BalanceQuery{After: &ledger.TransactionCursor{}},
			},
			wantErr: true,
		},
		{
			name: "zero as of",
			input: GetBalanceServiceInput{
				CustomerID:   valid.CustomerID,
				Currency:     valid.Currency,
				BalanceQuery: ledger.BalanceQuery{AsOf: &time.Time{}},
			},
			wantErr: true,
		},
		{
			name: "after and as of both set",
			input: GetBalanceServiceInput{
				CustomerID: valid.CustomerID,
				Currency:   valid.Currency,
				BalanceQuery: ledger.BalanceQuery{
					After: &validCursor,
					AsOf:  &now,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetBalanceServiceInputRoutes(t *testing.T) {
	input := GetBalanceServiceInput{
		Currency:      currencyx.Code("USD"),
		FeatureFilter: NewFeatureFilter([]string{"feature-a"}),
	}

	bookedRoute := input.bookedRoute()
	require.Equal(t, currencyx.Code("USD"), bookedRoute.Currency)
	require.Equal(t, "feature-a", bookedRoute.MatchFeature)
	require.True(t, bookedRoute.CostBasis.IsAbsent())

	advanceRoute := input.advanceRoute()
	require.Equal(t, currencyx.Code("USD"), advanceRoute.Currency)
	require.Equal(t, "feature-a", advanceRoute.MatchFeature)
	require.True(t, advanceRoute.CostBasis.IsPresent())
	costBasis, _ := advanceRoute.CostBasis.Get()
	require.Nil(t, costBasis)

	unrestrictedRoute := GetBalanceServiceInput{
		Currency:      currencyx.Code("USD"),
		FeatureFilter: NewUnrestrictedFeatureFilter(),
	}.bookedRoute()
	require.True(t, unrestrictedRoute.Features.IsPresent())
	features, _ := unrestrictedRoute.Features.Get()
	require.Empty(t, features)
	require.Empty(t, unrestrictedRoute.MatchFeature)
}

func TestGetBalance(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, env *testEnv)
		wantSettled int64
		wantLive    int64
	}{
		{
			name: "flat fee credit only",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 100,
			wantLive:    70,
		},
		{
			name: "flat fee credit then invoice",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(20))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode, env.sp())
			},
			wantSettled: 20,
			wantLive:    0,
		},
		{
			name: "credit only charge with no starting credits creates advance",
			setup: func(t *testing.T, env *testEnv) {
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 0,
			wantLive:    -30,
		},
		{
			name: "credit only charge advance settles",
			setup: func(t *testing.T, env *testEnv) {
				ch := env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
				env.passTimeAfterServicePeriod(t, env.sp())
				env.advanceFlatFeeCharge(t, ch)
			},
			wantSettled: -30,
			wantLive:    -30,
		},
		{
			name: "usage based credit only",
			setup: func(t *testing.T, env *testEnv) {
				env.addUsage(30, clock.Now().Add(-30*time.Minute))
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 100,
			wantLive:    70,
		},
		{
			name: "usage based credit then invoice",
			setup: func(t *testing.T, env *testEnv) {
				env.addUsage(30, clock.Now().Add(-30*time.Minute))
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(20))
				env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditThenInvoiceSettlementMode, env.sp())
			},
			wantSettled: 20,
			wantLive:    0,
		},
		{
			name: "mixed modes are pessimistic",
			setup: func(t *testing.T, env *testEnv) {
				env.addUsage(150, clock.Now().Add(-30*time.Minute))
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(80), productcatalog.CreditThenInvoiceSettlementMode, env.sp())
				env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 100,
			wantLive:    -130,
		},
		{
			name: "future usage-based charges are excluded until service period starts",
			setup: func(t *testing.T, env *testEnv) {
				futureServicePeriod := timeutil.ClosedPeriod{
					From: clock.Now().Add(time.Hour),
					To:   clock.Now().Add(2 * time.Hour),
				}

				env.addUsage(30, clock.Now().Add(-30*time.Minute))
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, futureServicePeriod)
			},
			wantSettled: 100,
			wantLive:    100,
		},
		{
			name: "future in-advance flat fee impacts live balance at invoice_at",
			setup: func(t *testing.T, env *testEnv) {
				futureServicePeriod := timeutil.ClosedPeriod{
					From: clock.Now().Add(time.Hour),
					To:   clock.Now().Add(2 * time.Hour),
				}

				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode, futureServicePeriod)
			},
			wantSettled: 100,
			wantLive:    70,
		},
		{
			name: "usage is settled",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(70))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(70))

				charge := env.createFlatFeeCharge(t,
					alpacadecimal.NewFromInt(30),
					productcatalog.CreditOnlySettlementMode,
					env.sp(),
				)

				env.advanceFlatFeeCharge(t, charge)
			},
			wantSettled: 40,
			wantLive:    40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			tt.setup(t, env)

			balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
				CustomerID:    env.CustomerID,
				Currency:      env.Currency,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)
			require.Equal(t, float64(tt.wantSettled), balance.Settled().InexactFloat64(), "settled: %s", balance.Settled())
			require.Equal(t, float64(tt.wantLive), balance.Live().InexactFloat64(), "live: %s", balance.Live())
		})
	}
}

func TestGetBalanceForFlatFeeCreditOnlyInvoiceAtBeforeServiceStart(t *testing.T) {
	createAt := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	invoiceAt := createAt.Add(time.Hour)
	servicePeriod := timeutil.ClosedPeriod{
		From: createAt.Add(2 * time.Hour),
		To:   createAt.Add(3 * time.Hour),
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	env := newTestEnv(t)
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))

	requireBalance := func(t *testing.T, settled float64, live float64) {
		t.Helper()

		balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
			CustomerID:    env.CustomerID,
			Currency:      env.Currency,
			FeatureFilter: AllFeatureFilter(),
		})
		require.NoError(t, err)
		require.Equal(t, settled, balance.Settled().InexactFloat64(), "settled: %s", balance.Settled())
		require.Equal(t, live, balance.Live().InexactFloat64(), "live: %s", balance.Live())
	}

	// given:
	// - the customer has enough credit before the charge exists
	requireBalance(t, 100, 100)

	createdCharges, err := env.flatFeeService.Create(t.Context(), flatfee.CreateInput{
		Namespace:     env.Namespace,
		FeatureMeters: env.featureMeters,
		Intents: []flatfee.Intent{
			{
				Intent: chargemeta.Intent{
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: env.CustomerID.ID,
					Currency:   currenciestestutils.NewFiatCurrency(t, env.Currency),
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: env.taxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: chargemeta.IntentMutableFields{
						Name:              "Platform Fee",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             invoiceAt,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(30),
				},
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, createdCharges, 1)

	charge := createdCharges[0].Charge
	require.Equal(t, flatfee.StatusCreated, charge.Status)
	require.NotNil(t, charge.State.AdvanceAfter)
	require.True(t, invoiceAt.Equal(*charge.State.AdvanceAfter))

	// then:
	// - before invoice_at the charge exists but does not impact live balance
	requireBalance(t, 100, 100)

	clock.FreezeTime(invoiceAt)

	// then:
	// - once invoice_at is current, the created charge impacts live balance
	// - the future booked_at ledger allocation is not visible in settled balance yet
	requireBalance(t, 100, 70)

	advancedCharge, err := env.flatFeeService.AdvanceCharge(t.Context(), flatfee.AdvanceChargeInput{
		ChargeID: charge.GetChargeID(),
	})
	require.NoError(t, err)
	require.NotNil(t, advancedCharge)
	require.Equal(t, flatfee.StatusActive, advancedCharge.Status)
	require.NotNil(t, advancedCharge.State.AdvanceAfter)
	require.True(t, servicePeriod.From.Equal(*advancedCharge.State.AdvanceAfter))
	require.Nil(t, advancedCharge.Realizations.CurrentRun)

	// then:
	// - after advancement, the active charge still impacts live balance
	requireBalance(t, 100, 70)

	clock.FreezeTime(servicePeriod.From)
	advancedCharge, err = env.flatFeeService.AdvanceCharge(t.Context(), flatfee.AdvanceChargeInput{
		ChargeID: charge.GetChargeID(),
	})
	require.NoError(t, err)
	require.NotNil(t, advancedCharge)
	require.Equal(t, flatfee.StatusFinal, advancedCharge.Status)
	require.Nil(t, advancedCharge.State.AdvanceAfter)
	require.NotNil(t, advancedCharge.Realizations.CurrentRun)
	require.Len(t, advancedCharge.Realizations.CurrentRun.CreditRealizations, 1)
	require.Equal(t, float64(30), advancedCharge.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())

	// then:
	// - once booked_at is current, the ledger allocation carries the balance
	requireBalance(t, 70, 70)

	clock.FreezeTime(servicePeriod.To)
	requireBalance(t, 70, 70)
}

func TestImpactRealizedCreditsSkipsVoidedUsageBasedBillingHistory(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	impact, err := NewImpact(charges.NewCharge(usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			Intent: usagebased.Intent{
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			}.AsOverridableIntent(),
		},
		Realizations: usagebased.RealizationRuns{
			{
				RealizationRunBase: usagebased.RealizationRunBase{
					Type: usagebased.RealizationRunTypeFinalRealization,
				},
				CreditsAllocated: creditrealization.Realizations{
					{
						CreateInput: creditrealization.CreateInput{
							Amount: alpacadecimal.NewFromInt(7),
						},
					},
				},
			},
			{
				RealizationRunBase: usagebased.RealizationRunBase{
					Type: usagebased.RealizationRunTypeInvalidDueToUnsupportedCreditNote,
				},
				CreditsAllocated: creditrealization.Realizations{
					{
						CreateInput: creditrealization.CreateInput{
							Amount: alpacadecimal.NewFromInt(10),
						},
					},
				},
			},
			{
				RealizationRunBase: usagebased.RealizationRunBase{
					Type: usagebased.RealizationRunTypePartialInvoice,
					ManagedModel: models.ManagedModel{
						DeletedAt: &deletedAt,
					},
				},
				CreditsAllocated: creditrealization.Realizations{
					{
						CreateInput: creditrealization.CreateInput{
							Amount: alpacadecimal.NewFromInt(20),
						},
					},
				},
			},
		},
	}), alpacadecimal.NewFromInt(50))
	require.NoError(t, err)

	require.Equal(t, float64(7), impact.RealizedCredits().InexactFloat64())
}

func TestImpactRealizedCreditsSkipsVoidedFlatFeeBillingHistory(t *testing.T) {
	impact, err := NewImpact(charges.NewCharge(flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			Intent: flatfee.Intent{
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			}.AsOverridableIntent(),
		},
		Realizations: flatfee.Realizations{
			CurrentRun: &flatfee.RealizationRun{
				RealizationRunBase: flatfee.RealizationRunBase{
					Type: flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote,
				},
				CreditRealizations: creditrealization.Realizations{
					{
						CreateInput: creditrealization.CreateInput{
							Amount: alpacadecimal.NewFromInt(10),
						},
					},
				},
			},
		},
	}), alpacadecimal.NewFromInt(50))
	require.NoError(t, err)

	require.True(t, impact.RealizedCredits().Equal(alpacadecimal.Zero))
}

func TestGetBalanceWithDifferentCurrency(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.fundOpenReceivableInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.fundOpenReceivableInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp(), "USD")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(70), productcatalog.CreditOnlySettlementMode, env.sp(), "EUR")

	usdBalance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      currencyx.Code("USD"),
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Equal(t, float64(100), usdBalance.Settled().InexactFloat64())
	require.Equal(t, float64(70), usdBalance.Live().InexactFloat64())

	eurBalance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      currencyx.Code("EUR"),
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Equal(t, float64(200), eurBalance.Settled().InexactFloat64())
	require.Equal(t, float64(130), eurBalance.Live().InexactFloat64())
}

func TestGetBalanceFeatureFilter(t *testing.T) {
	env := newTestEnv(t)

	unrestricted := alpacadecimal.NewFromInt(100)
	featureA := alpacadecimal.NewFromInt(10)
	featureAOrB := alpacadecimal.NewFromInt(10)

	env.bookFBOBalanceWithFeatures(t, unrestricted, nil)
	env.fundOpenReceivableWithFeatures(t, unrestricted, nil)
	env.bookFBOBalanceWithFeatures(t, featureA, []string{"feature-a"})
	env.fundOpenReceivableWithFeatures(t, featureA, []string{"feature-a"})
	env.bookFBOBalanceWithFeatures(t, featureAOrB, []string{"feature-a", "feature-b"})
	env.fundOpenReceivableWithFeatures(t, featureAOrB, []string{"feature-a", "feature-b"})

	tests := []struct {
		name   string
		filter mo.Option[creditpurchase.FeatureFilters]
		want   float64
	}{
		{
			name:   "omitted filter returns total portfolio balance",
			filter: AllFeatureFilter(),
			want:   120,
		},
		{
			name:   "unrestricted filter returns routes without feature restrictions",
			filter: NewUnrestrictedFeatureFilter(),
			want:   100,
		},
		{
			name:   "feature A filter includes unrestricted and overlapping restricted routes",
			filter: NewFeatureFilter([]string{"feature-a"}),
			want:   120,
		},
		{
			name:   "feature B filter includes unrestricted and A-or-B route",
			filter: NewFeatureFilter([]string{"feature-b"}),
			want:   110,
		},
		{
			name:   "unknown feature filter includes unrestricted routes only",
			filter: NewFeatureFilter([]string{"feature-c"}),
			want:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
				CustomerID:    env.CustomerID,
				Currency:      env.Currency,
				FeatureFilter: tt.filter,
			})
			require.NoError(t, err)

			require.Equal(t, tt.want, balance.Settled().InexactFloat64())
			require.Equal(t, tt.want, balance.Live().InexactFloat64())
		})
	}

	_, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: NewFeatureFilter([]string{"feature-a", "feature-b"}),
	})
	require.Error(t, err)

	customerAccounts, err := env.Deps.ResolversService.GetCustomerAccounts(t.Context(), env.CustomerID)
	require.NoError(t, err)

	now := clock.Now()
	exactFeatureABalance, err := env.Deps.HistoricalLedger.GetAccountBalance(t.Context(), customerAccounts.FBOAccount, ledger.RouteFilter{
		Currency: env.Currency,
		Features: mo.Some([]string{"feature-a"}),
	}, ledger.BalanceQuery{AsOf: &now})
	require.NoError(t, err)
	require.Equal(t, float64(10), exactFeatureABalance.InexactFloat64())
}

func TestGetBalanceFeatureFilterPendingChargeImpacts(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceWithFeatures(t, alpacadecimal.NewFromInt(100), nil)
	env.fundOpenReceivableWithFeatures(t, alpacadecimal.NewFromInt(100), nil)
	env.bookFBOBalanceWithFeatures(t, alpacadecimal.NewFromInt(10), []string{testFeatureKey})
	env.fundOpenReceivableWithFeatures(t, alpacadecimal.NewFromInt(10), []string{testFeatureKey})
	env.bookFBOBalanceWithFeatures(t, alpacadecimal.NewFromInt(20), []string{"storage"})
	env.fundOpenReceivableWithFeatures(t, alpacadecimal.NewFromInt(20), []string{"storage"})

	env.addUsage(30, clock.Now().Add(-30*time.Minute))
	env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(5), productcatalog.CreditOnlySettlementMode, env.sp())
	env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(7), productcatalog.CreditOnlySettlementMode, env.sp(), testFeatureKey)
	env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, env.sp())

	tests := []struct {
		name        string
		filter      mo.Option[creditpurchase.FeatureFilters]
		wantSettled float64
		wantLive    float64
	}{
		{
			name:        "omitted filter includes every charge impact",
			filter:      AllFeatureFilter(),
			wantSettled: 130,
			wantLive:    88,
		},
		{
			name:        "unrestricted filter includes only unrestricted charge impacts",
			filter:      NewUnrestrictedFeatureFilter(),
			wantSettled: 100,
			wantLive:    95,
		},
		{
			name:        "matching feature filter includes unrestricted and matching charge impacts",
			filter:      NewFeatureFilter([]string{testFeatureKey}),
			wantSettled: 110,
			wantLive:    68,
		},
		{
			name:        "non-matching feature filter excludes restricted charge impacts for other features",
			filter:      NewFeatureFilter([]string{"storage"}),
			wantSettled: 120,
			wantLive:    115,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
				CustomerID:    env.CustomerID,
				Currency:      env.Currency,
				FeatureFilter: tt.filter,
			})
			require.NoError(t, err)

			require.Equal(t, tt.wantSettled, balance.Settled().InexactFloat64())
			require.Equal(t, tt.wantLive, balance.Live().InexactFloat64())
		})
	}
}

func TestGetBalanceAllFeatureFilterDoesNotApplyBoundedUsageToIneligibleRestrictedCredits(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceWithFeatures(t, alpacadecimal.NewFromInt(100), []string{"storage"})
	env.fundOpenReceivableWithFeatures(t, alpacadecimal.NewFromInt(100), []string{"storage"})

	env.addUsage(30, clock.Now().Add(-30*time.Minute))
	env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditThenInvoiceSettlementMode, env.sp())

	allBalance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Equal(t, float64(100), allBalance.Settled().InexactFloat64())
	require.Equal(t, float64(100), allBalance.Live().InexactFloat64())

	storageBalance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: NewFeatureFilter([]string{"storage"}),
	})
	require.NoError(t, err)
	require.Equal(t, float64(100), storageBalance.Settled().InexactFloat64())
	require.Equal(t, float64(100), storageBalance.Live().InexactFloat64())

	apiRequestsBalance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: NewFeatureFilter([]string{testFeatureKey}),
	})
	require.NoError(t, err)
	require.Equal(t, float64(0), apiRequestsBalance.Settled().InexactFloat64())
	require.Equal(t, float64(0), apiRequestsBalance.Live().InexactFloat64())
}

func TestGetBalancePendingGrants(t *testing.T) {
	env := newTestEnv(t)

	now := clock.Now()
	futureEffectiveAt := now.Add(time.Hour)

	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(30), env.Currency)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(20), env.Currency, &futureEffectiveAt)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(10), env.Currency, nil)

	balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: AllFeatureFilter(),
		BalanceQuery: ledger.BalanceQuery{
			AsOf: &now,
		},
	})
	require.NoError(t, err)

	require.Equal(t, float64(10), balance.Settled().InexactFloat64())
	require.Equal(t, float64(0), balance.Live().InexactFloat64())
	require.Equal(t, float64(50), balance.Pending().InexactFloat64())

	afterFutureEffectiveAt := futureEffectiveAt.Add(time.Second)
	balance, err = env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: AllFeatureFilter(),
		BalanceQuery: ledger.BalanceQuery{
			AsOf: &afterFutureEffectiveAt,
		},
	})
	require.NoError(t, err)

	require.Equal(t, float64(30), balance.Settled().InexactFloat64())
	require.Equal(t, float64(0), balance.Live().InexactFloat64())
	require.Equal(t, float64(30), balance.Pending().InexactFloat64())
}

func TestGetBalancePendingInvoiceGrantBeforeDraft(t *testing.T) {
	env := newTestEnv(t)

	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(30), env.Currency)

	balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	require.Equal(t, float64(0), balance.Settled().InexactFloat64())
	require.Equal(t, float64(0), balance.Live().InexactFloat64())
	require.Equal(t, float64(30), balance.Pending().InexactFloat64())
}

func TestGetBalancePendingGrantExcludesDeletedCharge(t *testing.T) {
	env := newTestEnv(t)

	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(30), env.Currency)
	deletedCharge := env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(20), env.Currency)
	env.markCreditPurchaseDeleted(t, deletedCharge)

	balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
		CustomerID:    env.CustomerID,
		Currency:      env.Currency,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	require.Equal(t, float64(30), balance.Pending().InexactFloat64())
}

func TestIsPendingCreditGrantAt(t *testing.T) {
	now := clock.Now().UTC()
	future := now.Add(time.Hour)
	deletedBefore := now.Add(-time.Minute)
	deletedAfter := now.Add(time.Minute)
	currency := currencyx.Code("USD")

	newCharge := func() creditpurchase.Charge {
		servicePeriod := timeutil.ClosedPeriod{
			From: future,
			To:   future,
		}

		return creditpurchase.Charge{
			ChargeBase: creditpurchase.ChargeBase{
				ManagedResource: chargemeta.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: "ns"},
					ManagedModel: models.ManagedModel{
						CreatedAt: now,
						UpdatedAt: now,
					},
					ID: "charge-id",
				},
				Status: creditpurchase.StatusCreated,
				Intent: creditpurchase.Intent{
					Intent: chargemeta.Intent{
						CustomerID: "customer-id",
						Currency:   currenciestestutils.NewFiatCurrency(t, currency),
					},
					IntentMutableFields: creditpurchase.IntentMutableFields{
						IntentMutableFields: chargemeta.IntentMutableFields{
							ServicePeriod:     servicePeriod,
							BillingPeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
						},
						CreditAmount: alpacadecimal.NewFromInt(10),
						Settlement: creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
							GenericSettlement: creditpurchase.GenericSettlement{
								Currency:  currency,
								CostBasis: alpacadecimal.NewFromInt(1),
							},
						}),
					},
				},
			},
		}
	}

	realizedGrant := &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "transaction-group-id"},
		Time:           now,
	}

	tests := []struct {
		name string
		asOf time.Time
		edit func(*creditpurchase.Charge)
		want bool
	}{
		{
			name: "invoice grant before draft is pending",
			asOf: now,
			want: true,
		},
		{
			name: "charge created after as of is not pending",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.CreatedAt = future
			},
			want: false,
		},
		{
			name: "future realized grant is pending before booked time",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.Realizations.CreditGrantRealization = realizedGrant
				charge.Status = creditpurchase.StatusActive
			},
			want: true,
		},
		{
			name: "future realized grant is not pending at booked time",
			asOf: future,
			edit: func(charge *creditpurchase.Charge) {
				charge.Realizations.CreditGrantRealization = realizedGrant
				charge.Status = creditpurchase.StatusActive
			},
			want: false,
		},
		{
			name: "deleted charge status is not pending",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.Status = creditpurchase.StatusDeleted
			},
			want: false,
		},
		{
			name: "soft deleted charge is not pending after deletion time",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.DeletedAt = &deletedBefore
			},
			want: false,
		},
		{
			name: "soft deleted charge remains pending before deletion time",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.DeletedAt = &deletedAfter
			},
			want: true,
		},
		{
			name: "final charge without grant realization is not pending",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.Status = creditpurchase.StatusFinal
			},
			want: false,
		},
		{
			name: "voided invoice settlement is not pending",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.Realizations.CreditGrantRealization = realizedGrant
				charge.Realizations.InvoiceSettlement = &payment.Invoiced{
					Payment: payment.Payment{
						ManagedModel: models.ManagedModel{
							CreatedAt: now,
							UpdatedAt: now,
							DeletedAt: &deletedBefore,
						},
					},
				}
			},
			want: false,
		},
		{
			name: "voided external settlement is not pending",
			asOf: now,
			edit: func(charge *creditpurchase.Charge) {
				charge.Realizations.CreditGrantRealization = realizedGrant
				charge.Realizations.ExternalPaymentSettlement = &payment.External{
					Payment: payment.Payment{
						ManagedModel: models.ManagedModel{
							CreatedAt: now,
							UpdatedAt: now,
							DeletedAt: &deletedBefore,
						},
					},
				}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charge := newCharge()
			if tt.edit != nil {
				tt.edit(&charge)
			}

			require.Equal(t, tt.want, isPendingCreditGrantAt(charge, tt.asOf))
		})
	}
}

func TestGetBalancePendingGrantFeatureFilter(t *testing.T) {
	env := newTestEnv(t)

	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency)
	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(10), env.Currency, testFeatureKey)
	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(20), env.Currency, "storage")

	tests := []struct {
		name   string
		filter mo.Option[creditpurchase.FeatureFilters]
		want   float64
	}{
		{
			name:   "omitted filter returns all pending grants",
			filter: AllFeatureFilter(),
			want:   130,
		},
		{
			name:   "unrestricted filter returns unrestricted pending grants",
			filter: NewUnrestrictedFeatureFilter(),
			want:   100,
		},
		{
			name:   "matching feature filter includes unrestricted and matching pending grants",
			filter: NewFeatureFilter([]string{testFeatureKey}),
			want:   110,
		},
		{
			name:   "non-matching feature filter excludes restricted pending grants for other features",
			filter: NewFeatureFilter([]string{"storage"}),
			want:   120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := env.Service.GetBalance(t.Context(), GetBalanceServiceInput{
				CustomerID:    env.CustomerID,
				Currency:      env.Currency,
				FeatureFilter: tt.filter,
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, balance.Pending().InexactFloat64())
		})
	}
}
