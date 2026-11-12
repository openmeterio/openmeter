package service_test

import (
	"testing"
	"time"

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
		{name: "unknown", key: "unknown", taxCodeID: "01JABSENTTAXC0DE0000000000"},
		{name: "from another namespace", key: "other-namespace", taxCodeID: otherNamespaceTaxCode.ID},
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("existing tax code", func(t *testing.T) {
			// given: a rate card referencing a live Tax Code in the namespace
			want := &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			}

			// when: the add-on is created
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "existing-tax-code", pctestutils.NewTestFlatRateCard("rc", want)))
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
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "behavior-only", pctestutils.NewTestFlatRateCard("rc", want)))
			require.NoError(t, err)

			// then: the behavior is stored without a Tax Code
			fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: a.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, fetched))
		})

		t.Run("no tax config", func(t *testing.T) {
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "no-tax-config", pctestutils.NewTestFlatRateCard("rc", nil)))
			require.NoError(t, err)

			assert.Nil(t, firstAddonRateCardTaxConfig(t, a))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: a rate card referencing a Tax Code the namespace can't use
				input := newTestAddonInput(t, namespace, "create-"+ref.key, pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: ref.taxCodeID,
				}))

				// when: the add-on is created
				_, err := env.Addon.CreateAddon(t.Context(), input)

				// then: it is rejected as a validation error
				pctestutils.RequireTaxCodeNotFound(t, err)
			})
		}

		t.Run("soft-deleted tax code", func(t *testing.T) {
			// given: a rate card referencing a soft-deleted Tax Code: creating the add-on would
			// author the reference, so it is rejected with the deleted error
			input := newTestAddonInput(t, namespace, "create-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: deletedTaxCode.ID,
			}))

			// when: the add-on is created
			_, err := env.Addon.CreateAddon(t.Context(), input)

			// then: it is rejected with the deleted error, not the not-found one
			pctestutils.RequireTaxCodeDeleted(t, err)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("add existing tax code", func(t *testing.T) {
			// given: an add-on without tax config
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-add-tax-code", pctestutils.NewTestFlatRateCard("rc", nil)))
			require.NoError(t, err)

			// when: its rate card is updated to reference a live Tax Code
			want := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

			updated, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc", want)},
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
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-remove-tax-code", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card is updated without tax config
			updated, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc", nil)},
			})
			require.NoError(t, err)

			// then: the reference is gone
			assert.Nil(t, firstAddonRateCardTaxConfig(t, updated))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: an add-on referencing a live Tax Code
				original := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

				a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-"+ref.key, pctestutils.NewTestFlatRateCard("rc", original)))
				require.NoError(t, err)

				// when: the rate card is updated to a Tax Code the namespace can't use
				_, err = env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
					NamespacedID: a.NamespacedID,
					RateCards: &productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
						TaxCodeID: ref.taxCodeID,
					})},
				})

				// then: it is rejected and the stored reference is unchanged
				pctestutils.RequireTaxCodeNotFound(t, err)

				fetched, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: a.NamespacedID})
				require.NoError(t, err)
				assert.Equal(t, original, firstAddonRateCardTaxConfig(t, fetched))
			})
		}

		t.Run("bringing in a deleted tax code the stored version does not hold", func(t *testing.T) {
			// given: an add-on referencing a live Tax Code
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-new-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card is updated to reference the deleted Tax Code
			_, err = env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards: &productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: deletedTaxCode.ID,
				})},
			})

			// then: the authored reference is rejected with the deleted error
			pctestutils.RequireTaxCodeDeleted(t, err)
		})

		t.Run("renaming a rate card holding a deleted tax code the stored version has", func(t *testing.T) {
			// given: an add-on referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)
			want := &productcatalog.TaxCodeConfig{TaxCodeID: doomed.ID}

			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-rename-deleted", pctestutils.NewTestFlatRateCard("rc-doomed", want)))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: doomed.NamespacedID,
			}))

			// when: the rate card holding the deleted reference is renamed
			updated, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc-doomed-renamed", want)},
			})

			// then: the reference is inherited from the stored version, so the update passes
			require.NoError(t, err)
			assert.Equal(t, want, firstAddonRateCardTaxConfig(t, updated))
		})

		t.Run("missing tax code", func(t *testing.T) {
			// given: an add-on referencing a live Tax Code
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-missing", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card is updated to reference an unknown Tax Code
			_, err = env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards: &productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: "01JABSENTTAXC0DE0000000000",
				})},
			})

			// then: the missing ID is rejected even though the stored version's IDs are allowed
			pctestutils.RequireTaxCodeNotFound(t, err)
		})

		t.Run("metadata-only update keeps a reference to a since-deleted tax code", func(t *testing.T) {
			// given: an add-on referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)
			want := &productcatalog.TaxCodeConfig{TaxCodeID: doomed.ID}

			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "update-metadata-deleted-tax-code", pctestutils.NewTestFlatRateCard("rc", want)))
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

	t.Run("Publish", func(t *testing.T) {
		t.Run("tax code deleted after creation", func(t *testing.T) {
			// given: a draft add-on referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "publish-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: the add-on is published
			publishAt := time.Now().Truncate(time.Microsecond)
			_, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
				NamespacedID:    a.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: &publishAt},
			})

			// then: publish is the gate and names the rate card holding the deleted reference
			pctestutils.RequireTaxCodeDeleted(t, err)
			assert.Contains(t, err.Error(), "ratecard.key=rc")
			assert.Contains(t, err.Error(), "tax_code.id="+doomed.ID)
		})

		t.Run("live tax code passes the gate", func(t *testing.T) {
			// given: a draft add-on referencing a live Tax Code
			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "publish-live", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the add-on is published
			publishAt := time.Now().Truncate(time.Microsecond)
			_, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
				NamespacedID:    a.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: &publishAt},
			})

			// then: the live reference passes the gate
			require.NoError(t, err)
		})
	})

	t.Run("NextAddon", func(t *testing.T) {
		t.Run("draft from a published add-on whose tax code was deleted after publishing", func(t *testing.T) {
			// given: a published add-on referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "next-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			publishAt := time.Now().Truncate(time.Microsecond)
			published, err := env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
				NamespacedID:    a.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: &publishAt},
			})
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: the next draft is created
			next, err := env.Addon.NextAddon(t.Context(), addon.NextAddonInput{
				NamespacedID: models.NamespacedID{Namespace: published.Namespace},
				Key:          published.Key,
			})

			// then: the draft is created with the inherited reference; publishing it is
			// rejected naming the rate card
			require.NoError(t, err)
			require.Equal(t, productcatalog.AddonStatusDraft, next.Status())
			assert.Equal(t, doomed.ID, firstAddonRateCardTaxConfig(t, next).TaxCodeID)

			_, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
				NamespacedID:    next.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(time.Now().Add(24 * time.Hour).Truncate(time.Microsecond))},
			})
			pctestutils.RequireTaxCodeDeleted(t, err)
			assert.Contains(t, err.Error(), "ratecard.key=rc")
		})
	})
}

func TestAddonWithPlanTaxCode(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)
	taxCode := env.TaxCodeEnv.CreateTaxCode(t, namespace)

	// given: a published add-on attached to a plan whose rate card references a Tax Code
	a, err := env.Addon.CreateAddon(t.Context(), newTestAddonInput(t, namespace, "addon-with-tax-plan", pctestutils.NewTestFlatRateCard("rc", nil)))
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
			RateCards: productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc", want)},
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
