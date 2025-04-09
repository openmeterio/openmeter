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
	featureadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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

var featureInput = feature.CreateFeatureInputs{
	Name:      "Feature 1",
	Key:       "feature1",
	Namespace: namespace,
}

var addonV1Input = addon.CreateAddonInput{
	NamespacedModel: models.NamespacedModel{
		Namespace: namespace,
	},
	Addon: productcatalog.Addon{
		AddonMeta: productcatalog.AddonMeta{
			Key:          "addon1",
			Name:         "Addon v1",
			Description:  lo.ToPtr("Addon v1"),
			Metadata:     models.Metadata{"name": "addon1"},
			Annotations:  models.Annotations{"key": "value"},
			Currency:     currency.USD,
			InstanceType: productcatalog.AddonInstanceTypeSingle,
		},
		RateCards: nil,
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

	logger := testutils.NewDiscardLogger(t)

	featureRepo := featureadapter.NewPostgresFeatureRepo(entClient, logger)

	addonRepo := &adapter{
		db:     entClient,
		logger: logger,
	}

	t.Run("Addon", func(t *testing.T) {
		var addonV1 *addon.Addon

		feature1, err := featureRepo.CreateFeature(ctx, featureInput)
		require.NoErrorf(t, err, "creating feature must not fail")

		addonV1Input.RateCards = productcatalog.RateCards{
			&productcatalog.UsageBasedRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:                 feature1.Key,
					Name:                feature1.Name,
					Description:         lo.ToPtr(feature1.Name),
					Metadata:            models.Metadata{"name": feature1.Name},
					FeatureKey:          lo.ToPtr(feature1.Key),
					FeatureID:           lo.ToPtr(feature1.ID),
					EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
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
		}

		t.Run("Create", func(t *testing.T) {
			addonV1, err = addonRepo.CreateAddon(ctx, addonV1Input)
			require.NoErrorf(t, err, "creating new add-on must not fail")

			require.NotNilf(t, addonV1, "add-on must not be nil")

			addon.AssertAddonCreateInputEqual(t, addonV1Input, *addonV1)
		})

		t.Run("Get", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				getAddonV1, err := addonRepo.GetAddon(ctx, addon.GetAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        addonV1.ID,
					},
				})
				assert.NoErrorf(t, err, "getting add-on by id must not fail")

				require.NotNilf(t, addonV1, "add-on must not be nil")

				addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
			})

			t.Run("ByKey", func(t *testing.T) {
				getAddonV1, err := addonRepo.GetAddon(ctx, addon.GetAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:           addonV1Input.Key,
					IncludeLatest: true,
				})
				assert.NoErrorf(t, err, "getting add-on by key must not fail")

				require.NotNilf(t, getAddonV1, "add-on must not be nil")

				addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				getAddonV1, err := addonRepo.GetAddon(ctx, addon.GetAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:     addonV1Input.Key,
					Version: 1,
				})
				assert.NoErrorf(t, err, "getting plan by key and version must not fail")

				require.NotNilf(t, getAddonV1, "plan must not be nil")

				addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
			})
		})

		t.Run("List", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				listAddonV1, err := addonRepo.ListAddons(ctx, addon.ListAddonsInput{
					Namespaces: []string{namespace},
					IDs:        []string{addonV1.ID},
				})
				assert.NoErrorf(t, err, "listing add-on by id must not fail")

				require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

				addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
			})

			t.Run("ByKey", func(t *testing.T) {
				listAddonV1, err := addonRepo.ListAddons(ctx, addon.ListAddonsInput{
					Namespaces: []string{namespace},
					Keys:       []string{addonV1Input.Key},
				})
				assert.NoErrorf(t, err, "getting add-on by key must not fail")

				require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

				addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				listAddonV1, err := addonRepo.ListAddons(ctx, addon.ListAddonsInput{
					Namespaces:  []string{namespace},
					KeyVersions: map[string][]int{addonV1Input.Key: {1}},
				})
				assert.NoErrorf(t, err, "getting add-on by key and version must not fail")

				require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

				addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
			})
		})

		t.Run("Update", func(t *testing.T) {
			now := time.Now()

			addonV1Update := addon.UpdateAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        addonV1.ID,
				},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(now.UTC()),
					EffectiveTo:   lo.ToPtr(now.Add(30 * 24 * time.Hour).UTC()),
				},
				Name:        lo.ToPtr("Addon v1 Published"),
				Description: lo.ToPtr("Addon v1 Published"),
				Metadata: &models.Metadata{
					"name":        "Addon v1 Published",
					"description": "Addon v1 Published",
				},
				RateCards: &productcatalog.RateCards{
					&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:                 "ratecard-2",
							Name:                "RateCard 2",
							Description:         lo.ToPtr("RateCard 2"),
							Metadata:            models.Metadata{"name": "ratecard-2"},
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
						BillingCadence: &MonthPeriod,
					},
				},
			}

			addonV1, err = addonRepo.UpdateAddon(ctx, addonV1Update)
			require.NoErrorf(t, err, "updating add-on must not fail")

			require.NotNilf(t, addonV1, "add-on must not be nil")

			addon.AssertAddonUpdateInputEqual(t, addonV1Update, *addonV1)
		})

		t.Run("Delete", func(t *testing.T) {
			err = addonRepo.DeleteAddon(ctx, addon.DeleteAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: addonV1.Namespace,
					ID:        addonV1.ID,
				},
			})
			require.NoErrorf(t, err, "deleting ad-on must not fail")

			getAddonV1, err := addonRepo.GetAddon(ctx, addon.GetAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        addonV1.ID,
				},
			})
			require.NoErrorf(t, err, "getting add-on by id must not fail")

			require.NotNilf(t, getAddonV1, "add-on must not be nil")

			addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
		})

		t.Run("ListStatusFilter", func(t *testing.T) {
			testListPlanStatusFilter(ctx, t, addonRepo)
		})
	})
}

