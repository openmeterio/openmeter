package plan

import (
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRateCard_JSON(t *testing.T) {
	tests := []struct {
		Name          string
		RateCard      productcatalog.RateCard
		ExpectedError bool
	}{
		{
			Name: "FlatFee",
			RateCard: &FlatFeeRateCard{
				RateCardManagedFields: RateCardManagedFields{
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
						UpdatedAt: time.Now().Add(-1 * time.Hour).UTC(),
						DeletedAt: lo.ToPtr(time.Now().UTC()),
					},
					NamespacedID: models.NamespacedID{
						Namespace: "namespace-1",
						ID:        "01JDPHJMKJ8SNYTK0GK88VD0E9",
					},
					PhaseID: "01JDPHJMKKT1S3XF47V2AGMA6J",
				},
				FlatFeeRateCard: productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "ratecard-1",
						Name:        "RateCard 1",
						Description: lo.ToPtr("RateCard 1"),
						Metadata: map[string]string{
							"key": "value",
						},
						Feature: &feature.Feature{
							ID:        "01JBP3SGZ20Y7VRVC351TDFXYZ",
							Name:      "Feature 1",
							Key:       "feature-1",
							MeterSlug: lo.ToPtr("meter-1"),
							MeterGroupByFilters: map[string]string{
								"key": "value",
							},
							Metadata: map[string]string{
								"key": "value",
							},
							CreatedAt:  time.Now().Add(-3 * time.Hour).UTC(),
							UpdatedAt:  time.Now().Add(-2 * time.Hour).UTC(),
							ArchivedAt: lo.ToPtr(time.Now().UTC()),
						},
						EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
							productcatalog.StaticEntitlementTemplate{
								Metadata: map[string]string{
									"key": "value",
								},
								Config: []byte(`{"key":"value"}`),
							}),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      decimal.NewFromInt(1000),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: lo.ToPtr(datex.MustParse(t, "P1M")),
				},
			},
		},
		{
			Name: "UsageBased",
			RateCard: &UsageBasedRateCard{
				RateCardManagedFields: RateCardManagedFields{
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
						UpdatedAt: time.Now().Add(-1 * time.Hour).UTC(),
						DeletedAt: lo.ToPtr(time.Now().UTC()),
					},
					NamespacedID: models.NamespacedID{
						Namespace: "namespace-2",
						ID:        "01JDPHJMKKHSBYD60YR9D26EST",
					},
					PhaseID: "01JDPHJMKKH4YDJTQY5F3EAHCF",
				},
				UsageBasedRateCard: productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "ratecard-2",
						Name:        "RateCard 2",
						Description: lo.ToPtr("RateCard 2"),
						Metadata: map[string]string{
							"key": "value",
						},
						Feature: &feature.Feature{
							ID:        "01JBP3SGZ20Y7VRVC351TDFXYZ",
							Name:      "Feature 2",
							Key:       "feature-2",
							MeterSlug: lo.ToPtr("meter-2"),
							MeterGroupByFilters: map[string]string{
								"key": "value",
							},
							Metadata: map[string]string{
								"key": "value",
							},
							CreatedAt:  time.Now().Add(-3 * time.Hour).UTC(),
							UpdatedAt:  time.Now().Add(-2 * time.Hour).UTC(),
							ArchivedAt: lo.ToPtr(time.Now().UTC()),
						},
						EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
							productcatalog.MeteredEntitlementTemplate{
								Metadata: map[string]string{
									"key": "value",
								},
								IsSoftLimit:             true,
								IssueAfterReset:         lo.ToPtr(500.0),
								IssueAfterResetPriority: lo.ToPtr[uint8](1),
								PreserveOverageAtReset:  lo.ToPtr(true),
								UsagePeriod:             datex.MustParse(t, "P1M"),
							}),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.UnitPrice{
								Amount:        decimal.NewFromInt(1000),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(10)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							}),
					},
					BillingCadence: datex.MustParse(t, "P1M"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.RateCard)
			require.NoErrorf(t, err, "serializing RateCard must not fail")

			t.Logf("Serialized RateCard: %s", string(b))

			rc, err := NewRateCardFrom(b)
			require.NoErrorf(t, err, "deserializing RateCard must not fail")

			assert.Equal(t, test.RateCard, rc)
		})
	}
}

