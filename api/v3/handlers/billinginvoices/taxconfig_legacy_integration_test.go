package billinginvoices

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type LegacyTaxConfigAPISuite struct {
	billingtest.BaseSuite
}

func TestLegacyTaxConfigAPI(t *testing.T) {
	suite.Run(t, new(LegacyTaxConfigAPISuite))
}

func (s *LegacyTaxConfigAPISuite) TestGetPutRoundTripAcceptsLegacyTaxConfigWithoutTaxCodeEdge() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("legacy-tax-config-api")

	app := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, app.GetID())
	s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "legacy-tax-config-api")
	created := s.CreateDraftInvoice(s.T(), ctx, billingtest.DraftInvoiceInput{
		Namespace: namespace,
		Customer:  customer,
	})
	lines := created.Lines.MustGet()
	s.Require().NotEmpty(lines)

	// Given a valid standard invoice line whose tax settings only exist in the
	// pre-migration JSON snapshot, with no normalized FK or behavior column.
	legacyTaxConfig := billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			Stripe: &productcatalog.StripeTaxConfig{
				Code: "txcd_93000000",
			},
		},
	}

	_, err := s.DBClient.BillingInvoiceLine.UpdateOneID(lines[0].ID).
		SetTaxConfig(legacyTaxConfig).
		ClearTaxCodeID().
		ClearTaxBehavior().
		Save(ctx)
	s.Require().NoError(err)

	persisted, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: created.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	persistedLines := persisted.Lines.MustGet()
	s.Require().NotEmpty(persistedLines)
	s.Require().NotNil(persistedLines[0].TaxConfig)
	s.Require().Nil(persistedLines[0].TaxConfig.TaxCode)

	handler := New(func(context.Context) (string, error) {
		return namespace, nil
	}, s.BillingService)

	// When a client reads the invoice and sends the representation back unchanged.
	getRequest := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v3/billing/invoices/"+created.ID, nil)
	getResponse := httptest.NewRecorder()
	handler.GetBillingInvoice().With(created.ID).ServeHTTP(getResponse, getRequest)
	s.Require().Equal(http.StatusOK, getResponse.Code, getResponse.Body.String())

	var result api.BillingInvoice
	s.Require().NoError(json.Unmarshal(getResponse.Body.Bytes(), &result))
	standard, err := result.AsBillingInvoiceStandard()
	s.Require().NoError(err)
	s.Require().NotNil(standard.Lines)
	s.Require().NotEmpty(*standard.Lines)
	line, err := (*standard.Lines)[0].AsBillingInvoiceStandardLine()
	s.Require().NoError(err)
	s.Require().NotNil(line.RateCard.TaxConfig, getResponse.Body.String())
	s.Require().NotNil(line.RateCard.TaxConfig.Behavior, getResponse.Body.String())

	putRequest := httptest.NewRequestWithContext(ctx, http.MethodPut, "/api/v3/billing/invoices/"+created.ID, bytes.NewReader(getResponse.Body.Bytes()))
	putRequest.Header.Set("Content-Type", "application/json")
	putResponse := httptest.NewRecorder()
	handler.UpdateBillingInvoice().With(created.ID).ServeHTTP(putResponse, putRequest)

	// Then a representation produced by GET must remain valid input for PUT.
	s.Require().Equal(http.StatusOK, putResponse.Code, putResponse.Body.String())
}
