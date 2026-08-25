package charges

import (
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestConvertAPIChargesExpand(t *testing.T) {
	// The API tokens and the domain enum diverge on purpose for
	// realization.detailed_lines; every supported token must map, and unknown
	// tokens must error so the handler can 400.
	want := map[api.BillingChargesExpand]meta.Expand{
		api.BillingChargesExpandRealTimeUsage:            meta.ExpandRealtimeUsage,
		api.BillingChargesExpandCustomer:                 meta.ExpandCustomer,
		api.BillingChargesExpandFeature:                  meta.ExpandFeature,
		api.BillingChargesExpandSubscription:             meta.ExpandSubscription,
		api.BillingChargesExpandRealizationInvoice:       meta.ExpandRealizationInvoice,
		api.BillingChargesExpandRealizationTotals:        meta.ExpandRealizationTotals,
		api.BillingChargesExpandRealizationDetailedLines: meta.ExpandDetailedLines,
	}

	for token, expected := range want {
		got, err := convertAPIChargesExpand(token)
		require.NoError(t, err, "token %s", token)
		require.Equal(t, expected, got, "token %s", token)
	}

	_, err := convertAPIChargesExpand(api.BillingChargesExpand("bogus"))
	require.ErrorContains(t, err, "unsupported expand")
}

func TestConvertExpandedUnionBranches(t *testing.T) {
	t.Run("customer expands to the full entity", func(t *testing.T) {
		out, err := convertChargeCustomerToAPI("cust-1", &customer.Customer{
			ManagedResource: models.ManagedResource{ID: "cust-1", Name: "Full Customer"},
		})
		require.NoError(t, err)

		full, err := out.AsBillingCustomer()
		require.NoError(t, err)
		require.Equal(t, "cust-1", full.Id)
		require.Equal(t, "Full Customer", full.Name)
	})

	t.Run("customer falls back to the id reference", func(t *testing.T) {
		out, err := convertChargeCustomerToAPI("cust-1", nil)
		require.NoError(t, err)

		ref, err := out.AsCustomerReference()
		require.NoError(t, err)
		require.Equal(t, "cust-1", ref.Id)
	})

	t.Run("subscription expands to the full entity", func(t *testing.T) {
		source := &meta.SubscriptionReference{SubscriptionID: "sub-1", PhaseID: "phase-1", ItemID: "item-1"}

		out, err := convertChargeSubscriptionToAPI(source, &subscription.Subscription{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "sub-1"},
			Name:         "Full Subscription",
		})
		require.NoError(t, err)
		require.NotNil(t, out)

		full, err := out.AsBillingSubscription()
		require.NoError(t, err)
		require.Equal(t, "sub-1", full.Id)
	})

	t.Run("subscription falls back to the reference and to absence", func(t *testing.T) {
		source := &meta.SubscriptionReference{SubscriptionID: "sub-1", PhaseID: "phase-1", ItemID: "item-1"}

		out, err := convertChargeSubscriptionToAPI(source, nil)
		require.NoError(t, err)
		require.NotNil(t, out)

		ref, err := out.AsBillingSubscriptionReference()
		require.NoError(t, err)
		require.Equal(t, api.ULID("sub-1"), ref.Id)

		none, err := convertChargeSubscriptionToAPI(nil, nil)
		require.NoError(t, err)
		require.Nil(t, none, "charges without a subscription have no field at all")
	})
}

