package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetBalance(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, env *testEnv)
		wantSettled int64
		wantPending int64
	}{
		{
			name: "flat fee credit only",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 100,
			wantPending: 70,
		},
		{
			name: "flat fee credit then invoice",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(20))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode, env.sp())
			},
			wantSettled: 20,
			wantPending: 0,
		},
		{
			name: "credit only charge with no starting credits creates advance",
			setup: func(t *testing.T, env *testEnv) {
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 0,
			wantPending: -30,
		},
		{
			name: "credit only charge advance settles",
			setup: func(t *testing.T, env *testEnv) {
				ch := env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
				env.passTimeAfterServicePeriod(t, env.sp())
				env.advanceFlatFeeCharge(t, ch)
			},
			wantSettled: -30,
			wantPending: -30,
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
			wantPending: 70,
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
			wantPending: 0,
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
			wantPending: -130,
		},
		{
			name: "future charges are excluded until service period starts",
			setup: func(t *testing.T, env *testEnv) {
				futureServicePeriod := timeutil.ClosedPeriod{
					From: clock.Now().Add(time.Hour),
					To:   clock.Now().Add(2 * time.Hour),
				}

				env.addUsage(30, clock.Now().Add(-30*time.Minute))
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
				env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, futureServicePeriod)
				env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, futureServicePeriod)
			},
			wantSettled: 100,
			wantPending: 100,
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
			wantPending: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			tt.setup(t, env)

			balance, err := env.Service.GetBalance(t.Context(), env.CustomerID, env.Currency, nil)
			require.NoError(t, err)
			assert.True(t, balance.Settled().Equal(alpacadecimal.NewFromInt(tt.wantSettled)), "settled: %s", balance.Settled())
			assert.True(t, balance.Pending().Equal(alpacadecimal.NewFromInt(tt.wantPending)), "pending: %s", balance.Pending())
		})
	}
}

func TestImpactRealizedCreditsSkipsVoidedUsageBasedBillingHistory(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	impact, err := NewImpact(charges.NewCharge(usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			Intent: usagebased.Intent{
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
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
			},
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

	usdBalance, err := env.Service.GetBalance(t.Context(), env.CustomerID, currencyx.Code("USD"), nil)
	require.NoError(t, err)
	require.True(t, usdBalance.Settled().Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, usdBalance.Pending().Equal(alpacadecimal.NewFromInt(70)))

	eurBalance, err := env.Service.GetBalance(t.Context(), env.CustomerID, currencyx.Code("EUR"), nil)
	require.NoError(t, err)
	require.True(t, eurBalance.Settled().Equal(alpacadecimal.NewFromInt(200)))
	require.True(t, eurBalance.Pending().Equal(alpacadecimal.NewFromInt(130)))
}
