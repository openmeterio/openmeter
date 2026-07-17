package addons

import (
	"encoding/json"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestBillingRateCardCurrencyRoundTrip(t *testing.T) {
	customCurrency := currencyx.Code("CREDITS")
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:      "fee",
			Name:     "Fee",
			Currency: customCurrency,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
	}

	apiRateCard, err := ToAPIBillingRateCard(rateCard)
	require.NoError(t, err)
	require.NotNil(t, apiRateCard.Currency)
	assert.Equal(t, customCurrency.String(), *apiRateCard.Currency)

	roundTripped, err := FromAPIBillingRateCard(apiRateCard)
	require.NoError(t, err)
	require.NotNil(t, roundTripped.AsMeta().Currency)
	assert.Equal(t, customCurrency, roundTripped.AsMeta().Currency.GetCode())
}

func roundTripStaticConfig(t *testing.T, config interface{}) interface{} {
	t.Helper()

	var apiEnt apiv3.BillingRateCardEntitlement
	require.NoError(t, apiEnt.FromBillingRateCardStaticEntitlement(apiv3.BillingRateCardStaticEntitlement{
		Type:   "static",
		Config: config,
	}))

	domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
	require.NoError(t, err)

	out, err := ToAPIBillingRateCardEntitlement(domain)
	require.NoError(t, err)

	static, err := out.AsBillingRateCardStaticEntitlement()
	require.NoError(t, err)

	return static.Config
}

func TestFromToAPIBillingRateCardEntitlement(t *testing.T) {
	t.Run("static — object config round-trips as a JSON object, not a string", func(t *testing.T) {
		config := map[string]interface{}{
			"integrations": []interface{}{"github", "slack"},
			"limit":        float64(10),
		}

		out := roundTripStaticConfig(t, config)

		assert.Equal(t, config, out)

		_, isObject := out.(map[string]interface{})
		assert.True(t, isObject, "config must round-trip as a JSON object")
	})

	t.Run("static — array config round-trips", func(t *testing.T) {
		config := []interface{}{float64(1), float64(2), float64(3)}
		assert.Equal(t, config, roundTripStaticConfig(t, config))
	})

	t.Run("static — scalar string config round-trips", func(t *testing.T) {
		config := "just-a-string"
		assert.Equal(t, config, roundTripStaticConfig(t, config))
	})

	t.Run("static — nil config round-trips as nil", func(t *testing.T) {
		assert.Nil(t, roundTripStaticConfig(t, nil))
	})

	t.Run("static — domain stores the v1 string-token convention", func(t *testing.T) {
		// Subscription materialization (subscriptionspec.go) unwraps the stored
		// config as a JSON string token containing JSON text. Pin that convention
		// so v3-created static entitlements stay materializable.
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardStaticEntitlement(apiv3.BillingRateCardStaticEntitlement{
			Type:   "static",
			Config: map[string]interface{}{"integrations": []interface{}{"github"}},
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.NoError(t, err)

		static, err := domain.AsStatic()
		require.NoError(t, err)

		var text string
		require.NoError(t, json.Unmarshal(static.Config, &text),
			"stored config must be a JSON string token wrapping the JSON text")
		assert.JSONEq(t, `{"integrations":["github"]}`, text)
	})

	t.Run("static — legacy non-token stored config is returned as stored, not an error", func(t *testing.T) {
		// Reads must never hard-fail on data that was valid to persist.
		domain := productcatalog.NewEntitlementTemplateFrom(productcatalog.StaticEntitlementTemplate{
			Config: json.RawMessage(`{"a":1}`),
		})

		out, err := ToAPIBillingRateCardEntitlement(domain)
		require.NoError(t, err)

		static, err := out.AsBillingRateCardStaticEntitlement()
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"a": float64(1)}, static.Config)
	})

	t.Run("metered — limit, soft limit, usage period round-trip", func(t *testing.T) {
		usagePeriod := apiv3.ISO8601Duration("P1M")

		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type:        "metered",
			IsSoftLimit: lo.ToPtr(true),
			UsagePeriod: &usagePeriod,
			Limit:       lo.ToPtr(float64(1000)),
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.NoError(t, err)

		out, err := ToAPIBillingRateCardEntitlement(domain)
		require.NoError(t, err)

		metered, err := out.AsBillingRateCardMeteredEntitlement()
		require.NoError(t, err)

		assert.Equal(t, "metered", string(metered.Type))
		require.NotNil(t, metered.Limit)
		assert.Equal(t, float64(1000), *metered.Limit)
		require.NotNil(t, metered.IsSoftLimit)
		assert.True(t, *metered.IsSoftLimit)
		require.NotNil(t, metered.UsagePeriod)
		assert.Equal(t, "P1M", *metered.UsagePeriod)
	})

	t.Run("metered — usage period defaults to billing cadence when omitted", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type:  "metered",
			Limit: lo.ToPtr(float64(5)),
		}))

		cadence, err := datetime.ISODurationString("P3M").Parse()
		require.NoError(t, err)

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, &cadence)
		require.NoError(t, err)

		metered, err := domain.AsMetered()
		require.NoError(t, err)
		assert.Equal(t, "P3M", metered.UsagePeriod.ISOString().String())
	})

	t.Run("metered — missing usage period with no billing cadence is a validation error", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type: "metered",
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.Error(t, err)
		assert.Nil(t, domain)
	})

	t.Run("metered — malformed usage period is a validation error", func(t *testing.T) {
		bad := apiv3.ISO8601Duration("not-a-duration")
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type:        "metered",
			UsagePeriod: &bad,
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid usage period")
		assert.True(t, models.IsGenericValidationError(err), "malformed durations must surface as validation errors")
		assert.Nil(t, domain)
	})

	t.Run("unknown entitlement type is rejected", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, json.Unmarshal([]byte(`{"type":"unknown"}`), &apiEnt))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported entitlement type")
		assert.Nil(t, domain)
	})

	t.Run("boolean — round-trips", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardBooleanEntitlement(apiv3.BillingRateCardBooleanEntitlement{
			Type: "boolean",
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.NoError(t, err)

		out, err := ToAPIBillingRateCardEntitlement(domain)
		require.NoError(t, err)

		boolean, err := out.AsBillingRateCardBooleanEntitlement()
		require.NoError(t, err)
		assert.Equal(t, "boolean", string(boolean.Type))
	})
}