func TestConvertDetailedLinesToAPI(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)}

	base := chargedetailedline.Base{
		Base: stddetailedline.Base{
			ManagedResource: models.ManagedResource{ID: "dl-1", Name: "detailed line"},
			Category:        stddetailedline.CategoryRegular,
			ServicePeriod:   period,
			PerUnitAmount:   decimal.NewFromFloat(2.5),
			Quantity:        decimal.NewFromInt(4),
			CreditsApplied: creditsapplied.CreditsApplied{
				{Amount: decimal.NewFromInt(3), Description: "promo credits", CreditRealizationID: "cr-1"},
				{Amount: decimal.NewFromInt(1), CreditRealizationID: "cr-2"},
			},
		},
		AmountDiscounts: chargedetailedline.AmountDiscounts{
			{ChildUniqueReferenceID: "disc-1", Amount: decimal.NewFromInt(2), RoundingAmount: decimal.NewFromFloat(0.01)},
			{ChildUniqueReferenceID: "disc-2", Amount: decimal.NewFromInt(1)},
		},
	}

	t.Run("usage based lines carry quantity, credits and discounts", func(t *testing.T) {
		out, err := convertUsageBasedDetailedLinesToAPI(mo.Some(usagebased.DetailedLines{{Base: base}}))
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Len(t, *out, 1)

		line, err := (*out)[0].AsBillingChargeRealizationDetailedLineUsageBased()
		require.NoError(t, err)
		require.Equal(t, "dl-1", line.Id)
		require.Equal(t, "4", line.Quantity)
		require.Equal(t, "2.5", line.UnitPrice)
		require.Equal(t, api.BillingChargeRealizationDetailedLineCategory("regular"), line.Category)

		require.NotNil(t, line.CreditsApplied)
		credits := *line.CreditsApplied
		require.Len(t, credits, 2)
		require.Equal(t, "3", credits[0].Amount)
		require.Equal(t, lo.ToPtr("promo credits"), credits[0].Description)
		require.Nil(t, credits[1].Description, "empty descriptions are omitted")

		require.Len(t, line.AmountDiscounts, 2)
		require.Equal(t, lo.ToPtr("0.01"), line.AmountDiscounts[0].RoundingAmount)
		require.Nil(t, line.AmountDiscounts[1].RoundingAmount, "zero rounding amounts are omitted")
	})

	t.Run("flat fee lines carry the same base without a quantity", func(t *testing.T) {
		out, err := convertFlatFeeDetailedLinesToAPI(mo.Some(flatfee.DetailedLines{base}))
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Len(t, *out, 1)

		line, err := (*out)[0].AsBillingChargeRealizationDetailedLineFlatFee()
		require.NoError(t, err)
		require.Equal(t, "dl-1", line.Id)
		require.Equal(t, "2.5", line.UnitPrice)
		require.NotNil(t, line.CreditsApplied)
	})

	t.Run("absent lines stay nil", func(t *testing.T) {
		out, err := convertUsageBasedDetailedLinesToAPI(mo.None[usagebased.DetailedLines]())
		require.NoError(t, err)
		require.Nil(t, out)
	})
}

func TestConvertTaxCodeConfigToAPI(t *testing.T) {
	tests := []struct {
		name  string
		input productcatalog.TaxCodeConfig
		want  *api.BillingTaxConfig
	}{
		{
			name:  "empty config returns nil",
			input: productcatalog.TaxCodeConfig{},
			want:  nil,
		},
		{
			name: "tax code ID only",
			input: productcatalog.TaxCodeConfig{
				TaxCodeID: "01JTEST00000000000000000001",
			},
			want: &api.BillingTaxConfig{
				TaxCode:   &api.TaxCodeReference{Id: "01JTEST00000000000000000001"},
				TaxCodeId: lo.ToPtr(api.ULID("01JTEST00000000000000000001")),
			},
		},
		{
			name: "both behavior and tax code ID",
			input: productcatalog.TaxCodeConfig{
				Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				TaxCodeID: "01JTEST00000000000000000002",
			},
			want: &api.BillingTaxConfig{
				Behavior:  lo.ToPtr(api.BillingTaxBehaviorExclusive),
				TaxCode:   &api.TaxCodeReference{Id: "01JTEST00000000000000000002"},
				TaxCodeId: lo.ToPtr(api.ULID("01JTEST00000000000000000002")),
			},
		},
		{
			name: "behavior only",
			input: productcatalog.TaxCodeConfig{
				Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			},
			want: &api.BillingTaxConfig{
				Behavior: lo.ToPtr(api.BillingTaxBehaviorInclusive),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTaxCodeConfigToAPI(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFromAPIChargeBillingPrice(t *testing.T) {
	newUnionPrice := func(t *testing.T, set func(*api.BillingPrice) error) api.BillingPrice {
		t.Helper()

		var p api.BillingPrice
		require.NoError(t, set(&p))
		return p
	}

	t.Run("unit price maps to a domain unit price", func(t *testing.T) {
		price := newUnionPrice(t, func(p *api.BillingPrice) error {
			return p.FromBillingPriceUnit(api.BillingPriceUnit{
				Type:   api.BillingPriceUnitTypeUnit,
				Amount: "2.5",
			})
		})

		got, err := fromAPIChargeBillingPrice(price)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, productcatalog.UnitPriceType, got.Type())

		unit, err := got.AsUnit()
		require.NoError(t, err)
		require.Equal(t, 2.5, unit.Amount.InexactFloat64())
	})

	t.Run("flat price maps to a domain flat price", func(t *testing.T) {
		price := newUnionPrice(t, func(p *api.BillingPrice) error {
			return p.FromBillingPriceFlat(api.BillingPriceFlat{
				Type:   api.BillingPriceFlatTypeFlat,
				Amount: "10",
			})
		})

		got, err := fromAPIChargeBillingPrice(price)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, productcatalog.FlatPriceType, got.Type())
	})

	t.Run("free price is rejected", func(t *testing.T) {
		price := newUnionPrice(t, func(p *api.BillingPrice) error {
			return p.FromBillingPriceFree(api.BillingPriceFree{
				Type: api.BillingPriceFreeTypeFree,
			})
		})

		_, err := fromAPIChargeBillingPrice(price)
		require.ErrorContains(t, err, "free price")
	})
}

func TestToAPIBillingChargeUsageBasedSystemIntentReturnsUnsupportedBasePriceError(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}

	intent := usagebased.NewOverridableIntent(usagebased.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.SubscriptionManagedLine,
			CustomerID: "customer-id",
			Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
		},
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "system intent",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt: now,
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: decimal.NewFromInt(1),
			}),
		},
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		FeatureKey:     "feature-key",
	}, &usagebased.IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			Name:              "override intent",
			ServicePeriod:     period,
			FullServicePeriod: period,
			BillingPeriod:     period,
		},
		InvoiceAt: now,
		Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.NewFromInt(1),
		}),
	})

	systemIntent, err := toAPIBillingChargeUsageBasedSystemIntent(intent)

	assert.Nil(t, systemIntent)
	assert.ErrorContains(t, err, "converting price")
}

