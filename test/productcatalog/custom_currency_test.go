package productcatalog_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type testEnv struct {
	*pctestutils.TestEnv
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	env := &testEnv{TestEnv: pctestutils.NewTestEnv(t)}
	t.Cleanup(func() { env.Close(t) })

	return env
}

func TestCustomCurrencyProductCatalogLifecycle(t *testing.T) {
	// given:
	// - a managed custom currency with a USD cost basis
	// - a USD plan and add-on whose matching rate cards are priced in that currency
	// when:
	// - both resources are created, updated from code-only inputs, assigned, published, fetched, and listed
	// then:
	// - every service and persistence boundary retains the original managed currency identity
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	env := newTestEnv(t)

	namespace := pctestutils.NewTestNamespace(t)
	customCode := currencyx.Code("CREDITS")
	originalCurrency, err := env.Currency.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		Code:      customCode.String(),
		Name:      "Credits",
		Symbol:    "cr",
	})
	require.NoError(t, err)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: originalCurrency.ID,
		FiatCode:   currency.USD.String(),
		Rate:       decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	planInput := pctestutils.NewTestPlan(
		t,
		namespace,
		pctestutils.WithPlanKey("custom-currency-plan"),
		pctestutils.WithPlanPhases(newCustomCurrencyPlanPhase(t, customCode, "Initial plan rate card")),
	)
	createdPlan, err := env.Plan.CreatePlan(t.Context(), planInput)
	require.NoError(t, err)
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *createdPlan)

	addonInput := pctestutils.NewTestAddon(t, namespace, newCustomCurrencyRateCard(t, customCode, "Initial add-on rate card"))
	addonInput.Key = "custom-currency-addon"
	createdAddon, err := env.Addon.CreateAddon(t.Context(), addonInput)
	require.NoError(t, err)
	env.requireAddonRateCardCurrencyID(t, originalCurrency.ID, *createdAddon)

	// Reusing a custom currency code must not retarget existing product catalog resources.
	err = env.Client.CustomCurrency.UpdateOneID(originalCurrency.ID).
		SetDeletedAt(now).
		Exec(t.Context())
	require.NoError(t, err)

	replacementCurrency, err := env.Currency.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		Code:      customCode.String(),
		Name:      "Replacement credits",
		Symbol:    "cr2",
	})
	require.NoError(t, err)
	require.NotEqual(t, originalCurrency.ID, replacementCurrency.ID)

	updatedPlanPhases := []productcatalog.Phase{newCustomCurrencyPlanPhase(t, customCode, "Updated plan rate card")}
	updatedPlan, err := env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
		NamespacedID: createdPlan.NamespacedID,
		Name:         lo.ToPtr("Updated custom currency plan"),
		Phases:       &updatedPlanPhases,
	})
	require.NoError(t, err)
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *updatedPlan)

	updatedAddonRateCards := productcatalog.RateCards{newCustomCurrencyRateCard(t, customCode, "Updated add-on rate card")}
	updatedAddon, err := env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
		NamespacedID: createdAddon.NamespacedID,
		Name:         lo.ToPtr("Updated custom currency add-on"),
		RateCards:    &updatedAddonRateCards,
	})
	require.NoError(t, err)
	env.requireAddonRateCardCurrencyID(t, originalCurrency.ID, *updatedAddon)

	publishedAddon, err := env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
		NamespacedID: updatedAddon.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(now.Add(-time.Second)),
		},
	})
	require.NoError(t, err)
	require.Equal(t, productcatalog.AddonStatusActive, publishedAddon.Status())

	assignment, err := env.PlanAddon.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		PlanID:          updatedPlan.ID,
		AddonID:         publishedAddon.ID,
		FromPlanPhase:   "default",
	})
	require.NoError(t, err)
	env.requirePlanAddonCurrencyIDs(t, originalCurrency.ID, *assignment)

	publishedPlan, err := env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: updatedPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(now.Add(time.Minute)),
		},
	})
	require.NoError(t, err)
	require.Equal(t, productcatalog.PlanStatusScheduled, publishedPlan.Status())

	storedPlan, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: publishedPlan.NamespacedID})
	require.NoError(t, err)
	require.Equal(t, "Updated custom currency plan", storedPlan.Name)
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *storedPlan)

	storedAddon, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: publishedAddon.NamespacedID})
	require.NoError(t, err)
	require.Equal(t, "Updated custom currency add-on", storedAddon.Name)
	env.requireAddonRateCardCurrencyID(t, originalCurrency.ID, *storedAddon)

	storedAssignment, err := env.PlanAddon.GetPlanAddon(t.Context(), planaddon.GetPlanAddonInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		ID:              assignment.ID,
	})
	require.NoError(t, err)
	env.requirePlanAddonCurrencyIDs(t, originalCurrency.ID, *storedAssignment)

	plans, err := env.Plan.ListPlans(t.Context(), plan.ListPlansInput{Namespaces: []string{namespace}})
	require.NoError(t, err)
	require.Len(t, plans.Items, 1)
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, plans.Items[0])

	addons, err := env.Addon.ListAddons(t.Context(), addon.ListAddonsInput{Namespaces: []string{namespace}})
	require.NoError(t, err)
	require.Len(t, addons.Items, 1)
	env.requireAddonRateCardCurrencyID(t, originalCurrency.ID, addons.Items[0])

	assignments, err := env.PlanAddon.ListPlanAddons(t.Context(), planaddon.ListPlanAddonsInput{
		Namespaces: []string{namespace},
		IDs:        []string{assignment.ID},
	})
	require.NoError(t, err)
	require.Len(t, assignments.Items, 1)
	env.requirePlanAddonCurrencyIDs(t, originalCurrency.ID, assignments.Items[0])
}

