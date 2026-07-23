package productcatalog_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
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

func TestArchivedCustomCurrencyCannotRetargetProductCatalogResources(t *testing.T) {
	// given:
	// - a plan and add-on whose rate cards reference a managed custom currency
	// - that currency is archived and its code is reused by another valid currency
	// when:
	// - the existing resources are updated from code-only inputs
	// then:
	// - validation rejects the archived currency and the stored references retain their original identity
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	env := newTestEnv(t)

	namespace := pctestutils.NewTestNamespace(t)
	customCode := currencyx.Code("CREDITS")
	originalCurrency, err := env.Currency.CreateCurrency(t.Context(), currencytestutils.NewCreateCurrencyInput(namespace, customCode, "Credits", "cr"))
	require.NoError(t, err)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: originalCurrency.ID,
		FiatCode:   currencyx.Code(currency.USD),
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

	replacementCurrency, err := env.Currency.CreateCurrency(t.Context(), currencytestutils.NewCreateCurrencyInput(namespace, customCode, "Replacement credits", "cr2"))
	require.NoError(t, err)
	require.NotEqual(t, originalCurrency.ID, replacementCurrency.ID)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: replacementCurrency.ID,
		FiatCode:   currencyx.Code(currency.USD),
		Rate:       decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	updatedPlanPhases := []productcatalog.Phase{newCustomCurrencyPlanPhase(t, customCode, "Updated plan rate card")}
	_, err = env.Plan.UpdatePlan(t.Context(), plan.UpdatePlanInput{
		NamespacedID: createdPlan.NamespacedID,
		Name:         lo.ToPtr("Updated custom currency plan"),
		Phases:       &updatedPlanPhases,
	})
	require.ErrorContains(t, err, productcatalog.ErrCurrencyNotFound.Error())

	updatedAddonRateCards := productcatalog.RateCards{newCustomCurrencyRateCard(t, customCode, "Updated add-on rate card")}
	_, err = env.Addon.UpdateAddon(t.Context(), addon.UpdateAddonInput{
		NamespacedID: createdAddon.NamespacedID,
		Name:         lo.ToPtr("Updated custom currency add-on"),
		RateCards:    &updatedAddonRateCards,
	})
	require.ErrorContains(t, err, productcatalog.ErrCurrencyNotFound.Error())

	storedPlan, err := env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: createdPlan.NamespacedID})
	require.NoError(t, err)
	require.Equal(t, planInput.Name, storedPlan.Name)
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *storedPlan)

	storedAddon, err := env.Addon.GetAddon(t.Context(), addon.GetAddonInput{NamespacedID: createdAddon.NamespacedID})
	require.NoError(t, err)
	require.Equal(t, addonInput.Name, storedAddon.Name)
	env.requireAddonRateCardCurrencyID(t, originalCurrency.ID, *storedAddon)
}

func TestArchivedCustomCurrencyPreventsPlanVersionPublication(t *testing.T) {
	// given:
	// - an active plan version priced in a managed custom currency
	// - the currency is archived and its code is reused by a different managed resource
	// when:
	// - the plan is cloned into a new version and publication is attempted
	// then:
	// - publication is rejected and both versions retain the original currency identity
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	env := newTestEnv(t)
	namespace := pctestutils.NewTestNamespace(t)
	customCode := currencyx.Code("CREDITS")
	originalCurrency, err := env.Currency.CreateCurrency(t.Context(), currencytestutils.NewCreateCurrencyInput(namespace, customCode, "Credits", "cr"))
	require.NoError(t, err)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: originalCurrency.ID,
		FiatCode:   currencyx.Code(currency.USD),
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

	replacementCurrency, err := env.Currency.CreateCurrency(t.Context(), currencytestutils.NewCreateCurrencyInput(namespace, customCode, "Replacement credits", "cr2"))
	require.NoError(t, err)
	require.NotEqual(t, originalCurrency.ID, replacementCurrency.ID)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: replacementCurrency.ID,
		FiatCode:   currencyx.Code(currency.USD),
		Rate:       decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	versionTwoDraft, err := env.Plan.NextPlan(t.Context(), plan.NextPlanInput{
		NamespacedID: versionOne.NamespacedID,
	})
	require.NoError(t, err)
	require.Equal(t, 2, versionTwoDraft.Version)
	require.Equal(t, productcatalog.PlanStatusDraft, versionTwoDraft.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionTwoDraft)

	cutover := now.Add(time.Hour)
	_, err = env.Plan.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: versionTwoDraft.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(cutover),
		},
	})
	require.ErrorIs(t, err, productcatalog.ErrCurrencyNotFound)

	versionOne, err = env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: versionOne.NamespacedID})
	require.NoError(t, err)
	require.Equal(t, productcatalog.PlanStatusActive, versionOne.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionOne)

	versionTwoDraft, err = env.Plan.GetPlan(t.Context(), plan.GetPlanInput{NamespacedID: versionTwoDraft.NamespacedID})
	require.NoError(t, err)
	require.Equal(t, productcatalog.PlanStatusDraft, versionTwoDraft.Status())
	env.requirePlanRateCardCurrencyID(t, originalCurrency.ID, *versionTwoDraft)
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
			Currency:    lo.ToPtr(currencies.NewCurrencyReference(code)),
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

func (e *testEnv) requireManagedCurrencyID(t *testing.T, expectedID string, reference *currencies.CurrencyReference) {
	t.Helper()

	require.NotNil(t, reference)
	require.NotNil(t, reference.CustomCurrencyID)
	require.Equal(t, expectedID, *reference.CustomCurrencyID)
	resolved, ok := reference.CustomCurrency()
	require.True(t, ok, "custom currency reference must retain its resolved resource")
	require.Equal(t, expectedID, resolved.ID)
}
