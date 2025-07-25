package addon

import (
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
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
			RateCard: &RateCard{
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
					AddonID: "01JDPHJMKKT1S3XF47V2AGMA6J",
				},
				RateCard: &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "feature-1",
						Name:        "RateCard 1",
						Description: lo.ToPtr("RateCard 1"),
						Metadata: map[string]string{
							"key": "value",
						},
						FeatureKey: lo.ToPtr("feature-1"),
						FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
					BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
				},
			},
		},
		{
			Name: "UsageBased",
			RateCard: &RateCard{
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
					AddonID: "01JDPHJMKKH4YDJTQY5F3EAHCF",
				},
				RateCard: &productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "feature-2",
						Name:        "RateCard 2",
						Description: lo.ToPtr("RateCard 2"),
						Metadata: map[string]string{
							"key": "value",
						},
						FeatureKey: lo.ToPtr("feature-2"),
						FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
						EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
							productcatalog.MeteredEntitlementTemplate{
								Metadata: map[string]string{
									"key": "value",
								},
								IsSoftLimit:             true,
								IssueAfterReset:         lo.ToPtr(500.0),
								IssueAfterResetPriority: lo.ToPtr[uint8](1),
								PreserveOverageAtReset:  lo.ToPtr(true),
								UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
							}),
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						Price: productcatalog.NewPriceFrom(
							productcatalog.UnitPrice{
								Amount: decimal.NewFromInt(1000),
								Commitments: productcatalog.Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(10)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
								},
							}),
						Discounts: productcatalog.Discounts{
							Percentage: &productcatalog.PercentageDiscount{
								Percentage: models.NewPercentage(10),
							},
							Usage: &productcatalog.UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							},
						},
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.RateCard)
			require.NoErrorf(t, err, "serializing RateCard must not fail")

			t.Logf("Serialized RateCard: %s", string(b))

			var rc *RateCard
			err = json.Unmarshal(b, &rc)
			require.NoErrorf(t, err, "deserializing RateCard must not fail")

			assert.Equal(t, test.RateCard, rc)
		})
	}
}

func TestFlatFeeRateCard(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			RateCard      RateCard
			ExpectedError bool
		}{
			{
				Name: "valid",
				RateCard: RateCard{
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
						AddonID: "01JDPHJMKK2WFF1D8AD5SYB2P1",
					},
					RateCard: &productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "feat-1",
							Name:        "Flat 1",
							Description: lo.ToPtr("Flat 1"),
							Metadata: map[string]string{
								"name": "Flat 1",
							},
							FeatureKey: lo.ToPtr("feat-1"),
							FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
						BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
					},
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: RateCard{
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
						AddonID: "01JDPHJMKKZCTPZMD5SYDJENP3",
					},
					RateCard: &productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "feat-2",
							Name:        "Flat 2",
							Description: lo.ToPtr("Flat 2"),
							Metadata: map[string]string{
								"name": "Flat 2",
							},
							FeatureKey: lo.ToPtr("feat-2"),
							FeatureID:  lo.ToPtr("01JBP3SGZ2YTM6DVH2W318TPNH"),
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
									PaymentTerm: "invalid",
								}),
						},
						BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P0M")),
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
			RateCard      RateCard
			ExpectedError bool
		}{
			{
				Name: "valid",
				RateCard: RateCard{
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
						AddonID: "01JDPHJMKK9J7Z45XRM4J3DS72",
					},
					RateCard: &productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "feat-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							FeatureKey: lo.ToPtr("feat-1"),
							FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
								productcatalog.MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
								}),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: productcatalog.NewPriceFrom(
								productcatalog.UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: productcatalog.Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
						},
						BillingCadence: datetime.MustParseDuration(t, "P1M"),
					},
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: RateCard{
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
						AddonID: "01JDPHJMKKBZFWS90VX5BFFKPE",
					},
					RateCard: &productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:         "feat-2",
							Name:        "Usage 2",
							Description: lo.ToPtr("Usage 2"),
							Metadata: map[string]string{
								"name": "usage-2",
							},
							FeatureKey: lo.ToPtr("feat-2"),
							FeatureID:  lo.ToPtr("01JBWYR0G2PYB9DVADKQXF8E0P"),
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
								productcatalog.MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
								}),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "invalid_code",
								},
							},
							Price: productcatalog.NewPriceFrom(
								productcatalog.UnitPrice{
									Amount: decimal.NewFromInt(-1000),
									Commitments: productcatalog.Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(500)),
									},
								}),
						},
						BillingCadence: datetime.MustParseDuration(t, "P0M"),
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
