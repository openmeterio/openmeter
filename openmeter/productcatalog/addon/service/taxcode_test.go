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
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newTestAddonFlatRateCard(key string, tc *productcatalog.TaxCodeConfig) productcatalog.RateCard {
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

func newTestAddonInput(t *testing.T, namespace string, key string, rc productcatalog.RateCard) addon.CreateAddonInput {
	t.Helper()

	input := pctestutils.NewTestAddon(t, namespace, rc)
	input.Key = key
	input.Name = key

	return input
}

func firstAddonRateCardTaxConfig(t *testing.T, a *addon.Addon) *productcatalog.TaxCodeConfig {
	t.Helper()

	require.NotEmpty(t, a.RateCards)

	return a.RateCards[0].AsMeta().TaxConfig
}

func requireTaxCodeNotFound(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)
}

func TestAddonTaxCodeReferences(t *testing.T) {
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

			// when: the add-on is created
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "existing-tax-code", newTestAddonFlatRateCard("rc", want)))
			require.NoError(t, err)

			// then: the reference is returned and stored
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, a))

			fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: a.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, fetched))
		})

		t.Run("behavior only", func(t *testing.T) {
			// given: a rate card with a tax behavior but no Tax Code
			want := &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)}

			// when: the add-on is created
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "behavior-only", newTestAddonFlatRateCard("rc", want)))
			require.NoError(t, err)

			// then: the behavior is stored without a Tax Code
			fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: a.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, fetched))
		})

		t.Run("no tax config", func(t *testing.T) {
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "no-tax-config", newTestAddonFlatRateCard("rc", nil)))
			require.NoError(t, err)

			assert.Nil(t, firstAddonRateCardTaxConfig(t, a))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: a rate card referencing a Tax Code the namespace can't use
				input := newTestAddonInput(t, namespace, "create-"+ref.key, newTestAddonFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: ref.taxCodeID,
				}))

				// when: the add-on is created
				_, err := env.Addon.CreateAddon(t.Context(), input)

				// then: it is rejected as a validation error
				requireTaxCodeNotFound(t, err)
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("add existing tax code", func(t *testing.T) {
			// given: an add-on without tax config
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-add-tax-code", newTestAddonFlatRateCard("rc", nil)))
			require.NoError(t, err)

			// when: its rate card is updated to reference a live Tax Code
			want := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

			updated, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &productcatalog.RateCards{newTestAddonFlatRateCard("rc", want)},
			})
			require.NoError(t, err)

			// then: the reference is returned and stored
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, updated))

			fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: a.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, fetched))
		})

		t.Run("remove tax config", func(t *testing.T) {
			// given: an add-on referencing a Tax Code
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-remove-tax-code", newTestAddonFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card is updated without tax config
			updated, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &productcatalog.RateCards{newTestAddonFlatRateCard("rc", nil)},
			})
			require.NoError(t, err)

			// then: the reference is gone
			assert.Nil(t, firstAddonRateCardTaxConfig(t, updated))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: an add-on referencing a live Tax Code
				original := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

				a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-"+ref.key, newTestAddonFlatRateCard("rc", original)))
				require.NoError(t, err)

				// when: the rate card is updated to a Tax Code the namespace can't use
				_, err = env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
					NamespacedID: a.NamespacedID,
					RateCards: &productcatalog.RateCards{newTestAddonFlatRateCard("rc", &productcatalog.TaxCodeConfig{
						TaxCodeID: ref.taxCodeID,
					})},
				})

				// then: it is rejected and the stored reference is unchanged
				requireTaxCodeNotFound(t, err)

				fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: a.NamespacedID})
				require.NoError(t, err)
				assert.Equal(t, original, firstAddonRateCardTaxConfig(t, fetched))
			})
		}

		t.Run("metadata-only update keeps a reference to a since-deleted tax code", func(t *testing.T) {
			// given: an add-on referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)
			want := &productcatalog.TaxCodeConfig{TaxCodeID: doomed.ID}

			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-metadata-deleted-tax-code", newTestAddonFlatRateCard("rc", want)))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: only the name is updated
			updated, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				Name:         lo.ToPtr("Renamed"),
			})

			// then: rate cards aren't re-validated, so the existing reference survives
			require.NoError(t, err)
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, updated))
		})
	})
}

func TestAddonWithPlanTaxCode(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)
	taxCode := env.TaxCodeEnv.CreateTaxCode(t, namespace)

	// given: a published add-on attached to a plan whose rate card references a Tax Code
	a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "addon-with-tax-plan", newTestAddonFlatRateCard("rc", nil)))
	require.NoError(t, err)

	publishAt := time.Now().Truncate(time.Microsecond)
	a, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
		NamespacedID:    a.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: &publishAt},
	})
	require.NoError(t, err)

	want := &productcatalog.TaxCodeConfig{
		TaxCodeID: taxCode.ID,
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}

	p, err := env.Plan.CreatePlan(t.Context(), pctestutils.NewTestPlan(t, namespace,
		pctestutils.WithPlanKey("plan-with-tax-code"),
		pctestutils.WithPlanPhases(productcatalog.Phase{
			PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
			RateCards: productcatalog.RateCards{newTestAddonFlatRateCard("rc", want)},
		}),
	))
	require.NoError(t, err)

	_, err = env.PlanAddon.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		PlanID:          p.ID,
		AddonID:         a.ID,
		FromPlanPhase:   p.Phases[0].Key,
	})
	require.NoError(t, err)

	// when: the add-on is read with plans expanded
	fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{
		NamespacedID: a.NamespacedID,
		Expand:       addon.ExpandFields{PlanAddons: true},
	})
	require.NoError(t, err)

	// then: the embedded plan rate card carries the Tax Code reference
	require.NotNil(t, fetched.Plans, "Plans must be expanded")
	require.Len(t, *fetched.Plans, 1)

	phases := (*fetched.Plans)[0].Phases
	require.Len(t, phases, 1)
	require.Len(t, phases[0].RateCards, 1)
	assert.Equal(t, want, phases[0].RateCards[0].AsMeta().TaxConfig)
}
