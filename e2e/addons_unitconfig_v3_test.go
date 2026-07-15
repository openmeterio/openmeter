package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// newMeteredFeature creates a SUM meter and a feature backed by it, returning
// the feature. Unit/usage-based rate cards require a feature, and the rate card
// key must equal the feature key on the v3 authoring path.
func newMeteredFeature(t *testing.T, c *v3Client) v3sdk.Feature {
	t.Helper()

	meterKey := uniqueKey("uc_meter")

	m, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
		Key:           meterKey,
		Name:          "Test Meter " + meterKey,
		Aggregation:   v3sdk.MeterAggregationSum,
		EventType:     uniqueKey("uc_event"),
		ValueProperty: lo.ToPtr("$.value"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, m)
	require.NotEmpty(t, m.ID)

	featureKey := uniqueKey("uc_feat")

	f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:   featureKey,
		Name:  "Test Feature " + featureKey,
		Meter: &v3sdk.FeatureMeterReferenceInput{ID: m.ID},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, f)
	require.NotEmpty(t, f.ID)

	return *f
}

// TestV3AddonUnitConfigAuthoring proves the v3 add-on rate card API can author,
// read back, and update a unit_config — the plan/add-on parity this ticket adds.
// Mirrors the Hurl round-trip (author → read → update → read) at the SDK layer.
func TestV3AddonUnitConfigAuthoring(t *testing.T) {
	c := newV3Client(t)
	f := newMeteredFeature(t, c)

	// Author: a unit-priced rate card carrying a full unit_config (bill GB:
	// divide bytes by 1e9, round up, label "GB").
	rc := validUnitRateCard(f)
	rc.UnitConfig = &v3sdk.UnitConfig{
		Operation:        v3sdk.UnitConfigOperationDivide,
		ConversionFactor: "1000000000",
		Rounding:         lo.ToPtr(v3sdk.UnitConfigRoundingModeCeiling),
		Precision:        lo.ToPtr(int64(0)),
		DisplayUnit:      lo.ToPtr("GB"),
	}

	body := validAddonRequest("uc_authoring")
	body.RateCards = []v3sdk.RateCardInput{rc}

	var addonID string

	t.Run("create surfaces the unit_config", func(t *testing.T) {
		addon, err := c.Addons.Create(t.Context(), body)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)
		require.Len(t, addon.RateCards, 1)

		uc := addon.RateCards[0].UnitConfig
		require.NotNil(t, uc, "unit_config dropped on create")
		assert.Equal(t, v3sdk.UnitConfigOperationDivide, uc.Operation)
		assert.Equal(t, v3sdk.Numeric("1000000000"), uc.ConversionFactor)
		require.NotNil(t, uc.Rounding)
		assert.Equal(t, v3sdk.UnitConfigRoundingModeCeiling, *uc.Rounding)
		assert.EqualValues(t, 0, lo.FromPtr(uc.Precision))
		assert.Equal(t, "GB", lo.FromPtr(uc.DisplayUnit))

		addonID = addon.ID
	})

	t.Run("get round-trips the stored unit_config verbatim", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		addon, err := c.Addons.Get(t.Context(), addonID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)
		require.Len(t, addon.RateCards, 1)

		uc := addon.RateCards[0].UnitConfig
		require.NotNil(t, uc)
		assert.Equal(t, v3sdk.UnitConfigOperationDivide, uc.Operation)
		assert.Equal(t, v3sdk.Numeric("1000000000"), uc.ConversionFactor)
		require.NotNil(t, uc.Rounding)
		assert.Equal(t, v3sdk.UnitConfigRoundingModeCeiling, *uc.Rounding)
		require.NotNil(t, uc.Precision)
		assert.EqualValues(t, 0, *uc.Precision)
		require.NotNil(t, uc.DisplayUnit)
		assert.Equal(t, "GB", *uc.DisplayUnit)
	})

	t.Run("update rewrites the unit_config", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		updated := validUnitRateCard(f)
		updated.UnitConfig = &v3sdk.UnitConfig{
			Operation:        v3sdk.UnitConfigOperationMultiply,
			ConversionFactor: "1000000",
			Rounding:         lo.ToPtr(v3sdk.UnitConfigRoundingModeHalfUp),
			Precision:        lo.ToPtr(int64(2)),
			DisplayUnit:      lo.ToPtr("M"),
		}

		addon, err := c.Addons.Update(t.Context(), addonID, v3sdk.UpsertAddonRequest{
			Name:         body.Name,
			InstanceType: body.InstanceType,
			RateCards:    []v3sdk.RateCardInput{updated},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)
		require.Len(t, addon.RateCards, 1)

		uc := addon.RateCards[0].UnitConfig
		require.NotNil(t, uc)
		assert.Equal(t, v3sdk.UnitConfigOperationMultiply, uc.Operation)
		assert.Equal(t, v3sdk.Numeric("1000000"), uc.ConversionFactor)
		require.NotNil(t, uc.Rounding)
		assert.Equal(t, v3sdk.UnitConfigRoundingModeHalfUp, *uc.Rounding)
		assert.EqualValues(t, 2, lo.FromPtr(uc.Precision))
		assert.Equal(t, "M", lo.FromPtr(uc.DisplayUnit))
	})

	t.Run("get reflects the updated unit_config", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		addon, err := c.Addons.Get(t.Context(), addonID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)
		require.Len(t, addon.RateCards, 1)

		uc := addon.RateCards[0].UnitConfig
		require.NotNil(t, uc)
		assert.Equal(t, v3sdk.UnitConfigOperationMultiply, uc.Operation)
		assert.Equal(t, v3sdk.Numeric("1000000"), uc.ConversionFactor)
		require.NotNil(t, uc.Rounding)
		assert.Equal(t, v3sdk.UnitConfigRoundingModeHalfUp, *uc.Rounding)
		require.NotNil(t, uc.Precision)
		assert.EqualValues(t, 2, *uc.Precision)
		require.NotNil(t, uc.DisplayUnit)
		assert.Equal(t, "M", *uc.DisplayUnit)
	})
}

