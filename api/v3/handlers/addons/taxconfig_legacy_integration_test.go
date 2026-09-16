package addons

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
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
	}, s.env.Addon, true)
}

func (s *LegacyTaxConfigAPISuite) TearDownSuite() {
	s.env.Close(s.T())
}

func (s *LegacyTaxConfigAPISuite) TestGetPutRoundTripAcceptsLegacyTaxConfigWithoutTaxCodeEdge() {
	ctx := s.T().Context()

	// Given a valid add-on whose rate card has only the pre-migration JSON tax config.
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:  "flat-fee",
			Name: "Flat fee",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		BillingCadence: &pctestutils.MonthPeriod,
	}
	created, err := s.env.Addon.CreateAddon(ctx, pctestutils.NewTestAddon(s.T(), s.namespace, rateCard))
	s.Require().NoError(err)
	s.Require().NotEmpty(created.RateCards)

	row, err := s.env.Client.AddonRateCard.Query().
		Where(
			addonratecard.AddonID(created.ID),
			addonratecard.Key(rateCard.Key()),
		).
		Only(ctx)
	s.Require().NoError(err)

	legacyTaxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_92000000",
		},
	}

	_, err = s.env.Client.AddonRateCard.UpdateOneID(row.ID).
		SetTaxConfig(legacyTaxConfig).
		ClearTaxCodeID().
		ClearTaxBehavior().
		Save(ctx)
	s.Require().NoError(err)

	persisted, err := s.env.Addon.GetAddon(ctx, addon.GetAddonInput{NamespacedID: created.NamespacedID})
	s.Require().NoError(err)
	s.Require().NotNil(persisted.RateCards[0].AsMeta().TaxConfig)
	s.Require().Nil(persisted.RateCards[0].AsMeta().TaxCode)

	// When a client reads the add-on and sends the representation back unchanged.
	getRequest := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v3/addons/"+created.ID, nil)
	getResponse := httptest.NewRecorder()
	s.handler.GetAddon().With(created.ID).ServeHTTP(getResponse, getRequest)
	s.Require().Equal(http.StatusOK, getResponse.Code, getResponse.Body.String())

	var result api.Addon
	s.Require().NoError(json.Unmarshal(getResponse.Body.Bytes(), &result))
	s.Require().Len(result.RateCards, 1)
	s.Require().NotNil(result.RateCards[0].TaxConfig, getResponse.Body.String())
	s.Require().NotNil(result.RateCards[0].TaxConfig.Behavior, getResponse.Body.String())

	updateBody, err := json.Marshal(api.UpsertAddonRequest{
		Name:         result.Name,
		InstanceType: result.InstanceType,
		RateCards:    result.RateCards,
	})
	s.Require().NoError(err)

	putRequest := httptest.NewRequestWithContext(ctx, http.MethodPut, "/api/v3/addons/"+created.ID, bytes.NewReader(updateBody))
	putRequest.Header.Set("Content-Type", "application/json")
	putResponse := httptest.NewRecorder()
	s.handler.UpdateAddon().With(created.ID).ServeHTTP(putResponse, putRequest)

	// Then a representation produced by GET must remain valid input for PUT.
	s.Require().Equal(http.StatusOK, putResponse.Code, putResponse.Body.String())
}