func TestFlatFeeRateCard(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			RateCard      FlatFeeRateCard
			ExpectedError bool
		}{
			{
				Name: "valid",
				RateCard: FlatFeeRateCard{
					RateCardManagedFields: RateCardManagedFields{
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
							UpdatedAt: time.Now().Add(-1 * time.Hour).UTC(),
							DeletedAt: lo.ToPtr(time.Now().UTC()),
						},
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-1",
							ID:        "01JDPHJMKKBARD45QV203H97CE",
						},
						PhaseID: "01JDPHJMKK2WFF1D8AD5SYB2P1",
					},
					FlatFeeRateCard: productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "flat-1",
							Name:        "Flat 1",
							Description: lo.ToPtr("Flat 1"),
							Metadata: map[string]string{
								"name": "Flat 1",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-1",
								ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
								Name:                "Feature 1",
								Key:                 "feat-1",
								MeterSlug:           lo.ToPtr("meter-1"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 1",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
								productcatalog.StaticEntitlementTemplate{
									Metadata: map[string]string{
										"name": "static-1",
									},
									Config: []byte(`"test"`),
								}),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount:      decimal.NewFromInt(1000),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
						},
						BillingCadence: lo.ToPtr(datex.MustParse(t, "P1M")),
					},
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: FlatFeeRateCard{
					RateCardManagedFields: RateCardManagedFields{
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
							UpdatedAt: time.Now().Add(-1 * time.Hour).UTC(),
							DeletedAt: lo.ToPtr(time.Now().UTC()),
						},
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-2",
							ID:        "01JDPHJMKK6T8QBKQQQWGCCXYT",
						},
						PhaseID: "01JDPHJMKKZCTPZMD5SYDJENP3",
					},
					FlatFeeRateCard: productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "flat-2",
							Name:        "Flat 2",
							Description: lo.ToPtr("Flat 2"),
							Metadata: map[string]string{
								"name": "Flat 2",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-2",
								ID:                  "01JBP3SGZ2YTM6DVH2W318TPNH",
								Name:                "Feature 2",
								Key:                 "feat-2",
								MeterSlug:           lo.ToPtr("meter-2"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 2",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
								productcatalog.StaticEntitlementTemplate{
									Metadata: map[string]string{
										"name": "static-1",
									},
									Config: []byte("invalid JSON"),
								}),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "invalid_code",
								},
							},
							Price: productcatalog.NewPriceFrom(
								productcatalog.FlatPrice{
									Amount:      decimal.NewFromInt(-1000),
									PaymentTerm: productcatalog.PaymentTermType("invalid"),
								}),
						},
						BillingCadence: lo.ToPtr(datex.MustParse(t, "P0M")),
					},
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.RateCard.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestUsageBasedRateCard(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			RateCard      UsageBasedRateCard
			ExpectedError bool
		}{
			{
				Name: "valid",
				RateCard: UsageBasedRateCard{
					RateCardManagedFields: RateCardManagedFields{
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
							UpdatedAt: time.Now().Add(-1 * time.Hour).UTC(),
							DeletedAt: lo.ToPtr(time.Now().UTC()),
						},
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-1",
							ID:        "01JDPHJMKKK8MN7DNTEPS7BJ65",
						},
						PhaseID: "01JDPHJMKK9J7Z45XRM4J3DS72",
					},
					UsageBasedRateCard: productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "usage-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-1",
								ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
								Name:                "Feature 1",
								Key:                 "feat-1",
								MeterSlug:           lo.ToPtr("meter-1"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 1",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
								productcatalog.MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             datex.MustParse(t, "P1M"),
								}),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: productcatalog.NewPriceFrom(
								productcatalog.UnitPrice{
									Amount:        decimal.NewFromInt(1000),
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
								}),
						},
						BillingCadence: datex.MustParse(t, "P1M"),
					},
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: UsageBasedRateCard{
					RateCardManagedFields: RateCardManagedFields{
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
							UpdatedAt: time.Now().Add(-1 * time.Hour).UTC(),
							DeletedAt: lo.ToPtr(time.Now().UTC()),
						},
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-2",
							ID:        "01JDPHJMKK6RGN078EQEPHVJS2",
						},
						PhaseID: "01JDPHJMKKBZFWS90VX5BFFKPE",
					},
					UsageBasedRateCard: productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "usage-2",
							Name:        "Usage 2",
							Description: lo.ToPtr("Usage 2"),
							Metadata: map[string]string{
								"name": "usage-2",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-2",
								ID:                  "01JBWYR0G2PYB9DVADKQXF8E0P",
								Name:                "Feature 2",
								Key:                 "feat-2",
								MeterSlug:           lo.ToPtr("meter-2"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 2",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
								productcatalog.MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             datex.MustParse(t, "P1M"),
								}),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "invalid_code",
								},
							},
							Price: productcatalog.NewPriceFrom(
								productcatalog.UnitPrice{
									Amount:        decimal.NewFromInt(-1000),
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(500)),
								}),
						},
						BillingCadence: datex.MustParse(t, "P0M"),
					},
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.RateCard.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
