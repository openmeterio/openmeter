package plans

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
)

type LegacyTaxConfigAPISuite struct {
	suite.Suite

	env       *pctestutils.TestEnv
	namespace string
	handler   Handler
}

func TestLegacyTaxConfigAPI(t *testing.T) {
	suite.Run(t, new(LegacyTaxConfigAPISuite))
}

func (s *LegacyTaxConfigAPISuite) SetupSuite() {
	s.env = pctestutils.NewTestEnv(s.T())
	s.namespace = pctestutils.NewTestNamespace(s.T())
	s.handler = New(func(context.Context) (string, error) {
		return s.namespace, nil
	}, s.env.Plan, true)
}

func (s *LegacyTaxConfigAPISuite) TearDownSuite() {
	s.env.Close(s.T())
}

func (s *LegacyTaxConfigAPISuite) TestGetPreservesLegacyTaxConfigWithoutTaxCodeEdge() {
	ctx := s.T().Context()

	// Given a valid plan whose rate card has only the pre-migration JSON tax config.
	input := pctestutils.NewTestPlan(s.T(), s.namespace)
	flatFeeRateCard, ok := input.Phases[0].RateCards[0].(*productcatalog.FlatFeeRateCard)
	s.Require().True(ok)
	flatFeeRateCard.TaxConfig = nil

	created, err := s.env.Plan.CreatePlan(ctx, input)
	s.Require().NoError(err)
	s.Require().NotEmpty(created.Phases)
	s.Require().NotEmpty(created.Phases[0].RateCards)

	phaseID := created.Phases[0].ID
	rateCardKey := created.Phases[0].RateCards[0].Key()
	row, err := s.env.Client.PlanRateCard.Query().
		Where(
			planratecard.PhaseID(phaseID),
			planratecard.Key(rateCardKey),
		).
		Only(ctx)
	s.Require().NoError(err)

	legacyTaxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_91000000",
		},
	}

	_, err = s.env.Client.PlanRateCard.UpdateOneID(row.ID).
		SetTaxConfig(legacyTaxConfig).
		ClearTaxCodeID().
		ClearTaxBehavior().
		Save(ctx)
	s.Require().NoError(err)

	persisted, err := s.env.Plan.GetPlan(ctx, plan.GetPlanInput{NamespacedID: created.NamespacedID})
	s.Require().NoError(err)
	s.Require().NotNil(persisted.Phases[0].RateCards[0].AsMeta().TaxConfig)
	s.Require().Nil(persisted.Phases[0].RateCards[0].AsMeta().TaxCode)

	// When the real v3 handler reads the plan through the service and Ent adapter.
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v3/plans/"+created.ID, nil)
	response := httptest.NewRecorder()
	s.handler.GetPlan().With(created.ID).ServeHTTP(response, request)
	s.Require().Equal(http.StatusOK, response.Code, response.Body.String())

	var result api.BillingPlan
	s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &result))
	s.Require().Len(result.Phases, 1)
	s.Require().Len(result.Phases[0].RateCards, 1)

	// Then the API must not discard tax settings that are still valid persisted state.
	require.NotNil(s.T(), result.Phases[0].RateCards[0].TaxConfig, response.Body.String())
}