// TestV3AddonV1PriceReadFallback proves the v1→v3 read fallback for add-ons: v3
// cannot author a package/dynamic price, so an add-on only carries one when
// authored through the v1 API. Reading such an add-on through the v3 API must
// render the price as a unit price paired with a synthesized unit_config that
// describes the conversion v1 applied implicitly — exactly as the v3 plan API
// already does. Two versions: package and dynamic.
func TestV3AddonV1PriceReadFallback(t *testing.T) {
	cases := []struct {
		name string
		// buildPrice returns the v1 usage-based price under test.
		buildPrice func(t *testing.T) api.RateCardUsageBasedPrice
		// expected v3-rendered unit price amount + synthesized unit_config.
		wantUnitAmount       string
		wantOperation        v3sdk.UnitConfigOperation
		wantConversionFactor v3sdk.Numeric
		wantRounding         *v3sdk.UnitConfigRoundingMode
	}{
		{
			name: "package price renders as unit + divide unit_config",
			buildPrice: func(t *testing.T) api.RateCardUsageBasedPrice {
				var p api.RateCardUsageBasedPrice
				require.NoError(t, p.FromPackagePriceWithCommitments(api.PackagePriceWithCommitments{
					Type:               api.PackagePriceWithCommitmentsTypePackage,
					Amount:             "10",
					QuantityPerPackage: "1000",
				}))
				return p
			},
			wantUnitAmount:       "10",
			wantOperation:        v3sdk.UnitConfigOperationDivide,
			wantConversionFactor: "1000",
			wantRounding:         lo.ToPtr(v3sdk.UnitConfigRoundingModeCeiling),
		},
		{
			name: "dynamic price renders as unit 1 + multiply unit_config",
			buildPrice: func(t *testing.T) api.RateCardUsageBasedPrice {
				var p api.RateCardUsageBasedPrice
				require.NoError(t, p.FromDynamicPriceWithCommitments(api.DynamicPriceWithCommitments{
					Type:       api.DynamicPriceWithCommitmentsTypeDynamic,
					Multiplier: lo.ToPtr(api.Numeric("1.2")),
				}))
				return p
			},
			wantUnitAmount:       "1",
			wantOperation:        v3sdk.UnitConfigOperationMultiply,
			wantConversionFactor: "1.2",
			wantRounding:         nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newV3Client(t)
			v1 := initClient(t)

			f := newMeteredFeature(t, c)

			// Author the add-on through the v1 API (camelCase; unions built via
			// FromX constructors). The package/dynamic price is v1's native shape;
			// no unit_config is stored.
			price := tc.buildPrice(t)

			var rc api.RateCard
			require.NoError(t, rc.FromRateCardUsageBased(api.RateCardUsageBased{
				Type:           api.RateCardUsageBasedTypeUsageBased,
				Key:            f.Key,
				Name:           "v1 fallback rc",
				FeatureKey:     &f.Key,
				BillingCadence: "P1M",
				Price:          &price,
			}))

			createResp, err := v1.CreateAddonWithResponse(t.Context(), api.AddonCreate{
				Name:         "v1-fallback-addon " + f.Key,
				Key:          uniqueKey("v1_fallback"),
				Currency:     "USD",
				InstanceType: api.AddonInstanceTypeSingle,
				RateCards:    []api.RateCard{rc},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, createResp.StatusCode(), "v1 create body: %s", string(createResp.Body))
			require.NotNil(t, createResp.JSON201)
			addonID := createResp.JSON201.Id
			require.NotEmpty(t, addonID)

			// Read the SAME add-on through the v3 API: the fallback must project the
			// v1 price to a unit price + synthesized unit_config.
			addon, err := c.Addons.Get(t.Context(), addonID)
			c.requireStatus(http.StatusOK, err)
			require.NotNil(t, addon)
			require.Len(t, addon.RateCards, 1)

			gotRC := addon.RateCards[0]

			unitPrice, err := gotRC.Price.AsPriceUnit()
			require.NoError(t, err, "v1 %s price should read back as a v3 unit price", tc.name)
			assert.Equal(t, tc.wantUnitAmount, unitPrice.Amount)

			uc := gotRC.UnitConfig
			require.NotNil(t, uc, "v1 price did not synthesize a unit_config on v3 read")
			assert.Equal(t, tc.wantOperation, uc.Operation)
			assert.Equal(t, tc.wantConversionFactor, uc.ConversionFactor)
			assert.Equal(t, tc.wantRounding, uc.Rounding)
		})
	}
}