func TestToAPIBillingRateCardUnitConfig(t *testing.T) {
	t.Run("nil price has no unit config", func(t *testing.T) {
		result, err := ToAPIBillingRateCardUnitConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("flat price has no unit config", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromFloat(5)})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("unit price has no unit config", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromFloat(0.05)})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("dynamic price produces multiply unit config", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: decimal.NewFromFloat(1.2),
		})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, apiv3.BillingUnitConfigOperationMultiply, result.Operation)
		assert.Equal(t, apiv3.Numeric("1.2"), result.ConversionFactor)
		assert.Nil(t, result.Rounding)
		assert.Nil(t, result.Precision)
		assert.Nil(t, result.DisplayUnit)
	})

	t.Run("package price produces divide unit config with ceiling rounding", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             decimal.NewFromFloat(10),
			QuantityPerPackage: decimal.NewFromInt(1000),
		})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, apiv3.BillingUnitConfigOperationDivide, result.Operation)
		assert.Equal(t, apiv3.Numeric("1000"), result.ConversionFactor)
		require.NotNil(t, result.Rounding)
		assert.Equal(t, apiv3.BillingUnitConfigRoundingModeCeiling, *result.Rounding)
	})
}

func TestToAPIBillingRateCard_DynamicAndPackagePrices(t *testing.T) {
	cadence, err := datetime.ISODurationString("P1M").Parse()
	require.NoError(t, err)

	t.Run("dynamic price renders as unit price plus multiply unit config and preserves commitments", func(t *testing.T) {
		minAmt := decimal.NewFromFloat(10)
		maxAmt := decimal.NewFromFloat(100)
		price := productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: decimal.NewFromFloat(1.2),
			Commitments: productcatalog.Commitments{
				MinimumAmount: &minAmt,
				MaximumAmount: &maxAmt,
			},
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "tokens",
				Name:  "Tokens",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.Price.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, apiv3.Numeric("1"), unit.Amount)

		require.NotNil(t, result.UnitConfig)
		assert.Equal(t, apiv3.BillingUnitConfigOperationMultiply, result.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1.2"), result.UnitConfig.ConversionFactor)
		assert.Nil(t, result.UnitConfig.Rounding)

		require.NotNil(t, result.Commitments)
		assert.Equal(t, lo.ToPtr(apiv3.Numeric("10")), result.Commitments.MinimumAmount)
		assert.Equal(t, lo.ToPtr(apiv3.Numeric("100")), result.Commitments.MaximumAmount)
	})

	t.Run("package price renders as unit price plus divide unit config and preserves commitments", func(t *testing.T) {
		minAmt := decimal.NewFromFloat(5)
		price := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             decimal.NewFromFloat(0.5),
			QuantityPerPackage: decimal.NewFromInt(1000),
			Commitments: productcatalog.Commitments{
				MinimumAmount: &minAmt,
			},
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "api-calls",
				Name:  "API Calls",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.Price.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, apiv3.Numeric("0.5"), unit.Amount)

		require.NotNil(t, result.UnitConfig)
		assert.Equal(t, apiv3.BillingUnitConfigOperationDivide, result.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1000"), result.UnitConfig.ConversionFactor)
		require.NotNil(t, result.UnitConfig.Rounding)
		assert.Equal(t, apiv3.BillingUnitConfigRoundingModeCeiling, *result.UnitConfig.Rounding)

		require.NotNil(t, result.Commitments)
		assert.Equal(t, lo.ToPtr(apiv3.Numeric("5")), result.Commitments.MinimumAmount)
		assert.Nil(t, result.Commitments.MaximumAmount)
	})

	t.Run("unit price has no unit config on rate card", func(t *testing.T) {
		price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.NewFromFloat(0.05),
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "api-calls",
				Name:  "API Calls",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)
		assert.Nil(t, result.UnitConfig)
	})
}