type createAddonVersionInput struct {
	Namespace       string
	Version         int
	EffectivePeriod productcatalog.EffectivePeriod
	Template        addon.CreateAddonInput
}

func createAddonVersion(ctx context.Context, repo *adapter, in createAddonVersionInput) error {
	createInput := in.Template
	createInput.Namespace = in.Namespace
	createInput.Addon.AddonMeta.Version = in.Version

	addonVersion, err := repo.CreateAddon(ctx, createInput)
	if err != nil {
		return err
	}

	_, err = repo.UpdateAddon(ctx, addon.UpdateAddonInput{
		NamespacedID: models.NamespacedID{
			Namespace: in.Namespace,
			ID:        addonVersion.ID,
		},
		EffectivePeriod: in.EffectivePeriod,
	})

	return err
}

func testListPlanStatusFilter(ctx context.Context, t *testing.T, repo *adapter) {
	defer clock.ResetTime()

	ns := "list-plan-status-filter"

	err := createAddonVersion(ctx, repo, createAddonVersionInput{
		Namespace: ns,
		Version:   1,
		Template:  addonV1Input,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T00:00:00Z")),
			EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T12:00:00Z")),
		},
	})
	require.NoError(t, err, "creating add-on version must not fail")

	err = createAddonVersion(ctx, repo, createAddonVersionInput{
		Namespace: ns,
		Version:   2,
		Template:  addonV1Input,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T12:00:00Z")),
		},
	})
	require.NoErrorf(t, err, "creating add-on version must not fail")

	err = createAddonVersion(ctx, repo, createAddonVersionInput{
		Namespace:       ns,
		Version:         3,
		Template:        addonV1Input,
		EffectivePeriod: productcatalog.EffectivePeriod{},
	})
	require.NoErrorf(t, err, "creating add-on version must not fail")

	tests := []struct {
		name          string
		at            time.Time
		filter        []productcatalog.AddonStatus
		expectVersion []int
	}{
		{
			name: "list latest active",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.AddonStatus{
				productcatalog.AddonStatusActive,
			},
			expectVersion: []int{2},
		},
		{
			name: "list latest draft",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.AddonStatus{
				productcatalog.AddonStatusDraft,
			},
			expectVersion: []int{3},
		},
		{
			name: "list latest archived",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.AddonStatus{
				productcatalog.AddonStatusArchived,
			},
			expectVersion: []int{1},
		},
		{
			name: "list all",
			at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
			filter: []productcatalog.AddonStatus{
				productcatalog.AddonStatusActive,
				productcatalog.AddonStatusDraft,
				productcatalog.AddonStatusArchived,
			},
			expectVersion: []int{1, 2, 3},
		},
		{
			name: "schedules in the future",
			at:   testutils.GetRFC3339Time(t, "2025-03-15T01:00:00Z"),
			filter: []productcatalog.AddonStatus{
				productcatalog.AddonStatusInvalid,
			},
			expectVersion: []int{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clock.SetTime(test.at)

			list, err := repo.ListAddons(ctx, addon.ListAddonsInput{
				Namespaces: []string{ns},
				Status:     test.filter,
			})
			require.NoError(t, err, "listing add-ons must not fail")

			versions := lo.Map(list.Items, func(item addon.Addon, _ int) int {
				return item.Version
			})

			require.ElementsMatch(t, test.expectVersion, versions)
		})
	}
}