// The API exposes the charge's feature as a FeatureOrReference union whose reference
// branch carries the feature version ID the charge actually resolved to — carried on
// State, not on the intent. Reading the ID off the intent silently yields the zero
// value, because OverridableIntent does not carry it.
func TestConvertChargeToAPIReadsFeatureIDFromState(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}

	t.Run("usage based", func(t *testing.T) {
		charge := usagebased.Charge{
			ChargeBase: usagebased.ChargeBase{
				ManagedResource: meta.ManagedResource{ID: "charge-id"},
				Status:          usagebased.StatusCreated,
				State: usagebased.State{
					FeatureID: "usage-feature-version-id",
				},
				Intent: usagebased.NewOverridableIntent(usagebased.Intent{
					Intent: meta.Intent{
						ManagedBy:  billing.ManuallyManagedLine,
						CustomerID: "customer-id",
						Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
					},
					IntentMutableFields: usagebased.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name:              "usage based charge",
							ServicePeriod:     period,
							FullServicePeriod: period,
							BillingPeriod:     period,
						},
						InvoiceAt: now,
						Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: decimal.NewFromInt(1),
						}),
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					FeatureKey:     "usage-feature-key",
				}, nil),
			},
		}

		out, err := convertUsageBasedChargeToAPI(billingcharges.CustomerCharge{
			Charge: billingcharges.NewCharge(charge),
		}, meta.ExpandNone)

		assert.NoError(t, err)
		featureRef, err := out.Feature.AsFeatureReference()
		assert.NoError(t, err)
		assert.Equal(t, "usage-feature-version-id", featureRef.Id)
	})

	t.Run("flat fee", func(t *testing.T) {
		charge := flatfee.Charge{
			ChargeBase: flatfee.ChargeBase{
				ManagedResource: meta.ManagedResource{ID: "charge-id"},
				Status:          flatfee.StatusCreated,
				State: flatfee.State{
					FeatureID:            lo.ToPtr("flat-fee-feature-version-id"),
					AmountAfterProration: decimal.NewFromInt(10),
				},
				Intent: flatfee.NewOverridableIntent(flatfee.Intent{
					Intent: meta.Intent{
						ManagedBy:  billing.ManuallyManagedLine,
						CustomerID: "customer-id",
						Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
					},
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name:              "flat fee charge",
							ServicePeriod:     period,
							FullServicePeriod: period,
							BillingPeriod:     period,
						},
						InvoiceAt:             now,
						PaymentTerm:           productcatalog.InAdvancePaymentTerm,
						AmountBeforeProration: decimal.NewFromInt(10),
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					FeatureKey:     lo.ToPtr("flat-fee-feature-key"),
				}, nil),
			},
		}

		out, err := convertFlatFeeChargeToAPI(billingcharges.CustomerCharge{
			Charge: billingcharges.NewCharge(charge),
		}, meta.ExpandNone)

		assert.NoError(t, err)
		featureRef, err := out.Feature.AsFeatureReference()
		assert.NoError(t, err)
		assert.Equal(t, "flat-fee-feature-version-id", featureRef.Id)
	})
}

