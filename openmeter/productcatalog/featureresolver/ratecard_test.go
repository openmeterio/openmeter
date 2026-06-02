package featureresolver_test

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func Test_ResolveFeaturesForRateCards(t *testing.T) {
	// Setup test environment
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := pctestutils.NewTestNamespace(t)

	// Setup meter repository
	err := env.Meter.ReplaceMeters(t.Context(), pctestutils.NewTestMeters(t, namespace))
	require.NoError(t, err, "replacing meters must not fail")

	result, err := env.Meter.ListMeters(t.Context(), meter.ListMetersParams{
		Page: pagination.Page{
			PageSize:   1000,
			PageNumber: 1,
		},
		Namespace: namespace,
	})
	require.NoErrorf(t, err, "listing meters must not fail")

	meters := result.Items
	require.NotEmptyf(t, meters, "list of Meters must not be empty")

	// Set a feature for each meter
	features := make([]feature.Feature, 0, len(meters))
	for _, m := range meters {
		input := pctestutils.NewTestFeatureFromMeter(t, &m)

		feat, err := env.Feature.CreateFeature(t.Context(), input)
		require.NoErrorf(t, err, "creating feature must not fail")
		require.NotNil(t, feat, "feature must not be empty")

		features = append(features, feat)
	}
	require.NotEmptyf(t, features, "list of Features must not be empty")
	require.Lenf(t, features, len(meters), "list of Features must have the same length as the list of Meters")

	MonthPeriod := datetime.MustParseDuration(t, "P1M")

	tests := []struct {
		name        string
		ratecards   *productcatalog.RateCards
		expectedErr error
	}{
		{
			name: "success",
			ratecards: &productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         features[0].Key,
						Name:        features[0].Name,
						Description: lo.ToPtr("RateCard 1"),
						Metadata:    models.Metadata{"name": features[0].Name},
						FeatureKey:  lo.ToPtr(features[0].Key),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.FlatPrice{
								Amount:      decimal.NewFromInt(0),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
					},
					BillingCadence: &MonthPeriod,
				},
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         features[1].Key,
						Name:        features[1].Name,
						Description: lo.ToPtr("RateCard 2"),
						Metadata:    models.Metadata{"name": features[1].Name},
						FeatureID:   lo.ToPtr(features[1].ID),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.TieredPrice{
								Mode: productcatalog.VolumeTieredPrice,
								Tiers: []productcatalog.PriceTier{
									{
										UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
										FlatPrice: &productcatalog.PriceTierFlatPrice{
											Amount: decimal.NewFromInt(100),
										},
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: decimal.NewFromInt(50),
										},
									},
									{
										UpToAmount: nil,
										FlatPrice: &productcatalog.PriceTierFlatPrice{
											Amount: decimal.NewFromInt(75),
										},
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: decimal.NewFromInt(25),
										},
									},
								},
								Commitments: productcatalog.Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
									MaximumAmount: nil,
								},
							}),
					},
					BillingCadence: MonthPeriod,
				},
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         features[2].Key,
						Name:        features[2].Name,
						Description: lo.ToPtr("RateCard 3"),
						Metadata:    models.Metadata{"name": features[2].Name},
						FeatureKey:  lo.ToPtr(features[2].Key),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.FlatPrice{
								Amount:      decimal.NewFromInt(0),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
					},
					BillingCadence: &MonthPeriod,
				},
			},
		},
		{
			name: "not found",
			ratecards: &productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "abracadabra",
						Name:        "abracadabra",
						Description: lo.ToPtr("RateCard 4"),
						Metadata:    models.Metadata{"name": "abracadabra"},
						FeatureKey:  lo.ToPtr("abracadabra"),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.FlatPrice{
								Amount:      decimal.NewFromInt(0),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
					},
					BillingCadence: &MonthPeriod,
				},
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "abracadabra-2",
						Name:        "abracadabra-2",
						Description: lo.ToPtr("RateCard 4"),
						Metadata:    models.Metadata{"name": "abracadabra-2"},
						FeatureID:   lo.ToPtr("abracadabra-2"),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.FlatPrice{
								Amount:      decimal.NewFromInt(0),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
					},
					BillingCadence: &MonthPeriod,
				},
			},
			expectedErr: productcatalog.ErrRateCardFeatureNotFound,
		},
		{
			name: "mismatch",
			ratecards: &productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         features[0].Key,
						Name:        features[0].Name,
						Description: lo.ToPtr("RateCard 4"),
						Metadata:    models.Metadata{"name": features[0].Name},
						FeatureKey:  lo.ToPtr(features[0].Key),
						FeatureID:   lo.ToPtr(features[1].ID),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.FlatPrice{
								Amount:      decimal.NewFromInt(0),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
					},
					BillingCadence: &MonthPeriod,
				},
			},
			expectedErr: productcatalog.ErrRateCardFeatureMismatch,
		},
		{
			name: "id is actually a key",
			ratecards: &productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:       features[0].Key,
						Name:      features[0].Name,
						FeatureID: lo.ToPtr(features[0].Key), // wrong slot
						Price:     productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromInt(0), PaymentTerm: productcatalog.InArrearsPaymentTerm}),
					},
					BillingCadence: &MonthPeriod,
				},
			},
			expectedErr: productcatalog.ErrRateCardFeatureMismatch,
		},
		{
			name: "key is actually an id",
			ratecards: &productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:        features[0].Key,
						Name:       features[0].Name,
						FeatureKey: lo.ToPtr(features[0].ID), // wrong slot
						Price:      productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromInt(0), PaymentTerm: productcatalog.InArrearsPaymentTerm}),
					},
					BillingCadence: &MonthPeriod,
				},
			},
			expectedErr: productcatalog.ErrRateCardFeatureMismatch,
		},
	}

	resolver, err := featureresolver.New(env.Feature)
	require.NoError(t, err, "creating feature resolver must not fail")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = featureresolver.ResolveFeaturesForRateCards(t.Context(), resolver, namespace, test.ratecards)
			if test.expectedErr != nil {
				require.Error(t, err, "expected error")
				assert.ErrorIsf(t, err, test.expectedErr, "expected error message")
			} else {
				for idx, rc := range *test.ratecards {
					assert.Equal(t, features[idx].ID, lo.FromPtr(rc.GetFeatureID()), "resolved feature id must be equal to the one we set")
					assert.Equal(t, features[idx].Key, lo.FromPtr(rc.GetFeatureKey()), "resolved feature key must be equal to the one we set")
				}
			}
		})
	}
}
