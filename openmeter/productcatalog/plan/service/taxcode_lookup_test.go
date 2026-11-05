package service_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// failingTaxCodeResolver answers BatchResolve with err, or with no Tax Codes when err is nil.
type failingTaxCodeResolver struct {
	err error
}

var _ productcatalog.TaxCodeResolver = (*failingTaxCodeResolver)(nil)

func (r failingTaxCodeResolver) Resolve(context.Context, string, string) (*taxcode.TaxCode, error) {
	return nil, errors.New("not implemented")
}

func (r failingTaxCodeResolver) BatchResolve(context.Context, string, ...string) (map[string]*taxcode.TaxCode, error) {
	if r.err != nil {
		return nil, r.err
	}

	return map[string]*taxcode.TaxCode{}, nil
}

func (r failingTaxCodeResolver) WithNamespace(string) productcatalog.NamespacedTaxCodeResolver {
	return nil
}

// The plan service prefixes tax code errors with the phase. A field prefix turns any error into a
// validation issue, so this checks a failed lookup still reaches the HTTP layer as an internal error.
func TestPlanTaxCodeLookupErrors(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)

	featureResolver, err := featureresolver.New(env.Feature)
	require.NoError(t, err)

	t.Run("Create", func(t *testing.T) {
		t.Run("lookup failure", func(t *testing.T) {
			// given: a tax code resolver that can't reach the database
			svc := newLookupTestPlanService(t, env, featureResolver, failingTaxCodeResolver{err: errTaxCodeLookupFailure})

			// when: a plan referencing a Tax Code is created
			_, err := svc.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "create-lookup-failure", lookupTestRateCard()))

			// then: the error stays internal
			requireLookupInternalError(t, err)
		})

		t.Run("missing tax code", func(t *testing.T) {
			// given: a tax code resolver that finds nothing
			svc := newLookupTestPlanService(t, env, featureResolver, failingTaxCodeResolver{})

			// when: a plan referencing a Tax Code is created
			_, err := svc.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "create-missing", lookupTestRateCard()))

			// then: it is a validation issue on the phase
			requireLookupPhaseValidationError(t, err)
		})
	})

	t.Run("Update", func(t *testing.T) {
		p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-lookup", pctestutils.NewTestFlatRateCard("rc", nil)))
		require.NoError(t, err)

		t.Run("lookup failure", func(t *testing.T) {
			// given: a tax code resolver that can't reach the database
			svc := newLookupTestPlanService(t, env, featureResolver, failingTaxCodeResolver{err: errTaxCodeLookupFailure})

			// when: the plan's rate card is updated to reference a Tax Code
			_, err := svc.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       lookupTestUpdatePhases(),
			})

			// then: the error stays internal
			requireLookupInternalError(t, err)
		})

		t.Run("missing tax code", func(t *testing.T) {
			// given: a tax code resolver that finds nothing
			svc := newLookupTestPlanService(t, env, featureResolver, failingTaxCodeResolver{})

			// when: the plan's rate card is updated to reference a Tax Code
			_, err := svc.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       lookupTestUpdatePhases(),
			})

			// then: it is a validation issue on the phase
			requireLookupPhaseValidationError(t, err)
		})
	})
}

// errTaxCodeLookupFailure simulates the resolver failing to reach the database.
var errTaxCodeLookupFailure = errors.New("connection refused")

// newLookupTestPlanService builds the plan service around a custom Tax Code resolver.
func newLookupTestPlanService(
	t *testing.T,
	env *pctestutils.TestEnv,
	featureResolver productcatalog.FeatureResolver,
	resolver productcatalog.TaxCodeResolver,
) plan.Service {
	t.Helper()

	svc, err := planservice.New(planservice.Config{
		Adapter:          env.PlanRepository,
		Logger:           env.Logger,
		Publisher:        env.Publisher,
		FeatureResolver:  featureResolver,
		CurrencyResolver: env.CurrencyResolver,
		TaxCodeResolver:  resolver,
	})
	require.NoError(t, err)

	return svc
}

// handledAsValidationError reports whether the v1 plan error encoder answers err with a 400.
func handledAsValidationError(t *testing.T, err error) bool {
	t.Helper()

	rec := httptest.NewRecorder()
	handled := productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)(
		t.Context(), err, rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if handled {
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	return handled
}

// lookupTestRateCard returns a flat fee rate card referencing a Tax Code by ID.
func lookupTestRateCard() productcatalog.RateCard {
	return pctestutils.NewTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{TaxCodeID: "01JTAXC0DE0000000000000000"})
}

// lookupTestUpdatePhases wraps a taxed rate card in a single replacement phase.
func lookupTestUpdatePhases() *[]productcatalog.Phase {
	return &[]productcatalog.Phase{{
		PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
		RateCards: productcatalog.RateCards{lookupTestRateCard()},
	}}
}

// requireLookupInternalError asserts the error stays an internal failure, not a validation issue.
func requireLookupInternalError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.ErrorIs(t, err, errTaxCodeLookupFailure)
	assert.False(t, models.IsGenericValidationError(err), "lookup failure must not be a validation error: %v", err)

	_, issuesErr := models.AsValidationIssues(err)
	assert.Error(t, issuesErr, "lookup failure must not convert to validation issues")
	assert.False(t, handledAsValidationError(t, err), "lookup failure must not be answered with a 400")
}

// requireLookupPhaseValidationError asserts the error is a validation issue on the phase.
func requireLookupPhaseValidationError(t *testing.T, err error) {
	t.Helper()

	pctestutils.RequireTaxCodeNotFound(t, err)

	issues, issuesErr := models.AsValidationIssues(err)
	require.NoError(t, issuesErr)
	require.NotEmpty(t, issues)
	assert.Contains(t, issues[0].Field().String(), "phases")
	assert.True(t, handledAsValidationError(t, err))
}
