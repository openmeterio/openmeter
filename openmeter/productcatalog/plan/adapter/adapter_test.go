package adapter_test

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestPostgresAdapter(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

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

	planPhases := []productcatalog.Phase{
		{
			PhaseMeta: productcatalog.PhaseMeta{
				Key:         "trial",
				Name:        "Trial",
				Description: lo.ToPtr("Trial phase"),
				Metadata:    models.Metadata{"name": "trial"},
				Duration:    &pctestutils.MonthPeriod,
			},
			RateCards: []productcatalog.RateCard{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:                 "trial-ratecard-1",
						Name:                "Trial RateCard 1",
						Description:         lo.ToPtr("Trial RateCard 1"),
						Metadata:            models.Metadata{"name": "trial-ratecard-1"},
						FeatureKey:          nil,
						FeatureID:           nil,
						EntitlementTemplate: nil,
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      decimal.NewFromInt(0),
							PaymentTerm: productcatalog.InArrearsPaymentTerm,
						}),
					},
					BillingCadence: &pctestutils.MonthPeriod,
				},
			},
		},
		{
			PhaseMeta: productcatalog.PhaseMeta{
				Key:         "pro",
				Name:        "Pro",
				Description: lo.ToPtr("Pro phase"),
				Metadata:    models.Metadata{"name": "pro"},
				Duration:    nil,
			},
			RateCards: []productcatalog.RateCard{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:                 features[0].Key,
						Name:                "Pro RateCard 1",
						Description:         lo.ToPtr("Pro RateCard 1"),
						Metadata:            models.Metadata{"name": features[0].Key},
						FeatureKey:          &features[0].Key,
						FeatureID:           nil,
						EntitlementTemplate: nil,
						TaxConfig: &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{
								Code: "txcd_10000000",
							},
						},
						Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
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
										Amount: decimal.NewFromInt(5),
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
					BillingCadence: pctestutils.MonthPeriod,
				},
			},
		},
	}

	planV1Input := pctestutils.NewTestPlan(t, namespace, pctestutils.WithPlanPhases(planPhases...))

	t.Run("Plan", func(t *testing.T) {
		var (
			err    error
			planV1 *plan.Plan
		)

		t.Run("Create", func(t *testing.T) {
			planV1, err = env.Plan.CreatePlan(t.Context(), planV1Input)
			require.NoErrorf(t, err, "creating new plan must not fail")

			require.NotNilf(t, planV1, "plan must not be nil")

			plan.AssertPlanCreateInputEqual(t, planV1Input, *planV1)
		})

		t.Run("CreateWithCreditOnlySettlement", func(t *testing.T) {
			input := pctestutils.NewTestPlan(t, namespace,
				pctestutils.WithPlanPhases(planPhases...),
				func(t *testing.T, p *productcatalog.Plan) {
					t.Helper()
					p.Key = "test-credit-only"
					p.SettlementMode = productcatalog.CreditOnlySettlementMode
				},
			)

			p, err := env.PlanRepository.CreatePlan(t.Context(), input)
			require.NoErrorf(t, err, "creating plan with credit_only settlement must not fail")
			require.NotNilf(t, p, "plan must not be nil")

			assert.Equalf(t, productcatalog.CreditOnlySettlementMode, p.SettlementMode,
				"settlement mode mismatch: expected=%s, actual=%s", productcatalog.CreditOnlySettlementMode, p.SettlementMode)

			// Verify persistence via GetPlan
			fetched, err := env.PlanRepository.GetPlan(t.Context(), plan.GetPlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        p.ID,
				},
			})
			require.NoErrorf(t, err, "getting plan by id must not fail")

			assert.Equalf(t, productcatalog.CreditOnlySettlementMode, fetched.SettlementMode,
				"persisted settlement mode mismatch: expected=%s, actual=%s", productcatalog.CreditOnlySettlementMode, fetched.SettlementMode)
		})

		t.Run("Get", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				getPlanV1, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        planV1.ID,
					},
				})
				assert.NoErrorf(t, err, "getting plan by id must not fail")

				require.NotNilf(t, planV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})

			t.Run("ByKey", func(t *testing.T) {
				getPlanV1, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:           planV1Input.Key,
					IncludeLatest: true,
				})
				assert.NoErrorf(t, err, "getting plan by key must not fail")

				require.NotNilf(t, getPlanV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				getPlanV1, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:     planV1Input.Key,
					Version: 1,
				})
				assert.NoErrorf(t, err, "getting plan by key and version must not fail")

				require.NotNilf(t, getPlanV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})

			t.Run("ByIdExpandAddon", func(t *testing.T) {
				getPlanV1, err := env.PlanRepository.GetPlan(t.Context(), plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        planV1.ID,
					},
					Expand: plan.ExpandFields{
						PlanAddons: true,
					},
				})
				assert.NoErrorf(t, err, "getting plan by id must not fail")

				require.NotNilf(t, planV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})
		})

		t.Run("List", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				listPlanV1, err := env.PlanRepository.ListPlans(t.Context(), plan.ListPlansInput{
					Namespaces: []string{namespace},
					IDs:        []string{planV1.ID},
				})
				assert.NoErrorf(t, err, "listing plan by id must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})

			t.Run("ByKey", func(t *testing.T) {
				listPlanV1, err := env.PlanRepository.ListPlans(t.Context(), plan.ListPlansInput{
					Namespaces: []string{namespace},
					Keys:       []string{planV1Input.Key},
				})
				assert.NoErrorf(t, err, "getting plan by key must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				listPlanV1, err := env.PlanRepository.ListPlans(t.Context(), plan.ListPlansInput{
					Namespaces:  []string{namespace},
					KeyVersions: map[string][]int{planV1Input.Key: {1}},
				})
				assert.NoErrorf(t, err, "getting plan by key and version must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})
		})

		t.Run("Update", func(t *testing.T) {
			now := time.Now()

			planV1Update := plan.UpdatePlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        planV1.ID,
				},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(now.UTC()),
					EffectiveTo:   lo.ToPtr(now.Add(30 * 24 * time.Hour).UTC()),
				},
				Name:        lo.ToPtr("Pro Published"),
				Description: lo.ToPtr("Pro Published"),
				Metadata: &models.Metadata{
					"name":        "Pro Published",
					"description": "Pro Published",
				},
				Phases: nil,
			}

			planV1, err = env.PlanRepository.UpdatePlan(t.Context(), planV1Update)
			require.NoErrorf(t, err, "updating plan must not fail")

			require.NotNilf(t, planV1, "plan must not be nil")

			plan.AssertPlanUpdateInputEqual(t, planV1Update, *planV1)
		})

		t.Run("UpdateSettlementMode", func(t *testing.T) {
			newMode := productcatalog.CreditOnlySettlementMode

			planV1, err = env.PlanRepository.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        planV1.ID,
				},
				SettlementMode: &newMode,
			})
			require.NoErrorf(t, err, "updating settlement mode must not fail")
			require.NotNilf(t, planV1, "plan must not be nil")

			assert.Equalf(t, productcatalog.CreditOnlySettlementMode, planV1.SettlementMode,
				"settlement mode mismatch: expected=%s, actual=%s", productcatalog.CreditOnlySettlementMode, planV1.SettlementMode)

			// Verify persistence
			fetched, err := env.PlanRepository.GetPlan(t.Context(), plan.GetPlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        planV1.ID,
				},
			})
			require.NoErrorf(t, err, "getting plan by id must not fail")

			assert.Equalf(t, productcatalog.CreditOnlySettlementMode, fetched.SettlementMode,
				"persisted settlement mode mismatch: expected=%s, actual=%s", productcatalog.CreditOnlySettlementMode, fetched.SettlementMode)
		})

		t.Run("Delete", func(t *testing.T) {
			err = env.PlanRepository.DeletePlan(t.Context(), plan.DeletePlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: planV1.Namespace,
					ID:        planV1.ID,
				},
			})
			require.NoErrorf(t, err, "deleting plan must not fail")

			getPlanV1, err := env.PlanRepository.GetPlan(t.Context(), plan.GetPlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        planV1.ID,
				},
			})
			require.NoErrorf(t, err, "getting plan by id must not fail")

			require.NotNilf(t, getPlanV1, "plan must not be nil")

			plan.AssertPlanEqual(t, *planV1, *getPlanV1)
		})

		t.Run("ListStatusFilter", func(t *testing.T) {
			testListPlanStatusFilter(t.Context(), t, env.PlanRepository)
		})
	})
}

