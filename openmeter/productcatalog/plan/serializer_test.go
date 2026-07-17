package plan

import (
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPlanSerialization(t *testing.T) {
	now := time.Now().UTC()
	duration := datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0) // P1M

	plan := Plan{
		NamespacedID: models.NamespacedID{
			Namespace: "test",
			ID:        "plan-1",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		PlanMeta: productcatalog.PlanMeta{
			Name:        "Test Plan",
			Description: lo.ToPtr("Test plan description"),
			Currency:    currencyx.Code("USD"),
			Metadata: models.Metadata{
				"key1": "value1",
			},
			BillingCadence: datetime.MustParseDuration(t, "P1M"),
			ProRatingConfig: productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
		},
		Phases: []Phase{
			{
				PhaseManagedFields: PhaseManagedFields{
					ManagedModel: models.ManagedModel{
						CreatedAt: now,
						UpdatedAt: now,
					},
					NamespacedID: models.NamespacedID{
						Namespace: "test",
						ID:        "phase-1",
					},
					PlanID: "plan-1",
				},
				Phase: productcatalog.Phase{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "phase-1",
						Name:        "Test Phase",
						Description: lo.ToPtr("Test phase description"),
						Metadata: models.Metadata{
							"key2": "value2",
						},
						Duration: &duration,
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:         "flat-fee-1",
								Name:        "Test Flat Fee",
								Description: lo.ToPtr("Test flat fee description"),
								Metadata: models.Metadata{
									"key3": "value3",
								},
								FeatureKey: lo.ToPtr("feature-1"),
								FeatureID:  lo.ToPtr("feature-1"),
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      decimal.NewFromInt(1000),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: &duration,
						},
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:         "usage-based-1",
								Name:        "Test Usage Based",
								Description: lo.ToPtr("Test usage based description"),
								Metadata: models.Metadata{
									"key5": "value5",
								},
								Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
									Mode: productcatalog.VolumeTieredPrice,
									Tiers: []productcatalog.PriceTier{
										{
											UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
											FlatPrice: &productcatalog.PriceTierFlatPrice{
												Amount: decimal.NewFromInt(1000),
											},
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: decimal.NewFromInt(5),
											},
										},
										{
											UpToAmount: nil,
											FlatPrice: &productcatalog.PriceTierFlatPrice{
												Amount: decimal.NewFromInt(1500),
											},
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: decimal.NewFromInt(1),
											},
										},
									},
									Commitments: productcatalog.Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
									},
								}),
							},
							BillingCadence: duration,
						},
					},
				},
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(plan)
	require.NoError(t, err)

	// Test unmarshaling
	var unmarshaled Plan
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Compare the original and unmarshaled plans
	assert.Equal(t, plan.NamespacedID, unmarshaled.NamespacedID)
	assert.Equal(t, plan.PlanMeta, unmarshaled.PlanMeta)
	assert.Equal(t, len(plan.Phases), len(unmarshaled.Phases))

	for i := range plan.Phases {
		assert.Equal(t, plan.Phases[i].PhaseManagedFields, unmarshaled.Phases[i].PhaseManagedFields)
		assert.Equal(t, plan.Phases[i].PhaseMeta, unmarshaled.Phases[i].PhaseMeta)
		assert.Equal(t, len(plan.Phases[i].RateCards), len(unmarshaled.Phases[i].RateCards))

		for j := range plan.Phases[i].RateCards {
			assert.Equal(t, plan.Phases[i].RateCards[j].Type(), unmarshaled.Phases[i].RateCards[j].Type())
			assert.Equal(t, plan.Phases[i].RateCards[j].Key(), unmarshaled.Phases[i].RateCards[j].Key())
			assert.Equal(t, plan.Phases[i].RateCards[j].AsMeta(), unmarshaled.Phases[i].RateCards[j].AsMeta())
		}
	}
}

func TestPlanSerializationUsesCurrencyCodes(t *testing.T) {
	// given:
	// - a plan and rate card backed by a managed custom currency
	managedCurrency := &currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: "test",
			ID:        "currency-resource-id",
		},
		Code: "CREDITS",
		Name: "Credits",
	}
	plan := Plan{
		PlanMeta: productcatalog.PlanMeta{
			Currency: managedCurrency,
		},
		Phases: []Phase{
			{
				Phase: productcatalog.Phase{
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Currency: managedCurrency,
							},
						},
					},
				},
			},
		},
	}

	// when:
	// - a plan containing resolved managed currency identities crosses the JSON event boundary
	data, err := json.Marshal(plan)
	require.NoError(t, err)

	// then:
	// - only stable currency codes are serialized, and decoding restores code identities
	var serialized struct {
		Currency currencyx.Code `json:"currency"`
		Phases   []struct {
			RateCards []struct {
				Currency currencyx.Code `json:"currency"`
			} `json:"rateCards"`
		} `json:"phases"`
	}
	require.NoError(t, json.Unmarshal(data, &serialized))
	assert.Equal(t, currencyx.Code("CREDITS"), serialized.Currency)
	require.Len(t, serialized.Phases, 1)
	require.Len(t, serialized.Phases[0].RateCards, 1)
	assert.Equal(t, currencyx.Code("CREDITS"), serialized.Phases[0].RateCards[0].Currency)
	assert.NotContains(t, string(data), managedCurrency.ID)

	var decoded Plan
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.IsType(t, currencyx.Code(""), decoded.Currency)
	assert.Equal(t, currencyx.Code("CREDITS"), decoded.Currency.GetCode())
	require.Len(t, decoded.Phases, 1)
	require.Len(t, decoded.Phases[0].RateCards, 1)
	assert.IsType(t, currencyx.Code(""), decoded.Phases[0].RateCards[0].AsMeta().Currency)
	assert.Equal(t, currencyx.Code("CREDITS"), decoded.Phases[0].RateCards[0].AsMeta().Currency.GetCode())
}

func TestPlanSerializationErrors(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantErr  bool
		errMatch string
	}{
		{
			name:     "invalid JSON",
			json:     `{`,
			wantErr:  true,
			errMatch: "unexpected end of JSON input",
		},
		{
			name: "invalid rate card type",
			json: `{
				"namespace": "test",
				"id": "plan-1",
				"name": "Test Plan",
				"phases": [{
					"key": "phase-1",
					"name": "Test Phase",
					"rateCards": [{
						"type": "invalid",
						"key": "rate-card-1",
						"name": "Test Rate Card"
					}]
				}]
			}`,
			wantErr:  true,
			errMatch: "unsupported rate card type: invalid",
		},
		{
			name: "invalid billing cadence",
			json: `{
				"namespace": "test",
				"id": "plan-1",
				"name": "Test Plan",
				"phases": [{
					"key": "phase-1",
					"name": "Test Phase",
					"rateCards": [{
						"type": "flat_fee",
						"key": "rate-card-1",
						"name": "Test Rate Card",
						"billingCadence": "invalid"
					}]
				}]
			}`,
			wantErr:  true,
			errMatch: "invalid billing cadence for rate card \"rate-card-1\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var plan Plan
			err := json.Unmarshal([]byte(tt.json), &plan)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMatch)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
