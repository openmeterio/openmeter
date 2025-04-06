package adapter

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	MonthPeriod      = isodate.FromDuration(30 * 24 * time.Hour)
	TwoMonthPeriod   = isodate.FromDuration(60 * 24 * time.Hour)
	ThreeMonthPeriod = isodate.FromDuration(90 * 24 * time.Hour)
)

var namespace = "01JBX0P4GQZCQY1WNGX3VT94P4"

var planV1Input = plan.CreatePlanInput{
	NamespacedModel: models.NamespacedModel{
		Namespace: namespace,
	},
	Plan: productcatalog.Plan{
		PlanMeta: productcatalog.PlanMeta{
			Key:         "pro",
			Name:        "Pro",
			Description: lo.ToPtr("Pro plan v1"),
			Metadata:    models.Metadata{"name": "pro"},
			Currency:    currency.USD,
		},
		Phases: []productcatalog.Phase{
			{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:         "trial",
					Name:        "Trial",
					Description: lo.ToPtr("Trial phase"),
					Metadata:    models.Metadata{"name": "trial"},
					Duration:    &MonthPeriod,
				},
				RateCards: []productcatalog.RateCard{
					&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:                 "trial-ratecard-1",
							Name:                "Trial RateCard 1",
							Description:         lo.ToPtr("Trial RateCard 1"),
							Metadata:            models.Metadata{"name": "trial-ratecard-1"},
							Feature:             nil,
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
						BillingCadence: &MonthPeriod,
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
							Key:                 "pro-ratecard-1",
							Name:                "Pro RateCard 1",
							Description:         lo.ToPtr("Pro RateCard 1"),
							Metadata:            models.Metadata{"name": "pro-ratecard-1"},
							Feature:             nil,
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
						BillingCadence: MonthPeriod,
					},
				},
			},
		},
	},
}

func TestPostgresAdapter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pg := testutils.InitPostgresDB(t)
	defer func() {
		if err := pg.EntDriver.Close(); err != nil {
			t.Errorf("failed to close ent driver: %v", err)
		}

		if err := pg.PGDriver.Close(); err != nil {
			t.Errorf("failed to postgres driver: %v", err)
		}
	}()

	err := pg.EntDriver.Client().Schema.Create(context.Background())
	require.NoErrorf(t, err, "schema migration must not fail")

	entClient := pg.EntDriver.Client()
	defer func() {
		if err = entClient.Close(); err != nil {
			t.Errorf("failed to close ent client: %v", err)
		}
	}()

	repo := &adapter{
		db:     pg.EntDriver.Client(),
		logger: testutils.NewDiscardLogger(t),
	}

	t.Run("Plan", func(t *testing.T) {
		var planV1 *plan.Plan

		t.Run("Create", func(t *testing.T) {
			planV1, err = repo.CreatePlan(ctx, planV1Input)
			require.NoErrorf(t, err, "creating new plan must not fail")

			require.NotNilf(t, planV1, "plan must not be nil")

			plan.AssertPlanCreateInputEqual(t, planV1Input, *planV1)
		})

		t.Run("Get", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
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
				getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
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
				getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
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
		})

		t.Run("List", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				listPlanV1, err := repo.ListPlans(ctx, plan.ListPlansInput{
					Namespaces: []string{namespace},
					IDs:        []string{planV1.ID},
				})
				assert.NoErrorf(t, err, "listing plan by id must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})

			t.Run("ByKey", func(t *testing.T) {
				listPlanV1, err := repo.ListPlans(ctx, plan.ListPlansInput{
					Namespaces: []string{namespace},
					Keys:       []string{planV1Input.Key},
				})
				assert.NoErrorf(t, err, "getting plan by key must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				listPlanV1, err := repo.ListPlans(ctx, plan.ListPlansInput{
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

			planV1, err = repo.UpdatePlan(ctx, planV1Update)
			require.NoErrorf(t, err, "updating plan must not fail")

			require.NotNilf(t, planV1, "plan must not be nil")

			plan.AssertPlanUpdateInputEqual(t, planV1Update, *planV1)
		})

		t.Run("Delete", func(t *testing.T) {
			err = repo.DeletePlan(ctx, plan.DeletePlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: planV1.Namespace,
					ID:        planV1.ID,
				},
			})
			require.NoErrorf(t, err, "deleting plan must not fail")

			getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
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
			testListPlanStatusFilter(ctx, t, repo)
		})
	})
}

type createPlanVersionInput struct {
	Namespace       string
	Version         int
	EffectivePeriod productcatalog.EffectivePeriod
	Template        plan.CreatePlanInput
}

func createPlanVersion(ctx context.Context, repo *adapter, in createPlanVersionInput) error {
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

func testListPlanStatusFilter(ctx context.Context, t *testing.T, repo *adapter) {
	defer clock.ResetTime()

	ns := "list-plan-status-filter"

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