// TestV3AddonUnitConfigRejectedByV1Read proves the OM-409 guard that add-on
// unit_config authoring unblocks: once an add-on can carry a stored unit_config
// (v3 authoring, above), the v1 add-on read surface — which cannot represent it
// — must reject it. Author + publish a unit_config add-on via v3, then GET it
// through the v1 API and expect a 400 unit_config_not_representable.
func TestV3AddonUnitConfigRejectedByV1Read(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	f := newMeteredFeature(t, c)

	rc := validUnitRateCard(f)
	rc.UnitConfig = &v3sdk.UnitConfig{
		Operation:        v3sdk.UnitConfigOperationDivide,
		ConversionFactor: "1000000000",
		Rounding:         lo.ToPtr(v3sdk.UnitConfigRoundingModeCeiling),
		DisplayUnit:      lo.ToPtr("GB"),
	}

	body := validAddonRequest("uc_v1_reject")
	body.RateCards = []v3sdk.RateCardInput{rc}

	addon, err := c.Addons.Create(t.Context(), body)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, addon)

	// A unit price is usage-based, so the unit_config price-type rule is satisfied
	// and publish succeeds — putting the add-on into active state for the v1 GET.
	published, err := c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, published)

	resp, err := v1.GetAddonWithResponse(t.Context(), addon.ID, &api.GetAddonParams{})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode(),
		"v1 add-on GET of a unit_config add-on must be rejected; body: %s", string(resp.Body))
	assert.Contains(t, string(resp.Body), "unit_config_not_representable")
}
