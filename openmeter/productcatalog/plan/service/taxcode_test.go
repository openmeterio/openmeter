package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

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

// newUpdatePhases wraps the rate card in a single replacement phase.
func newUpdatePhases(rc productcatalog.RateCard) *[]productcatalog.Phase {
	return &[]productcatalog.Phase{{
		PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
		RateCards: productcatalog.RateCards{rc},
	}}
}

// Draft updates may keep a deleted Tax Code the stored plan already holds; a deleted one the update
// brings in is rejected, and publishing a version that holds one is rejected as well.
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

			// when: the plan is created
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "existing-tax-code", pctestutils.NewTestFlatRateCard("rc", want)))
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
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "behavior-only", pctestutils.NewTestFlatRateCard("rc", want)))
			require.NoError(t, err)

			// then: the behavior is stored without a Tax Code
			fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: p.NamespacedID})
			require.NoError(t, err)
			assert.Equal(t, want, firstRateCardTaxConfig(t, fetched))
		})

		t.Run("no tax config", func(t *testing.T) {
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "no-tax-config", pctestutils.NewTestFlatRateCard("rc", nil)))
			require.NoError(t, err)

			assert.Nil(t, firstRateCardTaxConfig(t, p))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: a rate card referencing a Tax Code the namespace can't use
				input := newTestPlanInput(t, namespace, "create-"+ref.key, pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: ref.taxCodeID,
				}))

				// when: the plan is created
				_, err := env.Plan.CreatePlan(t.Context(), input)

				// then: it is rejected as a validation error
				pctestutils.RequireTaxCodeNotFound(t, err)
			})
		}

		t.Run("soft-deleted tax code", func(t *testing.T) {
			// given: a rate card referencing a soft-deleted Tax Code: creating the plan would
			// author the reference, so it is rejected with the deleted error
			input := newTestPlanInput(t, namespace, "create-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: deletedTaxCode.ID,
			}))

			// when: the plan is created
			_, err := env.Plan.CreatePlan(t.Context(), input)

			// then: it is rejected with the deleted error, not the not-found one
			pctestutils.RequireTaxCodeDeleted(t, err)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("add existing tax code", func(t *testing.T) {
			// given: a plan without tax config
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-add-tax-code", pctestutils.NewTestFlatRateCard("rc", nil)))
			require.NoError(t, err)

			// when: its rate card is updated to reference a live Tax Code
			want := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       newUpdatePhases(pctestutils.NewTestFlatRateCard("rc", want)),
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
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-remove-tax-code", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card is updated without tax config
			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       newUpdatePhases(pctestutils.NewTestFlatRateCard("rc", nil)),
			})
			require.NoError(t, err)

			// then: the reference is gone
			assert.Nil(t, firstRateCardTaxConfig(t, updated))
		})

		for _, ref := range invalidReferences {
			t.Run(ref.name+" tax code", func(t *testing.T) {
				// given: a plan referencing a live Tax Code
				original := &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}

				p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-"+ref.key, pctestutils.NewTestFlatRateCard("rc", original)))
				require.NoError(t, err)

				// when: the rate card is updated to a Tax Code the namespace can't use
				_, err = env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
					NamespacedID: p.NamespacedID,
					Phases: newUpdatePhases(pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
						TaxCodeID: ref.taxCodeID,
					})),
				})

				// then: it is rejected and the stored reference is unchanged
				pctestutils.RequireTaxCodeNotFound(t, err)

				fetched, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: p.NamespacedID})
				require.NoError(t, err)
				assert.Equal(t, original, firstRateCardTaxConfig(t, fetched))
			})
		}

		t.Run("bringing in a deleted tax code the stored version does not hold", func(t *testing.T) {
			// given: a plan referencing a live Tax Code
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-new-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the rate card switches to the deleted Tax Code
			_, err = env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases: newUpdatePhases(pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
					TaxCodeID: deletedTaxCode.ID,
				})),
			})

			// then: it is rejected with the deleted error
			pctestutils.RequireTaxCodeDeleted(t, err)
		})

		t.Run("moving and renaming a rate card holding a deleted tax code the stored version has", func(t *testing.T) {
			// given: a two-phase plan whose second phase references the Tax Code that gets
			// deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)
			want := &productcatalog.TaxCodeConfig{TaxCodeID: doomed.ID}

			twoMonths := datetime.MustParseDuration(t, "P2M")

			p, err := env.Plan.CreatePlan(t.Context(), pctestutils.NewTestPlan(t, namespace,
				pctestutils.WithPlanKey("update-move-deleted"),
				pctestutils.WithPlanPhases(
					productcatalog.Phase{
						PhaseMeta: productcatalog.PhaseMeta{Key: "phase-1", Name: "Phase 1", Duration: &twoMonths},
						RateCards: productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc-free", nil)},
					},
					productcatalog.Phase{
						PhaseMeta: productcatalog.PhaseMeta{Key: "phase-2", Name: "Phase 2"},
						RateCards: productcatalog.RateCards{pctestutils.NewTestFlatRateCard("rc-doomed", want)},
					},
				),
			))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: doomed.NamespacedID,
			}))

			// when: the rate card holding the deleted reference is renamed and moved to the
			// first phase
			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases: &[]productcatalog.Phase{
					{
						PhaseMeta: productcatalog.PhaseMeta{Key: "phase-1-renamed", Name: "Phase 1"},
						RateCards: productcatalog.RateCards{
							pctestutils.NewTestFlatRateCard("rc-free", nil),
							pctestutils.NewTestFlatRateCard("rc-doomed-renamed", want),
						},
					},
				},
			})

			// then: the reference is inherited from the stored version, so the update passes
			require.NoError(t, err)
			require.Len(t, updated.Phases, 1)
			assert.Equal(t, want, updated.Phases[0].RateCards[1].AsMeta().TaxConfig)
		})

		t.Run("metadata-only update keeps a reference to a since-deleted tax code", func(t *testing.T) {
			// given: a plan referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)
			want := &productcatalog.TaxCodeConfig{TaxCodeID: doomed.ID}

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-metadata-deleted-tax-code", pctestutils.NewTestFlatRateCard("rc", want)))
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

		t.Run("scheduled plan update rejects a deleted tax code the stored version holds", func(t *testing.T) {
			// given: a plan referencing a Tax Code, scheduled for later activation, whose Tax
			// Code gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-scheduled-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			publishAt := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
			scheduled, err := env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &publishAt,
				},
			})
			require.NoError(t, err)
			require.Equal(t, productcatalog.PlanStatusScheduled, scheduled.Status())

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: the scheduled plan's name is updated without touching rate cards
			_, err = env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Name:         lo.ToPtr("Renamed"),
			})

			// then: the update is rejected: a scheduled plan must stay publishable, and
			// publish rejects deleted references
			pctestutils.RequireTaxCodeDeleted(t, err)
		})

		t.Run("scheduled plan update replacing the deleted tax code passes", func(t *testing.T) {
			// given: a scheduled plan whose Tax Code gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-scheduled-replace", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			publishAt := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
			_, err = env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &publishAt,
				},
			})
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: the rate card is updated to a live Tax Code
			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       newUpdatePhases(pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID})),
			})

			// then: the update passes: the candidate is publishable again, and the reference
			// is replaced
			require.NoError(t, err)
			assert.Equal(t, taxCode.ID, firstRateCardTaxConfig(t, updated).TaxCodeID)
		})

		t.Run("metadata-only update of a draft keeps a deleted tax code", func(t *testing.T) {
			// given: a draft plan referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-draft-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: only the name is updated
			updated, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Name:         lo.ToPtr("Renamed"),
			})

			// then: the draft keeps the inherited reference; publish blocks it later
			require.NoError(t, err)
			assert.Equal(t, doomed.ID, firstRateCardTaxConfig(t, updated).TaxCodeID)
		})
	})

	t.Run("Publish", func(t *testing.T) {
		t.Run("tax code deleted after creation", func(t *testing.T) {
			// given: a draft plan referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "publish-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: the plan is published
			publishAt := time.Now().Truncate(time.Microsecond)
			_, err = env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &publishAt,
				},
			})

			// then: publish is the gate and names the rate card holding the deleted reference
			pctestutils.RequireTaxCodeDeleted(t, err)
			assert.Contains(t, err.Error(), "ratecard.key=rc")
			assert.Contains(t, err.Error(), "tax_code.id="+doomed.ID)
		})

		t.Run("live tax code passes the gate", func(t *testing.T) {
			// given: a draft plan referencing a live Tax Code
			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "publish-live", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: taxCode.ID,
			})))
			require.NoError(t, err)

			// when: the plan is published
			publishAt := time.Now().Truncate(time.Microsecond)
			_, err = env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &publishAt,
				},
			})

			// then: the live reference passes the gate
			require.NoError(t, err)
		})
	})

	t.Run("NextPlan", func(t *testing.T) {
		t.Run("draft from a published plan whose tax code was deleted after publishing", func(t *testing.T) {
			// given: a published plan referencing a Tax Code that gets deleted afterwards
			doomed := env.TaxCodeEnv.CreateTaxCode(t, namespace)

			p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "next-deleted", pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{
				TaxCodeID: doomed.ID,
			})))
			require.NoError(t, err)

			publishAt := time.Now().Truncate(time.Microsecond)
			published, err := env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &publishAt,
				},
			})
			require.NoError(t, err)

			require.NoError(t, env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{NamespacedID: doomed.NamespacedID}))

			// when: the next draft is created
			next, err := env.Plan.NextPlan(t.Context(), plan.NextPlanInput{NamespacedID: published.NamespacedID})

			// then: the draft is created with the inherited reference; publishing it is
			// rejected naming the rate card
			require.NoError(t, err)
			require.Equal(t, productcatalog.PlanStatusDraft, next.Status())
			assert.Equal(t, doomed.ID, firstRateCardTaxConfig(t, next).TaxCodeID)

			_, err = env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
				NamespacedID: next.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)),
				},
			})
			pctestutils.RequireTaxCodeDeleted(t, err)
			assert.Contains(t, err.Error(), "ratecard.key=rc")
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

	addonInput := pctestutils.NewTestAddon(t, namespace, pctestutils.NewTestFlatRateCard("rc", want))
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

	p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "plan-with-tax-addon", pctestutils.NewTestFlatRateCard("rc", nil)))
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
