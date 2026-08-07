package charges

import (
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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

		out, err := convertUsageBasedChargeToAPI(charge)

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

		out, err := convertFlatFeeChargeToAPI(charge)

		assert.NoError(t, err)
		featureRef, err := out.Feature.AsFeatureReference()
		assert.NoError(t, err)
		assert.Equal(t, "flat-fee-feature-version-id", featureRef.Id)
	})

	t.Run("flat fee without feature", func(t *testing.T) {
		// given a flat-fee charge with no associated feature, which is a legal
		// domain state (State.FeatureID is a nullable pointer)
		charge := flatfee.Charge{
			ChargeBase: flatfee.ChargeBase{
				ManagedResource: meta.ManagedResource{ID: "charge-id"},
				Status:          flatfee.StatusCreated,
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

		// when converting to the API representation
		out, err := convertFlatFeeChargeToAPI(charge)

		// then the conversion succeeds with no feature set
		assert.NoError(t, err)
		assert.Nil(t, out.Feature)
	})
}

// The contract marks realizations as a required array, so a nil slice would
// serialize as null and violate the published schema; pin the wire shape until
// the realization mapping lands.
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

	out, err := convertUsageBasedChargeToAPI(charge)
	require.NoError(t, err)

	raw, err := json.Marshal(out)
	require.NoError(t, err)

	var wire map[string]any
	require.NoError(t, json.Unmarshal(raw, &wire))
	assert.NotNil(t, wire["realizations"], "required realizations must not serialize as null")
}