// The domain status carries detailed sub-states the API enum does not admit;
// the converter must collapse them to the coarse charge status instead of
// leaking values like "active.realization.processing" onto the wire.
func TestConvertFlatFeeChargeToAPIMapsDetailedStatus(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}

	charge := flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-id"},
			Status:          flatfee.StatusActiveRealizationProcessing,
			State: flatfee.State{
				AmountAfterProration: decimal.NewFromInt(10),
			},
			Intent: flatfee.NewOverridableIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.ManuallyManagedLine,
					CustomerID: "customer-id",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat fee charge",
						ServicePeriod:     period,
						FullServicePeriod: period,
						BillingPeriod:     period,
					},
					InvoiceAt:             now,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: decimal.NewFromInt(10),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			}, nil),
		},
	}

	out, err := convertFlatFeeChargeToAPI(billingcharges.CustomerCharge{
		Charge: billingcharges.NewCharge(charge),
	}, meta.ExpandNone)

	assert.NoError(t, err)
	assert.Equal(t, api.BillingChargeStatusActive, out.Status)
}

// The converter maps the resolved realization view the facade attaches:
// stitched periods and per-run quantities for live runs, `voided` rows for
// voided history, and the outstanding tail projection.
func TestConvertUsageBasedRealizationsToAPI(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(2 * time.Hour),
	}

	newRun := func(id string, servicePeriodTo time.Time, meteredQuantity int64) usagebased.RealizationRun {
		return usagebased.RealizationRun{
			RealizationRunBase: usagebased.RealizationRunBase{
				ID:              usagebased.RealizationRunID(models.NamespacedID{Namespace: "ns", ID: id}),
				ManagedModel:    models.ManagedModel{CreatedAt: now},
				FeatureID:       "feature-id",
				Type:            usagebased.RealizationRunTypePartialInvoice,
				InitialType:     usagebased.RealizationRunTypePartialInvoice,
				StoredAtLT:      servicePeriodTo,
				ServicePeriodTo: servicePeriodTo,
				MeteredQuantity: decimal.NewFromInt(meteredQuantity),
			},
		}
	}

	// given two booked runs supplied out of order plus a deleted one, the second of
	// which is settled through an invoice
	firstRun := newRun("run-1", now.Add(30*time.Minute), 10)

	secondRun := newRun("run-2", now.Add(time.Hour), 25)
	secondRun.LineID = lo.ToPtr("line-2")
	secondRun.InvoiceID = lo.ToPtr("invoice-2")
	secondRun.Payment = &payment.Invoiced{
		Payment: payment.Payment{
			Base: payment.Base{Status: payment.StatusSettled},
		},
	}

	deletedRun := newRun("run-deleted", now.Add(90*time.Minute), 40)
	deletedRun.DeletedAt = lo.ToPtr(now)

	// The resolver stitches these into contiguous service periods with
	// per-run quantities and marks the deleted run voided; hand-built here
	// since the resolver itself now lives in the service package as an
	// unexported function this test package cannot call.
	bookedAndVoided := []billingcharges.CustomerChargeUsageBasedRealization{
		{
			Run:           &firstRun,
			ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(30 * time.Minute)},
			Quantity:      decimal.NewFromInt(10),
		},
		{
			Run:           &secondRun,
			ServicePeriod: timeutil.ClosedPeriod{From: now.Add(30 * time.Minute), To: now.Add(time.Hour)},
			Quantity:      decimal.NewFromInt(15),
		},
		{
			Run:           &deletedRun,
			ServicePeriod: timeutil.ClosedPeriod{From: now.Add(time.Hour), To: now.Add(90 * time.Minute)},
			Quantity:      decimal.NewFromInt(15),
			Voided:        true,
		},
	}

	realizations := append(bookedAndVoided, billingcharges.CustomerChargeUsageBasedRealization{
		ServicePeriod: timeutil.ClosedPeriod{From: now.Add(time.Hour), To: period.To},
	})

	// when converting the resolved realizations
	out, err := convertUsageBasedRealizationsToAPI(realizations, meta.ExpandNone)
	require.NoError(t, err)

	// then the booked runs are ordered and stitched into contiguous service
	// periods, the deleted run surfaces as an inert voided entry, and the tail
	// not covered by live runs is projected as outstanding
	require.Len(t, out, 4)

	assert.Equal(t, lo.ToPtr("run-1"), out[0].Id)
	assert.Equal(t, api.ClosedPeriod{From: now, To: now.Add(30 * time.Minute)}, out[0].ServicePeriod)
	assert.Equal(t, lo.ToPtr("10"), out[0].Usage)
	assert.Equal(t, api.BillingChargeRealizationTypePartialInvoice, out[0].Type)
	assert.Nil(t, out[0].Invoice)
	assert.Nil(t, out[0].Payment)
	assert.Nil(t, out[0].DetailedLines, "detailed lines stay unset without the expand")
	assert.Nil(t, out[0].Totals, "totals stay unset without the expand")

	assert.Equal(t, lo.ToPtr("run-2"), out[1].Id)
	assert.Equal(t, api.ClosedPeriod{From: now.Add(30 * time.Minute), To: now.Add(time.Hour)}, out[1].ServicePeriod)
	assert.Equal(t, lo.ToPtr("15"), out[1].Usage, "usage reports the run's own quantity, not the cumulative one")
	assert.Equal(t, lo.ToPtr("line-2"), out[1].LineId)
	require.NotNil(t, out[1].Payment)
	assert.Equal(t, api.BillingChargeRealizationPaymentStatus("settled"), out[1].Payment.Status)
	require.NotNil(t, out[1].Invoice)
	invoiceRef, err := out[1].Invoice.AsChargeRealizationInvoiceReference()
	require.NoError(t, err)
	assert.Equal(t, "invoice-2", invoiceRef.Id)

	assert.Equal(t, lo.ToPtr("run-deleted"), out[2].Id)
	assert.Equal(t, api.BillingChargeRealizationTypeVoided, out[2].Type, "voided history is marked, not hidden")
	assert.Equal(t, api.ClosedPeriod{From: now.Add(time.Hour), To: now.Add(90 * time.Minute)}, out[2].ServicePeriod)

	assert.Nil(t, out[3].Id, "the outstanding projection is not a persisted run")
	assert.Equal(t, api.BillingChargeRealizationTypeOutstanding, out[3].Type)
	assert.Equal(t, api.ClosedPeriod{From: now.Add(time.Hour), To: period.To}, out[3].ServicePeriod,
		"outstanding starts at the last live run's end; voided history does not cover the period")
	assert.Nil(t, out[3].Invoice)
	assert.Nil(t, out[3].Totals)

	// and the realization.totals expand gates per-run totals emission
	t.Run("realization totals expand", func(t *testing.T) {
		out, err := convertUsageBasedRealizationsToAPI(realizations, meta.Expands{meta.ExpandRealizationTotals})
		require.NoError(t, err)

		require.Len(t, out, 4)
		require.NotNil(t, out[0].Totals)
		assert.Nil(t, out[3].Totals, "the outstanding projection has no booked totals")
	})

	// and a live metered quantity from the realtime usage expand fills the
	// outstanding row with the not-yet-booked remainder
	t.Run("realtime quantity fills the outstanding usage", func(t *testing.T) {
		// given the same booked and voided runs, but the resolver's outstanding
		// quantity for a live read of 31: the read minus the booked non-voided
		// cumulative (10 + 15)
		live := append(bookedAndVoided, billingcharges.CustomerChargeUsageBasedRealization{
			ServicePeriod: timeutil.ClosedPeriod{From: now.Add(time.Hour), To: period.To},
			Quantity:      decimal.NewFromInt(6),
		})

		out, err := convertUsageBasedRealizationsToAPI(live, meta.ExpandNone)
		require.NoError(t, err)

		require.Len(t, out, 4)
		assert.Equal(t, lo.ToPtr("10"), out[0].Usage, "booked usage is a billing fact and stays untouched")
		assert.Equal(t, lo.ToPtr("15"), out[1].Usage)
		assert.Equal(t, lo.ToPtr("6"), out[3].Usage, "outstanding reports the live read minus the booked non-voided cumulative (31 - 25)")
	})

	// and a deleted charge keeps its original service period without ever
	// realizing the remainder, so no outstanding projection is emitted for it
	t.Run("deleted charge", func(t *testing.T) {
		out, err := convertUsageBasedRealizationsToAPI(bookedAndVoided, meta.ExpandNone)
		require.NoError(t, err)

		require.Len(t, out, 3)
		assert.Equal(t, lo.ToPtr("run-1"), out[0].Id)
		assert.Equal(t, lo.ToPtr("run-2"), out[1].Id)
		assert.Equal(t, lo.ToPtr("run-deleted"), out[2].Id)
	})
}

