package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())
			},
			wantSettled: 100,
			wantPending: 70,
		},
		{
			name: "flat fee credit then invoice",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode, env.sp())
			},
			wantSettled: 20,
			wantPending: 0,
		},
		{
			name: "usage based credit only",
			setup: func(t *testing.T, env *testEnv) {
				env.addUsage(30, clock.Now().Add(-30*time.Minute))
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
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
				env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, futureServicePeriod)
				env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, futureServicePeriod)
			},
			wantSettled: 100,
			wantPending: 100,
		},
		{
			name: "already realized credits are not applied twice",
			setup: func(t *testing.T, env *testEnv) {
				env.bookFBOBalance(t, alpacadecimal.NewFromInt(70))

				charge := env.createFlatFeeCharge(t,
					alpacadecimal.NewFromInt(30),
					productcatalog.CreditOnlySettlementMode,
					env.sp(),
				)

				env.advanceFlatFeeCharge(t, charge)
			},
			wantSettled: 70,
			wantPending: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			tt.setup(t, env)

			priority := ledger.DefaultCustomerFBOPriority
			balance, err := env.Service.GetBalance(t.Context(), env.CustomerID, ledger.RouteFilter{
				Currency:       env.Currency,
				CreditPriority: &priority,
			}, nil)
			require.NoError(t, err)
			require.True(t, balance.Settled().Equal(alpacadecimal.NewFromInt(tt.wantSettled)))
			require.True(t, balance.Pending().Equal(alpacadecimal.NewFromInt(tt.wantPending)))
		})
	}
}

func TestGetBalanceWithDifferentCurrency(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp(), "USD")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(70), productcatalog.CreditOnlySettlementMode, env.sp(), "EUR")

	usdPriority := ledger.DefaultCustomerFBOPriority
	usdBalance, err := env.Service.GetBalance(t.Context(), env.CustomerID, ledger.RouteFilter{
		Currency:       currencyx.Code("USD"),
		CreditPriority: &usdPriority,
	}, nil)
	require.NoError(t, err)
	require.True(t, usdBalance.Settled().Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, usdBalance.Pending().Equal(alpacadecimal.NewFromInt(70)))

	eurPriority := ledger.DefaultCustomerFBOPriority
	eurBalance, err := env.Service.GetBalance(t.Context(), env.CustomerID, ledger.RouteFilter{
		Currency:       currencyx.Code("EUR"),
		CreditPriority: &eurPriority,
	}, nil)
	require.NoError(t, err)
	require.True(t, eurBalance.Settled().Equal(alpacadecimal.NewFromInt(200)))
	require.True(t, eurBalance.Pending().Equal(alpacadecimal.NewFromInt(130)))
}
