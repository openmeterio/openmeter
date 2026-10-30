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

	newPlanService := func(t *testing.T, resolver productcatalog.TaxCodeResolver) plan.Service {
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
	handledAsValidationError := func(t *testing.T, err error) bool {
		t.Helper()

		rec := httptest.NewRecorder()
		handled := productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)(
			t.Context(), err, rec, httptest.NewRequest(http.MethodPost, "/", nil))
		if handled {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}

		return handled
	}

	rateCard := func() productcatalog.RateCard {
		return newTestFlatRateCard("rc", &productcatalog.TaxCodeConfig{TaxCodeID: "01JTAXCODE00000000000000000"})
	}

	updatePhases := func() *[]productcatalog.Phase {
		return &[]productcatalog.Phase{{
			PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
			RateCards: productcatalog.RateCards{rateCard()},
		}}
	}

	lookupErr := errors.New("connection refused")

	requireInternalError := func(t *testing.T, err error) {
		t.Helper()

		require.Error(t, err)
		assert.ErrorIs(t, err, lookupErr)
		assert.False(t, models.IsGenericValidationError(err), "lookup failure must not be a validation error: %v", err)

		_, issuesErr := models.AsValidationIssues(err)
		assert.Error(t, issuesErr, "lookup failure must not convert to validation issues")
		assert.False(t, handledAsValidationError(t, err), "lookup failure must not be answered with a 400")
	}

	requirePhaseValidationError := func(t *testing.T, err error) {
		t.Helper()

		requireTaxCodeNotFound(t, err)

		issues, issuesErr := models.AsValidationIssues(err)
		require.NoError(t, issuesErr)
		require.NotEmpty(t, issues)
		assert.Contains(t, issues[0].Field().String(), "phases")
		assert.True(t, handledAsValidationError(t, err))
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("lookup failure", func(t *testing.T) {
			// given: a tax code resolver that can't reach the database
			svc := newPlanService(t, failingTaxCodeResolver{err: lookupErr})

			// when: a plan referencing a Tax Code is created
			_, err := svc.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "create-lookup-failure", rateCard()))

			// then: the error stays internal
			requireInternalError(t, err)
		})

		t.Run("missing tax code", func(t *testing.T) {
			// given: a tax code resolver that finds nothing
			svc := newPlanService(t, failingTaxCodeResolver{})

			// when: a plan referencing a Tax Code is created
			_, err := svc.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "create-missing", rateCard()))

			// then: it is a validation issue on the phase
			requirePhaseValidationError(t, err)
		})
	})

	t.Run("Update", func(t *testing.T) {
		p, err := env.Plan.CreatePlan(t.Context(), newTestPlanInput(t, namespace, "update-lookup", newTestFlatRateCard("rc", nil)))
		require.NoError(t, err)

		t.Run("lookup failure", func(t *testing.T) {
			// given: a tax code resolver that can't reach the database
			svc := newPlanService(t, failingTaxCodeResolver{err: lookupErr})

			// when: the plan's rate card is updated to reference a Tax Code
			_, err := svc.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       updatePhases(),
			})

			// then: the error stays internal
			requireInternalError(t, err)
		})

		t.Run("missing tax code", func(t *testing.T) {
			// given: a tax code resolver that finds nothing
			svc := newPlanService(t, failingTaxCodeResolver{})

			// when: the plan's rate card is updated to reference a Tax Code
			_, err := svc.UpdatePlan(t.Context(), plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       updatePhases(),
			})

			// then: it is a validation issue on the phase
			requirePhaseValidationError(t, err)
		})
	})
}
