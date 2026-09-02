package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestBuildCustomerCharge exercises the pure assembly step against an
// in-memory charge: the DB-backed facade suite cannot create a charge with a
// subscription reference (FK to real subscription rows), so the
// subscription/invoice attach wiring is covered here instead.
func TestBuildCustomerCharge(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{From: now, To: now.Add(2 * time.Hour)}

	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID:              usagebased.RealizationRunID(models.NamespacedID{Namespace: "ns", ID: "run-1"}),
			ManagedModel:    models.ManagedModel{CreatedAt: now},
			FeatureID:       "feat-1",
			Type:            usagebased.RealizationRunTypePartialInvoice,
			InitialType:     usagebased.RealizationRunTypePartialInvoice,
			StoredAtLT:      now.Add(time.Hour),
			ServicePeriodTo: now.Add(time.Hour),
			MeteredQuantity: alpacadecimal.NewFromInt(5),
			InvoiceID:       lo.ToPtr("inv-1"),
			LineID:          lo.ToPtr("line-1"),
		},
	}

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-1"},
			Status:          usagebased.StatusActive,
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SubscriptionManagedLine,
					CustomerID: "cust-1",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
					Subscription: &meta.SubscriptionReference{
						SubscriptionID: "sub-1",
						PhaseID:        "phase-1",
						ItemID:         "item-1",
					},
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "attach test charge",
						ServicePeriod:     period,
						FullServicePeriod: period,
						BillingPeriod:     period,
					},
					InvoiceAt: period.To,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     "feat-key",
			}, nil),
			State: usagebased.State{
				FeatureID:    "feat-1",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
		Realizations: usagebased.RealizationRuns{run},
	}

	entities := customerChargeEntities{
		customer:          &customer.Customer{ManagedResource: models.ManagedResource{ID: "cust-1", Name: "Attach Customer"}},
		featuresByRef:     map[ref.IDOrKey]feature.Feature{{ID: "feat-1"}: {ID: "feat-1", Name: "Attach Feature"}},
		subscriptionsByID: map[string]subscription.Subscription{"sub-1": {NamespacedID: models.NamespacedID{Namespace: "ns", ID: "sub-1"}, Name: "Attach Subscription"}},
		invoiceLinesByID:  map[string]billing.StandardInvoice{"line-1": {}},
	}

	// when attaching the loaded entities
	out, err := buildCustomerCharge(charges.NewCharge(charge), entities)
	require.NoError(t, err)

	// then every expanded member resolves from its loaded entity, and the
	// booked run carries the attached invoice
	require.NotNil(t, out.Customer)
	require.Equal(t, "Attach Customer", out.Customer.Name)

	require.NotNil(t, out.Feature)
	require.Equal(t, "Attach Feature", out.Feature.Name)

	require.NotNil(t, out.Subscription)
	require.Equal(t, "sub-1", out.Subscription.ID)

	require.Len(t, out.UsageBasedRealizations, 2)
	require.NotNil(t, out.UsageBasedRealizations[0].Invoice, "the booked run carries the loaded invoice")
	require.Nil(t, out.UsageBasedRealizations[1].Invoice, "the outstanding projection never has an invoice")

	// and without loaded entities every expanded member stays nil
	bare, err := buildCustomerCharge(charges.NewCharge(charge), customerChargeEntities{})
	require.NoError(t, err)
	require.Nil(t, bare.Customer)
	require.Nil(t, bare.Feature)
	require.Nil(t, bare.Subscription)
}

func TestVisibleResolvedCostBasis(t *testing.T) {
	fiat, err := currencyx.NewFiatCurrency(currencyx.Code("USD"))
	require.NoError(t, err)

	periodFrom := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	state := &costbasis.State{CostBasis: alpacadecimal.NewFromFloat(1.5), ResolvedAt: periodFrom}
	dynamic := lo.ToPtr(costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: fiat}))
	manual := lo.ToPtr(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: fiat, Rate: alpacadecimal.NewFromFloat(1.5)}))

	tests := []struct {
		name    string
		intent  *costbasis.Intent
		state   *costbasis.State
		now     time.Time
		visible bool
	}{
		{name: "no intent", now: periodFrom},
		{name: "unresolved", intent: dynamic, now: periodFrom},
		{name: "manual is visible before the service period", intent: manual, state: state, now: periodFrom.Add(-time.Hour), visible: true},
		{name: "dynamic is hidden before the service period", intent: dynamic, state: state, now: periodFrom.Add(-time.Hour)},
		{name: "dynamic is visible from the service period start", intent: dynamic, state: state, now: periodFrom, visible: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := visibleResolvedCostBasis(tt.intent, tt.state, periodFrom, tt.now)
			if tt.visible {
				require.Equal(t, state, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}
