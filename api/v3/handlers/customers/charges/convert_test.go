package charges

import (
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

func TestToAPIBillingChargeUsageBasedSystemIntentMapsLegacyDynamicPrice(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}
	minimumAmount := decimal.NewFromInt(10)

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
				Multiplier: decimal.NewFromFloat(1.2),
				Commitments: productcatalog.Commitments{
					MinimumAmount: &minimumAmount,
				},
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

	require.NoError(t, err)
	require.NotNil(t, systemIntent)

	price, err := systemIntent.Price.AsBillingPriceUnit()
	require.NoError(t, err)
	assert.Equal(t, api.Numeric("1"), price.Amount)
	require.NotNil(t, systemIntent.Commitments)
	assert.Equal(t, lo.ToPtr(api.Numeric("10")), systemIntent.Commitments.MinimumAmount)
	require.NotNil(t, systemIntent.UnitConfig)
	assert.Equal(t, api.BillingUnitConfigOperationMultiply, systemIntent.UnitConfig.Operation)
	assert.Equal(t, api.Numeric("1.2"), systemIntent.UnitConfig.ConversionFactor)
}

func TestConvertUsageBasedChargeToAPIMapsLegacyPackagePrice(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}
	maximumAmount := decimal.NewFromInt(100)

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-id"},
			Status:          usagebased.StatusCreated,
			State:           usagebased.State{FeatureID: "feature-id"},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SubscriptionManagedLine,
					CustomerID: "customer-id",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "package charge",
						ServicePeriod:     period,
						FullServicePeriod: period,
						BillingPeriod:     period,
					},
					InvoiceAt: now,
					Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
						Amount:             decimal.NewFromInt(10),
						QuantityPerPackage: decimal.NewFromInt(1000),
						Commitments: productcatalog.Commitments{
							MaximumAmount: &maximumAmount,
						},
					}),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     "feature-key",
			}, nil),
		},
	}

	result, err := convertUsageBasedChargeToAPI(charge)
	require.NoError(t, err)

	price, err := result.Price.AsBillingPriceUnit()
	require.NoError(t, err)
	assert.Equal(t, api.Numeric("10"), price.Amount)
	require.NotNil(t, result.Commitments)
	assert.Equal(t, lo.ToPtr(api.Numeric("100")), result.Commitments.MaximumAmount)
	require.NotNil(t, result.UnitConfig)
	assert.Equal(t, api.BillingUnitConfigOperationDivide, result.UnitConfig.Operation)
	assert.Equal(t, api.Numeric("1000"), result.UnitConfig.ConversionFactor)
	assert.Equal(t, lo.ToPtr(api.BillingUnitConfigRoundingModeCeiling), result.UnitConfig.Rounding)
}

func TestToAPIBillingUsageBasedRatingConfigurationPreservesStoredUnitConfig(t *testing.T) {
	minimumAmount := decimal.NewFromInt(5)
	displayUnit := "K requests"
	intent := usagebased.IntentMutableFields{
		Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.NewFromFloat(0.25),
			Commitments: productcatalog.Commitments{
				MinimumAmount: &minimumAmount,
			},
		}),
		UnitConfig: &productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         productcatalog.UnitConfigRoundingModeHalfUp,
			Precision:        2,
			DisplayUnit:      &displayUnit,
		},
	}

	result, err := toAPIBillingUsageBasedRatingConfiguration(intent)
	require.NoError(t, err)

	price, err := result.Price.AsBillingPriceUnit()
	require.NoError(t, err)
	assert.Equal(t, api.Numeric("0.25"), price.Amount)
	require.NotNil(t, result.Commitments)
	assert.Equal(t, lo.ToPtr(api.Numeric("5")), result.Commitments.MinimumAmount)
	require.NotNil(t, result.UnitConfig)
	assert.Equal(t, api.BillingUnitConfigOperationDivide, result.UnitConfig.Operation)
	assert.Equal(t, api.Numeric("1000"), result.UnitConfig.ConversionFactor)
	assert.Equal(t, lo.ToPtr(api.BillingUnitConfigRoundingModeHalfUp), result.UnitConfig.Rounding)
	assert.Equal(t, lo.ToPtr(2), result.UnitConfig.Precision)
	assert.Equal(t, &displayUnit, result.UnitConfig.DisplayUnit)
}

func TestFromAPICreateChargeUsageBasedRequestMapsRatingConfiguration(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	period := api.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}

	var price api.BillingPriceUsageBased
	require.NoError(t, price.FromBillingPriceUnit(api.BillingPriceUnit{
		Amount: "0.25",
		Type:   api.BillingPriceUnitTypeUnit,
	}))

	input, err := fromAPICreateChargeUsageBasedRequest("namespace", "customer-id", api.CreateChargeUsageBasedRequest{
		Commitments: &api.BillingSpendCommitments{
			MaximumAmount: lo.ToPtr(api.Numeric("50")),
			MinimumAmount: lo.ToPtr(api.Numeric("5")),
		},
		Currency:       api.CurrencyCode("USD"),
		FeatureId:      "feature-id",
		InvoiceAt:      now,
		Name:           "usage charge",
		Price:          price,
		ServicePeriod:  period,
		SettlementMode: api.BillingSettlementModeCreditThenInvoice,
		Type:           api.CreateChargeUsageBasedRequestTypeUsageBased,
		UnitConfig: &api.BillingUnitConfig{
			Operation:        api.BillingUnitConfigOperationDivide,
			ConversionFactor: "1000",
			Rounding:         lo.ToPtr(api.BillingUnitConfigRoundingModeCeiling),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, input.UsageBased)

	unitPrice, err := input.UsageBased.IntentMutableFields.Price.AsUnit()
	require.NoError(t, err)
	assert.Equal(t, "0.25", unitPrice.Amount.String())
	require.NotNil(t, unitPrice.Commitments.MinimumAmount)
	assert.Equal(t, "5", unitPrice.Commitments.MinimumAmount.String())
	require.NotNil(t, unitPrice.Commitments.MaximumAmount)
	assert.Equal(t, "50", unitPrice.Commitments.MaximumAmount.String())
	require.NotNil(t, input.UsageBased.IntentMutableFields.UnitConfig)
	assert.Equal(t, productcatalog.UnitConfigOperationDivide, input.UsageBased.IntentMutableFields.UnitConfig.Operation)
	assert.Equal(t, "1000", input.UsageBased.IntentMutableFields.UnitConfig.ConversionFactor.String())
	assert.Equal(t, productcatalog.UnitConfigRoundingModeCeiling, input.UsageBased.IntentMutableFields.UnitConfig.Rounding)
}

// The API exposes two distinct feature references: feature_key is the charge's logical
// identity, carried on the intent, while feature_id is the feature version the charge
// actually resolved to, carried on State. Reading feature_id off the intent silently
// yields the zero value, because OverridableIntent does not carry it.
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
		assert.Equal(t, "usage-feature-key", out.FeatureKey)
		assert.Equal(t, "usage-feature-version-id", out.FeatureId)
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
		assert.Equal(t, lo.ToPtr("flat-fee-feature-key"), out.FeatureKey)
		assert.Equal(t, lo.ToPtr("flat-fee-feature-version-id"), out.FeatureId)
		assert.Equal(t, api.BillingPriceFlat{
			Amount: "10",
			Type:   api.BillingPriceFlatTypeFlat,
		}, out.Price)
	})
}
