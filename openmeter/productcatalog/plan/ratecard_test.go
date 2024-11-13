package plan

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
					RateCardMeta: RateCardMeta{
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-1",
							ID:        "01JBP3SGZ2WVRN5N746HKJR67X",
						},
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Time{},
							UpdatedAt: time.Time{},
							DeletedAt: &time.Time{},
						},
						Key:         "flat-1",
						Type:        FlatFeeRateCardType,
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
						EntitlementTemplate: lo.ToPtr(NewEntitlementTemplateFrom(StaticEntitlementTemplate{
							Metadata: map[string]string{
								"name": "static-1",
							},
							Config: []byte(`"test"`),
						})),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						PhaseID: "",
					},
					BillingCadence: lo.ToPtr(datex.MustParse(t, "P1M")),
					Price: NewPriceFrom(FlatPrice{
						Amount:      decimal.NewFromInt(1000),
						PaymentTerm: InArrearsPaymentTerm,
					}),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-2",
							ID:        "01JBP3SGZ26YSACQ7HQRZHZAMH",
						},
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Time{},
							UpdatedAt: time.Time{},
							DeletedAt: &time.Time{},
						},
						Key:         "flat-2",
						Type:        FlatFeeRateCardType,
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
						EntitlementTemplate: lo.ToPtr(NewEntitlementTemplateFrom(StaticEntitlementTemplate{
							Metadata: map[string]string{
								"name": "static-1",
							},
							Config: []byte("invalid JSON"),
						})),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "invalid_code",
							},
						},
						PhaseID: "",
					},
					BillingCadence: lo.ToPtr(datex.MustParse(t, "P0M")),
					Price: NewPriceFrom(FlatPrice{
						Amount:      decimal.NewFromInt(-1000),
						PaymentTerm: PaymentTermType("invalid"),
					}),
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
					RateCardMeta: RateCardMeta{
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-1",
							ID:        "01JBP3SGZ2WVRN5N746HKJR67X",
						},
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Time{},
							UpdatedAt: time.Time{},
							DeletedAt: &time.Time{},
						},
						Key:         "usage-1",
						Type:        UsageBasedRateCardType,
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
						EntitlementTemplate: lo.ToPtr(NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "Entitlement 1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(500.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](1),
							PreserveOverageAtReset:  nil,
							UsagePeriod:             datex.MustParse(t, "P1M"),
						})),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						PhaseID: "",
					},
					BillingCadence: datex.MustParse(t, "P1M"),
					Price: lo.ToPtr(NewPriceFrom(UnitPrice{
						Amount:        decimal.NewFromInt(1000),
						MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
						MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
					})),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						NamespacedID: models.NamespacedID{
							Namespace: "namespace-2",
							ID:        "01JBWYPN0A4NJW0HCYVP47WQZ5",
						},
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Time{},
							UpdatedAt: time.Time{},
							DeletedAt: &time.Time{},
						},
						Key:         "usage-2",
						Type:        UsageBasedRateCardType,
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
						EntitlementTemplate: lo.ToPtr(NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "Entitlement 1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(500.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](1),
							PreserveOverageAtReset:  nil,
							UsagePeriod:             datex.MustParse(t, "P1M"),
						})),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "invalid_code",
							},
						},
						PhaseID: "",
					},
					BillingCadence: datex.MustParse(t, "P0M"),
					Price: lo.ToPtr(NewPriceFrom(UnitPrice{
						Amount:        decimal.NewFromInt(-1000),
						MinimumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
						MaximumAmount: lo.ToPtr(decimal.NewFromInt(500)),
					})),
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