func TestUnitConfigMapping(t *testing.T) {
	t.Run("FromAPI maps all five fields", func(t *testing.T) {
		uc, err := FromAPIBillingUnitConfig(apiv3.BillingUnitConfig{
			Operation:        apiv3.BillingUnitConfigOperationDivide,
			ConversionFactor: apiv3.Numeric("1000"),
			Rounding:         lo.ToPtr(apiv3.BillingUnitConfigRoundingModeCeiling),
			Precision:        lo.ToPtr(2),
			DisplayUnit:      lo.ToPtr("K"),
		})
		require.NoError(t, err)
		require.NotNil(t, uc)
		assert.Equal(t, productcatalog.UnitConfigOperationDivide, uc.Operation)
		assert.Equal(t, float64(1000), uc.ConversionFactor.InexactFloat64())
		assert.Equal(t, productcatalog.UnitConfigRoundingModeCeiling, uc.Rounding)
		assert.Equal(t, 2, uc.Precision)
		assert.Equal(t, "K", lo.FromPtr(uc.DisplayUnit))
	})

	t.Run("FromAPI defaults rounding to none and precision to zero when omitted", func(t *testing.T) {
		uc, err := FromAPIBillingUnitConfig(apiv3.BillingUnitConfig{
			Operation:        apiv3.BillingUnitConfigOperationMultiply,
			ConversionFactor: apiv3.Numeric("1.2"),
		})
		require.NoError(t, err)
		require.NotNil(t, uc)
		assert.True(t, uc.Rounding.IsNone())
		assert.Equal(t, 0, uc.Precision)
		assert.Nil(t, uc.DisplayUnit)
	})

	t.Run("FromAPI drops precision when rounding is omitted, mirroring ToAPI", func(t *testing.T) {
		// Precision is inert without rounding; carrying it would make From/To
		// disagree about what is stored across a round-trip.
		uc, err := FromAPIBillingUnitConfig(apiv3.BillingUnitConfig{
			Operation:        apiv3.BillingUnitConfigOperationMultiply,
			ConversionFactor: apiv3.Numeric("1.2"),
			Precision:        lo.ToPtr(3),
		})
		require.NoError(t, err)
		require.NotNil(t, uc)
		assert.True(t, uc.Rounding.IsNone())
		assert.Equal(t, 0, uc.Precision)
	})

	t.Run("FromAPI rejects a non-numeric conversion factor", func(t *testing.T) {
		_, err := FromAPIBillingUnitConfig(apiv3.BillingUnitConfig{
			Operation:        apiv3.BillingUnitConfigOperationDivide,
			ConversionFactor: apiv3.Numeric("not-a-number"),
		})
		require.Error(t, err)
	})

	t.Run("ToAPI omits rounding and precision when rounding is none", func(t *testing.T) {
		out := ToAPIBillingUnitConfig(productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationMultiply,
			ConversionFactor: decimal.NewFromFloat(1.2),
			Precision:        3, // inert without rounding, must not surface
		})
		assert.Equal(t, apiv3.BillingUnitConfigOperationMultiply, out.Operation)
		assert.Equal(t, apiv3.Numeric("1.2"), out.ConversionFactor)
		assert.Nil(t, out.Rounding)
		assert.Nil(t, out.Precision)
	})

	t.Run("round-trips through FromAPI and ToAPI", func(t *testing.T) {
		in := apiv3.BillingUnitConfig{
			Operation:        apiv3.BillingUnitConfigOperationDivide,
			ConversionFactor: apiv3.Numeric("1000"),
			Rounding:         lo.ToPtr(apiv3.BillingUnitConfigRoundingModeCeiling),
			Precision:        lo.ToPtr(0),
			DisplayUnit:      lo.ToPtr("GB"),
		}

		domain, err := FromAPIBillingUnitConfig(in)
		require.NoError(t, err)

		assert.Equal(t, in, ToAPIBillingUnitConfig(*domain))
	})

	t.Run("a stored unit config is surfaced verbatim on read, ahead of v1 synthesis", func(t *testing.T) {
		cadence, err := datetime.ISODurationString("P1M").Parse()
		require.NoError(t, err)

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "storage",
				Name:  "Storage",
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromFloat(0.05)}),
				UnitConfig: &productcatalog.UnitConfig{
					Operation:        productcatalog.UnitConfigOperationDivide,
					ConversionFactor: decimal.NewFromInt(1_000_000_000),
					Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
					DisplayUnit:      lo.ToPtr("GB"),
				},
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)
		require.NotNil(t, result.UnitConfig)
		assert.Equal(t, apiv3.BillingUnitConfigOperationDivide, result.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1000000000"), result.UnitConfig.ConversionFactor)
		require.NotNil(t, result.UnitConfig.Rounding)
		assert.Equal(t, apiv3.BillingUnitConfigRoundingModeCeiling, *result.UnitConfig.Rounding)
		assert.Equal(t, lo.ToPtr("GB"), result.UnitConfig.DisplayUnit)
	})

	t.Run("FromAPIBillingRateCard propagates unit_config into the rate card meta", func(t *testing.T) {
		// Guards the authoring wiring directly: a unit_config on an apiv3.BillingRateCard
		// must survive into meta.UnitConfig. Without this, dropping the propagation in
		// FromAPIBillingRateCard would still leave the helper round-trip tests green.
		var price apiv3.BillingPrice
		require.NoError(t, price.FromBillingPriceUnit(apiv3.BillingPriceUnit{Amount: "0.05", Type: "unit"}))

		bc := apiv3.ISO8601Duration("P1M")
		rc := apiv3.BillingRateCard{
			Key:            "storage",
			Name:           "Storage",
			Price:          price,
			BillingCadence: &bc,
			UnitConfig: &apiv3.BillingUnitConfig{
				Operation:        apiv3.BillingUnitConfigOperationDivide,
				ConversionFactor: apiv3.Numeric("1000000000"),
				Rounding:         lo.ToPtr(apiv3.BillingUnitConfigRoundingModeCeiling),
				Precision:        lo.ToPtr(0),
				DisplayUnit:      lo.ToPtr("GB"),
			},
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		uc := result.AsMeta().UnitConfig
		require.NotNil(t, uc)
		assert.Equal(t, productcatalog.UnitConfigOperationDivide, uc.Operation)
		assert.Equal(t, float64(1_000_000_000), uc.ConversionFactor.InexactFloat64())
		assert.Equal(t, productcatalog.UnitConfigRoundingModeCeiling, uc.Rounding)
		assert.Equal(t, 0, uc.Precision)
		assert.Equal(t, lo.ToPtr("GB"), uc.DisplayUnit)
	})
}
