package taxcode_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	taxcodevalidator "github.com/openmeterio/openmeter/openmeter/productcatalog/validators/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// newValidator creates a Validator backed by the given ent client.
func newValidator(t *testing.T, client *entdb.Client) *taxcodevalidator.Validator {
	t.Helper()
	v, err := taxcodevalidator.NewValidator(client)
	require.NoError(t, err)
	return v
}

// newTaxCode creates a bare tax code (no app mappings) via the taxcode test env
// that shares the same ent client as the productcatalog env.
func newTaxCode(t *testing.T, tcEnv *taxcodetestutils.TestEnv, namespace string) taxcode.TaxCode {
	t.Helper()
	return tcEnv.CreateTaxCode(t, namespace)
}

// insertPlanWithRateCard inserts a Plan + PlanPhase + PlanRateCard directly via the ent
// client, bypassing the service layer. This lets us control effective_from/effective_to
// and deleted_at precisely.
func insertPlanWithRateCard(
	t *testing.T,
	client *entdb.Client,
	namespace string,
	taxCodeID string,
	effectiveFrom time.Time,
	effectiveTo *time.Time, // nil = not archived
	deletedAt *time.Time, // nil = not soft-deleted
) {
	t.Helper()

	ctx := t.Context()

	pc := client.Plan.Create().
		SetNamespace(namespace).
		SetName("test-plan").
		SetKey("test-plan-" + namespace).
		SetVersion(1).
		SetCurrency("USD").
		SetBillingCadence("P1M").
		SetProRatingConfig(productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}).
		SetEffectiveFrom(effectiveFrom)

	if effectiveTo != nil {
		pc = pc.SetEffectiveTo(*effectiveTo)
	}
	if deletedAt != nil {
		pc = pc.SetDeletedAt(*deletedAt)
	}

	p, err := pc.Save(ctx)
	require.NoError(t, err)

	pp, err := client.PlanPhase.Create().
		SetNamespace(namespace).
		SetName("default").
		SetKey("default").
		SetPlanID(p.ID).
		SetIndex(0).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PlanRateCard.Create().
		SetNamespace(namespace).
		SetKey("test-rc").
		SetName("Test RC").
		SetType(productcatalog.FlatFeeRateCardType).
		SetPhaseID(pp.ID).
		SetTaxCodeID(taxCodeID).
		SetMetadata(map[string]string{}).
		Save(ctx)
	require.NoError(t, err)
}

// insertAddonWithRateCard inserts an Addon + AddonRateCard directly via the ent client.
func insertAddonWithRateCard(
	t *testing.T,
	client *entdb.Client,
	namespace string,
	taxCodeID string,
	effectiveTo *time.Time, // nil = not archived
	deletedAt *time.Time, // nil = not soft-deleted
) {
	t.Helper()

	ctx := t.Context()

	ac := client.Addon.Create().
		SetNamespace(namespace).
		SetName("test-addon").
		SetKey("test-addon-" + namespace).
		SetVersion(1).
		SetCurrency("USD").
		SetInstanceType(productcatalog.AddonInstanceTypeSingle).
		SetAnnotations(models.Annotations{})

	if effectiveTo != nil {
		ac = ac.SetEffectiveTo(*effectiveTo)
	}
	if deletedAt != nil {
		ac = ac.SetDeletedAt(*deletedAt)
	}

	a, err := ac.Save(ctx)
	require.NoError(t, err)

	_, err = client.AddonRateCard.Create().
		SetNamespace(namespace).
		SetAddonID(a.ID).
		SetKey("test-rc").
		SetName("Test RC").
		SetType(productcatalog.FlatFeeRateCardType).
		SetTaxCodeID(taxCodeID).
		SetMetadata(map[string]string{}).
		SetEntitlementTemplate(nil).
		SetDiscounts(nil).
		Save(ctx)
	require.NoError(t, err)
}

