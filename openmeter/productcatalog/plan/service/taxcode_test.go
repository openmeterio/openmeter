package service_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newTestFlatRateCard(key string, tc *productcatalog.TaxCodeConfig) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:       key,
			Name:      key,
			TaxConfig: tc,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		BillingCadence: &pctestutils.MonthPeriod,
	}
}

func newTestPlanInput(t *testing.T, namespace string, key string, rc productcatalog.RateCard) plan.CreatePlanInput {
	t.Helper()

	return pctestutils.NewTestPlan(t, namespace,
		pctestutils.WithPlanKey(key),
		pctestutils.WithPlanPhases(productcatalog.Phase{
			PhaseMeta: productcatalog.PhaseMeta{
				Key:  "default",
				Name: "Default",
			},
			RateCards: productcatalog.RateCards{rc},
		}),
	)
}

func firstRateCardTaxConfig(t *testing.T, p *plan.Plan) *productcatalog.TaxCodeConfig {
	t.Helper()

	require.NotEmpty(t, p.Phases)
	require.NotEmpty(t, p.Phases[0].RateCards)

	return p.Phases[0].RateCards[0].AsMeta().TaxConfig
}

func requireTaxCodeNotFound(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)
}

func TestPlanTaxCodeReferences(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)

	taxCode := env.TaxCodeEnv.CreateTaxCode(t, namespace)

	otherNamespaceTaxCode := env.TaxCodeEnv.CreateTaxCode(t, pctestutils.NewTestNamespace(t))

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	env.TaxCodeEnv.ProvisionDefaultTaxCodes(t, namespace)

	deletedTaxCode := env.TaxCodeEnv.CreateTaxCode(t, namespace)
	require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: deletedTaxCode.NamespacedID,
	}))

	invalidReferences := []struct {
		name      string
		key       string
		taxCodeID string
	}{
		{name: "unknown", key: "unknown", taxCodeID: "01JNONEXISTENTTAXCODE00000"},
		{name: "from another namespace", key: "other-namespace", taxCodeID: otherNamespaceTaxCode.ID},
		{name: "soft-deleted", key: "deleted", taxCodeID: deletedTaxCode.ID},
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("existing tax code", func(t *testing.T) {
			// given: a rate card referencing a live Tax Code in the namespace
			want := &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			}

			// when: the plan is created
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "existing-tax-code", newTestFlatRateCard("rc", want)))
			require.NoError(t, err)

			// then: the reference is returned and stored
			assert.Equal(t, want, firstRateCardTaxConfig(t, p))

			fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: p.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstRateCardTaxConfig(t, fetched))
		})

		t.Run("behavior only", func(t *testing.T) {
			// given: a rate card with a tax behavior but no Tax Code
			want := &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)}

			// when: the plan is created
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "behavior-only", newTestFlatRateCard("rc", want)))
			require.NoError(t, err)

			// then: the behavior is stored without a Tax Code
			fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: p.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstRateCardTaxConfig(t, fetched))
		})

		t.Run("no tax config", func(t *testing.T) {
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "no-tax-config", newTestFlatRateCard("rc", nil)))
			require.NoError(t, err)

			assert.Nil(t, firstRateCardTaxConfig(t, p))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: a rate card referencing a Tax Code the namespace can't use
				input := newTestPlanInput(t, namespace, "create-"+ref.key, newTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: ref.taxCodeID,
				}))

				// when: the plan is created
				_, err := env.Plan.CreatePlan(t.Context(), input)

				// then: it is rejected as a validation error
				requireTaxCodeNotFound(t, err)
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		updatePhases := func(rc productcatalog.RateCard) *[]productcatalog.Phase {
			return &[]productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
				RateCards: productcatalog.RateCards{rc},
			}}
		}

		t.Run("add existing tax code", func(t *testing.T) {
			// given: a plan without tax config
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-add-tax-code", newTestFlatRateCard("rc", nil)))
			require.NoError(t, err)

			// when: its rate card is updated to reference a live Tax Code
			want := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       updatePhases(newTestFlatRateCard("rc", want)),
			})
			require.NoError(t, err)

			// then: the reference is returned and stored
			assert.Equal(t, want, firstRateCardTaxConfig(t, updated))

			fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: p.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstRateCardTaxConfig(t, fetched))
		})

		t.Run("remove tax config", func(t *testing.T) {
			// given: a plan referencing a Tax Code
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-remove-tax-code", newTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card is updated without tax config
			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       updatePhases(newTestFlatRateCard("rc", nil)),
			})
			require.NoError(t, err)

			// then: the reference is gone
			assert.Nil(t, firstRateCardTaxConfig(t, updated))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: a plan referencing a live Tax Code
				original := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

				p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-"+ref.key, newTestFlatRateCard("rc", original)))
				require.NoError(t, err)

				// when: the rate card is updated to a Tax Code the namespace can't use
				_, err = env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
					NamespacedID: p.NamespacedID,
					Phases: updatePhases(newTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
						TaxCodeID: ref.taxCodeID,
					})),
				})

				// then: it is rejected and the stored reference is unchanged
				requireTaxCodeNotFound(t, err)

				fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: p.NamespacedID})
				require.NoError(t, err)
				assert.Equal(t, original, firstRateCardTaxConfig(t, fetched))
			})
		}

		t.Run("metadata-only update keeps a reference to a since-deleted tax code", func(t *testing.T) {
			// given: a plan referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)
			want := &productcatalog.TaxCodeConfig{TaxCodeID: doomed.ID}

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-metadata-deleted-tax-code", newTestFlatRateCard("rc", want)))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: only the name is updated
			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Name:         lo.ToPtr("Renamed"),
			})

			// then: rate cards aren't re-validated, so the existing reference survives
			require.NoError(t, err)
			assert.Equal(t, want, firstRateCardTaxConfig(t, updated))
		})
	})
}

func TestPlanWithAddonTaxCode(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)
	taxCode := env.TaxCodeEnv.CreateTaxCode(t, namespace)

	// given: a published add-on whose rate card references a Tax Code, attached to a plan
	want := &productcatalog.TaxCodeConfig{
		TaxCodeID: taxCode.ID,
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}

	addonInput := pctestutils.NewTestAddon(t, namespace, newTestFlatRateCard("rc", want))
	addonInput.Key = "tax-addon-for-plan"
	addonInput.Name = "Tax Addon For Plan"

	a, err := env.Addon.CreateAddon(t.Context(), addonInput)
	require.NoError(t, err)

	publishAt := time.Now().Truncate(time.Microsecond)
	a, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
		NamespacedID:    a.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: &publishAt},
	})
	require.NoError(t, err)

	p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "plan-with-tax-addon", newTestFlatRateCard("rc", nil)))
	require.NoError(t, err)

	_, err = env.PlanAddon.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		PlanID:          p.ID,
		AddonID:         a.ID,
		FromPlanPhase:   p.Phases[0].Key,
	})
	require.NoError(t, err)

	// when: the plan is read with add-ons expanded
	fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{
		NamespacedID: p.NamespacedID,
		Expand:       plan.ExpandFields{PlanAddons: true},
	})
	require.NoError(t, err)

	// then: the embedded add-on rate card carries the Tax Code reference
	require.NotNil(t, fetched.Addons, "Addons must be expanded")
	require.Len(t, *fetched.Addons, 1)

	addonRateCards := (*fetched.Addons)[0].RateCards
	require.Len(t, addonRateCards, 1)
	assert.Equal(t, want, addonRateCards[0].AsMeta().TaxConfig)
}