// A flat fee charge splits its runs between the in-progress current run and the
// prior ones; both are booked history and must land in a single ordered array.
func TestConvertFlatFeeRealizationsToAPI(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(2 * time.Hour),
	}

	newRun := func(id string, servicePeriod timeutil.ClosedPeriod) flatfee.RealizationRun {
		return flatfee.RealizationRun{
			RealizationRunBase: flatfee.RealizationRunBase{
				ID:                   flatfee.RealizationRunID(models.NamespacedID{Namespace: "ns", ID: id}),
				ManagedModel:         models.ManagedModel{CreatedAt: now},
				Type:                 flatfee.RealizationRunTypeFinalRealization,
				InitialType:          flatfee.RealizationRunTypeFinalRealization,
				ServicePeriod:        servicePeriod,
				AmountAfterProration: decimal.NewFromInt(10),
			},
		}
	}

	t.Run("booked run and outstanding entry", func(t *testing.T) {
		// given a hand-built resolved view carrying both entry kinds (a booked
		// run and an outstanding projection), exercising the converter's two
		// branches in one pass. Hand-built since the resolver lives in the
		// service package as an unexported function this test package cannot
		// call — the resolver itself never mixes the two for flat fees.
		run1 := newRun("run-1", timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)})
		realizations := []billingcharges.CustomerChargeFlatFeeRealization{
			{Run: &run1, ServicePeriod: run1.ServicePeriod},
			{ServicePeriod: timeutil.ClosedPeriod{From: now.Add(time.Hour), To: period.To}},
		}

		// when converting the resolved realizations
		out, err := convertFlatFeeRealizationsToAPI(realizations, meta.ExpandNone)
		require.NoError(t, err)

		// then the remaining half is projected as outstanding
		require.Len(t, out, 2)
		assert.Equal(t, lo.ToPtr("run-1"), out[0].Id)
		assert.Nil(t, out[0].Usage, "a flat fee is not metered, so its realizations carry no usage")
		assert.Equal(t, api.BillingChargeRealizationTypeOutstanding, out[1].Type)
		assert.Equal(t, api.ClosedPeriod{From: now.Add(time.Hour), To: period.To}, out[1].ServicePeriod)
		assert.Nil(t, out[1].Usage, "the outstanding projection of a flat fee carries no usage either")
	})

	t.Run("fully covered service period", func(t *testing.T) {
		// given a prior run and a current run that together cover the whole
		// period, so the resolver produces no outstanding projection
		run1 := newRun("run-1", timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)})
		run2 := newRun("run-2", timeutil.ClosedPeriod{From: now.Add(time.Hour), To: period.To})
		realizations := []billingcharges.CustomerChargeFlatFeeRealization{
			{Run: &run1, ServicePeriod: run1.ServicePeriod},
			{Run: &run2, ServicePeriod: run2.ServicePeriod},
		}

		// when converting the resolved realizations
		out, err := convertFlatFeeRealizationsToAPI(realizations, meta.ExpandNone)
		require.NoError(t, err)

		// then both runs are returned in service-period order with nothing left
		// outstanding
		require.Len(t, out, 2)
		assert.Equal(t, lo.ToPtr("run-1"), out[0].Id)
		assert.Equal(t, lo.ToPtr("run-2"), out[1].Id)
		assert.Equal(t, api.BillingChargeRealizationTypeFinalRealization, out[1].Type)
	})
}

// The contract marks realizations as a required array, so a nil slice would
// serialize as null and violate the published schema.
func TestConvertChargeToAPISerializesRequiredRealizations(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-id"},
			Status:          usagebased.StatusCreated,
			State: usagebased.State{
				FeatureID: "usage-feature-version-id",
			},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.ManuallyManagedLine,
					CustomerID: "customer-id",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "usage based charge",
						ServicePeriod:     period,
						FullServicePeriod: period,
						BillingPeriod:     period,
					},
					InvoiceAt: now,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: decimal.NewFromInt(1),
					}),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     "usage-feature-key",
			}, nil),
		},
	}

	out, err := convertUsageBasedChargeToAPI(billingcharges.CustomerCharge{
		Charge: billingcharges.NewCharge(charge),
	}, meta.ExpandNone)
	require.NoError(t, err)

	raw, err := json.Marshal(out)
	require.NoError(t, err)

	var wire map[string]any
	require.NoError(t, json.Unmarshal(raw, &wire))
	assert.NotNil(t, wire["realizations"], "required realizations must not serialize as null")
}