func TestCustomCurrencyPlanVersionLifecycle(t *testing.T) {
	// given:
	// - an active plan version priced in a managed custom currency
	// - the currency is archived and its code is reused by a different managed resource
	// when:
	// - the plan is cloned into a new version and the new version is published
	// then:
	// - both versions retain the original currency identity across the automatic version cutover
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	env := newTestEnv(t)
	namespace := pctestutils.NewTestNamespace(t)
	customCode := currencyx.Code("CREDITS")
	originalCurrency, err := env.Currency.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		Code:      customCode.String(),
		Name:      "Credits",
		Symbol:    "cr",
	})
	require.NoError(t, err)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: originalCurrency.ID,
		FiatCode:   currency.USD.String(),
		Rate:       decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	planInput := pctestutils.NewTestPlan(
		t,
		namespace,
		pctestutils.WithPlanKey("versioned-custom-currency-plan"),
		pctestutils.WithPlanPhases(newCustomCurrencyPlanPhase(t, customCode, "Version one")),
	)
	createdPlan, err := env.Plan.CreatePlan(t.Context(), planInput)
	require.NoError(t, err)

	versionOne, err := env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: createdPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(now.Add(-time.Second)),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, versionOne.Version)
	require.Equal(t, productcatalog.PlanStatusActive, versionOne.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionOne)

	err = env.Client.CustomCurrency.UpdateOneID(originalCurrency.ID).
		SetDeletedAt(now).
		Exec(t.Context())
	require.NoError(t, err)

	replacementCurrency, err := env.Currency.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		Code:      customCode.String(),
		Name:      "Replacement credits",
		Symbol:    "cr2",
	})
	require.NoError(t, err)
	require.NotEqual(t, originalCurrency.ID, replacementCurrency.ID)

	versionTwoDraft, err := env.Plan.NextPlan(t.Context(), plan.NextPlanInput{
		NamespacedID: versionOne.NamespacedID,
	})
	require.NoError(t, err)
	require.Equal(t, 2, versionTwoDraft.Version)
	require.Equal(t, productcatalog.PlanStatusDraft, versionTwoDraft.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionTwoDraft)

	cutover := now.Add(time.Hour)
	versionTwo, err := env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: versionTwoDraft.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(cutover),
		},
	})
	require.NoError(t, err)
	require.Equal(t, productcatalog.PlanStatusScheduled, versionTwo.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionTwo)

	versionOne, err = env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: versionOne.NamespacedID})
	require.NoError(t, err)
	require.NotNil(t, versionOne.EffectiveTo)
	require.WithinDuration(t, cutover, *versionOne.EffectiveTo, 0)
	require.Equal(t, productcatalog.PlanStatusActive, versionOne.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionOne)

	clock.FreezeTime(cutover.Add(time.Second))

	versions, err := env.Plan.ListPlans(t.Context(), plan.ListPlansInput{
		Namespaces: []string{namespace},
		Keys:       []string{versionOne.Key},
		OrderBy:    plan.OrderByVersion,
		Order:      plan.OrderAsc,
	})
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	require.Equal(t, productcatalog.PlanStatusArchived, versions.Items[0].Status())
	require.Equal(t, productcatalog.PlanStatusActive, versions.Items[1].Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, versions.Items[0])
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, versions.Items[1])
}

func newCustomCurrencyPlanPhase(t *testing.T, code currencyx.Code, description string) productcatalog.Phase {
	t.Helper()

	return productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
		RateCards: productcatalog.RateCards{newCustomCurrencyRateCard(t, code, description)},
	}
}

func newCustomCurrencyRateCard(t *testing.T, code currencyx.Code, description string) productcatalog.RateCard {
	t.Helper()

	month := datetime.MustParseDuration(t, "P1M")

	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         "credits",
			Name:        "Credits",
			Description: lo.ToPtr(description),
			Currency:    code,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(25),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &month,
	}
}

func (e *testEnv) requirePlanRateCardCurrencyID(t *testing.T, expectedID string, value plan.Plan) {
	t.Helper()

	require.Len(t, value.Phases, 1)
	require.Len(t, value.Phases[0].RateCards, 1)
	e.requireManagedCurrencyID(t, expectedID, value.Phases[0].RateCards[0].AsMeta().Currency)
}

func (e *testEnv) requireAddonRateCardCurrencyID(t *testing.T, expectedID string, value addon.Addon) {
	t.Helper()

	require.Len(t, value.RateCards, 1)
	e.requireManagedCurrencyID(t, expectedID, value.RateCards[0].AsMeta().Currency)
}

func (e *testEnv) requirePlanAddonCurrencyIDs(t *testing.T, expectedID string, value planaddon.PlanAddon) {
	t.Helper()

	e.requirePlanRateCardCurrencyID(t, expectedID, value.Plan)
	e.requireAddonRateCardCurrencyID(t, expectedID, value.Addon)
}

func (e *testEnv) requireManagedCurrencyID(t *testing.T, expectedID string, identity currencyx.CurrencyIdentity) {
	t.Helper()

	require.NotNil(t, identity)
	managed, ok := identity.(currencyx.ManagedCurrency)
	require.True(t, ok, "custom currency must retain its managed resource identity")
	require.Equal(t, expectedID, managed.GetID())
}