type createPlanVersionInput struct {
	Namespace       string
	Version         int
	EffectivePeriod productcatalog.EffectivePeriod
	Template        plan.CreatePlanInput
}

func createPlanVersion(ctx context.Context, repo plan.Repository, in createPlanVersionInput) error {
	createInput := in.Template
	createInput.Namespace = in.Namespace
	createInput.Plan.PlanMeta.Version = in.Version

	planVersion, err := repo.CreatePlan(ctx, createInput)
	if err != nil {
		return err
	}

	_, err = repo.UpdatePlan(ctx, plan.UpdatePlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: in.Namespace,
			ID:        planVersion.ID,
		},
		EffectivePeriod: in.EffectivePeriod,
	})

	return err
}

func testListPlanStatusFilter(ctx context.Context, t *testing.T, repo plan.Repository) {
	defer clock.ResetTime()

	ns := "list-plan-status-filter"

	planV1Input := pctestutils.NewTestPlan(t, ns)

	err := createPlanVersion(ctx, repo, createPlanVersionInput{
		Namespace: ns,
		Version:   1,
		Template:  planV1Input,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T00:00:00Z")),
			EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T12:00:00Z")),
		},
	})
	require.NoError(t, err, "creating plan version must not fail")

	err = createPlanVersion(ctx, repo, createPlanVersionInput{
		Namespace: ns,
		Version:   2,
		Template:  planV1Input,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T12:00:00Z")),
		},
	})
	require.NoErrorf(t, err, "creating plan version must not fail")

	err = createPlanVersion(ctx, repo, createPlanVersionInput{
		Namespace:       ns,
		Version:         3,
		Template:        planV1Input,
		EffectivePeriod: productcatalog.EffectivePeriod{},
	})
	require.NoErrorf(t, err, "creating plan version must not fail")

	tcs := []struct {
		name          string
		at            time.Time
		filter        []productcatalog.PlanStatus
		expectVersion []int
	}{
		{
			name: "list latest active",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusActive,
			},
			expectVersion: []int{2},
		},
		{
			name: "list latest draft",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusDraft,
			},
			expectVersion: []int{3},
		},
		{
			name: "list latest archived",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusArchived,
			},
			expectVersion: []int{1},
		},
		{
			name: "list all",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusActive,
				productcatalog.PlanStatusDraft,
				productcatalog.PlanStatusArchived,
			},
			expectVersion: []int{1, 2, 3},
		},
		{
			name: "plan schedule to be actived in the future - active filter",
			at:   testutils.GetRFC3339Time(t, "2025-03-15T01:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusActive,
			},
			expectVersion: []int{1}, // 2 is not yet active
		},
		{
			name: "plan schedule to be actived in the future - draft filter",
			at:   testutils.GetRFC3339Time(t, "2025-03-15T01:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusDraft,
			},
			expectVersion: []int{3},
		},
		{
			name: "plan schedule to be actived in the future - scheduled filter",
			at:   testutils.GetRFC3339Time(t, "2025-03-15T01:00:00Z"),
			filter: []productcatalog.PlanStatus{
				productcatalog.PlanStatusScheduled,
			},
			expectVersion: []int{2},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			clock.SetTime(tc.at)

			list, err := repo.ListPlans(ctx, plan.ListPlansInput{
				Namespaces: []string{ns},
				Status:     tc.filter,
			})
			require.NoError(t, err, "listing plans must not fail")

			versions := lo.Map(list.Items, func(item plan.Plan, _ int) int {
				return item.Version
			})

			require.ElementsMatch(t, tc.expectVersion, versions)
		})
	}
}