// TestValidateDeleteTaxCode is an integration test that exercises the validator
// against a real (migrated) Postgres schema.
func TestValidateDeleteTaxCode(t *testing.T) {
	// given: a productcatalog test env with schema migrated and a shared ent client
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	// taxcode test env reusing the same DB client (no second DB connection needed)
	tcEnv := taxcodetestutils.NewTestEnvFromClient(t, env.Client, nil)

	t.Run("NonArchivedPlanReferenceBlocksDeletion", func(t *testing.T) {
		// given: a fresh namespace, a tax code, and a non-archived plan with a ratecard
		// referencing that tax code
		namespace := pctestutils.NewTestNamespace(t)
		tc := newTaxCode(t, tcEnv, namespace)
		validator := newValidator(t, env.Client)

		// Insert a non-archived plan (effective_to IS NULL) with the ratecard referencing tc.
		insertPlanWithRateCard(t, env.Client, namespace, tc.ID,
			time.Now().Add(-time.Hour), // effective_from in the past
			nil,                        // not archived
			nil,                        // not deleted
		)

		// when: we try to delete the tax code
		err := validator.ValidateDeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: tc.ID},
		})

		// then: a conflict error is returned
		require.Error(t, err)
		require.True(t, models.IsGenericConflictError(err),
			"expected GenericConflictError, got: %T %v", err, err)
	})

	t.Run("NonArchivedAddonReferenceBlocksDeletion", func(t *testing.T) {
		// given: a fresh namespace, a tax code, and a non-archived addon with a ratecard
		// referencing that tax code
		namespace := pctestutils.NewTestNamespace(t)
		tc := newTaxCode(t, tcEnv, namespace)
		validator := newValidator(t, env.Client)

		// Insert a non-archived addon (effective_to IS NULL) with the ratecard referencing tc.
		insertAddonWithRateCard(t, env.Client, namespace, tc.ID,
			nil, // not archived
			nil, // not deleted
		)

		// when: we try to delete the tax code
		err := validator.ValidateDeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: tc.ID},
		})

		// then: a conflict error is returned
		require.Error(t, err)
		require.True(t, models.IsGenericConflictError(err),
			"expected GenericConflictError, got: %T %v", err, err)
	})

	t.Run("NoReferenceAllowsDeletion", func(t *testing.T) {
		// given: a tax code that has no ratecard references at all
		namespace := pctestutils.NewTestNamespace(t)
		tc := newTaxCode(t, tcEnv, namespace)
		validator := newValidator(t, env.Client)

		// when: we try to delete the tax code
		err := validator.ValidateDeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: tc.ID},
		})

		// then: deletion is allowed
		require.NoError(t, err)
	})

	t.Run("ArchivedPlanDoesNotBlockDeletion", func(t *testing.T) {
		// given: a tax code referenced only by an archived plan (effective_to in the past)
		namespace := pctestutils.NewTestNamespace(t)
		tc := newTaxCode(t, tcEnv, namespace)
		validator := newValidator(t, env.Client)

		past := time.Now().Add(-time.Hour)
		insertPlanWithRateCard(t, env.Client, namespace, tc.ID,
			time.Now().Add(-2*time.Hour), // effective_from two hours ago
			lo.ToPtr(past),               // archived: effective_to is in the past
			nil,                          // not soft-deleted
		)

		// when: we try to delete the tax code
		err := validator.ValidateDeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: tc.ID},
		})

		// then: archived plan does NOT block deletion
		require.NoError(t, err)
	})

	t.Run("SoftDeletedPlanDoesNotBlockDeletion", func(t *testing.T) {
		// given: a tax code referenced only by a soft-deleted plan
		namespace := pctestutils.NewTestNamespace(t)
		tc := newTaxCode(t, tcEnv, namespace)
		validator := newValidator(t, env.Client)

		now := time.Now()
		insertPlanWithRateCard(t, env.Client, namespace, tc.ID,
			now.Add(-time.Hour), // effective_from in the past
			nil,                 // not archived (effective_to IS NULL)
			lo.ToPtr(now),       // soft-deleted
		)

		// when: we try to delete the tax code
		err := validator.ValidateDeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: tc.ID},
		})

		// then: soft-deleted plan does NOT block deletion
		require.NoError(t, err)
	})

	t.Run("ArchivedAddonDoesNotBlockDeletion", func(t *testing.T) {
		// given: a tax code referenced only by an archived addon (effective_to in the past)
		namespace := pctestutils.NewTestNamespace(t)
		tc := newTaxCode(t, tcEnv, namespace)
		validator := newValidator(t, env.Client)

		past := time.Now().Add(-time.Hour)
		insertAddonWithRateCard(t, env.Client, namespace, tc.ID,
			lo.ToPtr(past), // archived
			nil,            // not soft-deleted
		)

		// when: we try to delete the tax code
		err := validator.ValidateDeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: tc.ID},
		})

		// then: archived addon does NOT block deletion
		require.NoError(t, err)
	})
}

// TestNewValidator_NilClient ensures the constructor rejects a nil ent client.
func TestNewValidator_NilClient(t *testing.T) {
	_, err := taxcodevalidator.NewValidator(nil)
	require.Error(t, err)
}
