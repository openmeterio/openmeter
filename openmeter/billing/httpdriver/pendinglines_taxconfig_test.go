package httpdriver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// PendingLineTaxConfigTestSuite covers the v1 pending-line create and invoice update handlers'
// tax config resolution: a line's stripe.code becomes a Tax Code ID inside the request's
// transaction, taxCodeId wins, and responses render stripe.code from the Tax Code's current
// mapping.
type PendingLineTaxConfigTestSuite struct {
	billingtest.BaseSuite

	namespace        string
	appID            app.AppID
	resolver         *countingTaxCodeResolver
	handler          Handler
	condemnedTaxCode taxcode.TaxCode
}

func TestPendingLineTaxConfig(t *testing.T) {
	suite.Run(t, new(PendingLineTaxConfigTestSuite))
}

func (s *PendingLineTaxConfigTestSuite) SetupTest() {
	s.namespace = s.GetUniqueNamespace("pendingline-taxconfig-v1")
	s.appID = s.InstallSandboxApp(s.T(), s.namespace).GetID()
	s.resolver = &countingTaxCodeResolver{TransactionalTaxCodeResolver: s.TaxCodeResolver}
	s.handler = New(testutils.NewDiscardLogger(s.T()), namespacedriver.StaticNamespaceDecoder(s.namespace),
		config.BillingFeatureSwitchesConfiguration{}, s.BillingService, s.AppService, nil, nil,
		config.CreditsConfiguration{}, nil, s.resolver)
}

func (s *PendingLineTaxConfigTestSuite) taxCodeCount() int {
	count, err := s.DBClient.TaxCode.Query().Where(taxcodedb.Namespace(s.namespace)).Count(s.T().Context())
	s.Require().NoError(err)

	return count
}

func (s *PendingLineTaxConfigTestSuite) createPendingLines(customerID string, lines []apiv1models.InvoicePendingLineCreate) *httptest.ResponseRecorder {
	raw, err := json.Marshal(apiv1models.InvoicePendingLineCreateInput{
		Currency: apiv1models.CurrencyCode(currency.USD),
		Lines:    lines,
	})
	s.Require().NoError(err)

	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	s.handler.CreatePendingLine().With(CreatePendingLineParams{CustomerID: customerID}).ServeHTTP(recorder, request)

	return recorder
}

func (s *PendingLineTaxConfigTestSuite) flatLine(name string, taxConfig *apiv1models.TaxConfig) apiv1models.InvoicePendingLineCreate {
	return apiv1models.InvoicePendingLineCreate{
		Name:      name,
		InvoiceAt: invoiceLineTestPeriod().To,
		Period:    apiv1models.Period(invoiceLineTestPeriod()),
		Price:     flatLinePriceForTaxConfigTest(),
		TaxConfig: taxConfig,
	}
}

func flatLinePriceForTaxConfigTest() *apiv1models.RateCardUsageBasedPrice {
	price := apiv1models.RateCardUsageBasedPrice{}
	// Prices here are static shapes; the encode step cannot fail for these inputs.
	_ = price.FromFlatPriceWithPaymentTerm(apiv1models.FlatPriceWithPaymentTerm{
		Amount: "10",
		Type:   apiv1models.FlatPriceWithPaymentTermTypeFlat,
	})

	return &price
}

func (s *PendingLineTaxConfigTestSuite) createResponse(recorder *httptest.ResponseRecorder) apiv1models.InvoicePendingLineCreateResponse {
	var resp apiv1models.InvoicePendingLineCreateResponse
	s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))

	return resp
}

func (s *PendingLineTaxConfigTestSuite) getInvoice(invoiceID string) apiv1models.Invoice {
	recorder := httptest.NewRecorder()
	s.handler.GetInvoice().With(GetInvoiceParams{InvoiceID: invoiceID}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

	var invoice apiv1models.Invoice
	s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &invoice))

	return invoice
}

func (s *PendingLineTaxConfigTestSuite) TestCreatePendingLineResolvesStripeCodeInRequestTransaction() {
	ctx := s.T().Context()

	// given: a customer with a billing profile and the organization default tax codes the
	// service stamps onto lines without a tax config
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), s.namespace)
	s.ProvisionBillingProfile(ctx, s.namespace, s.appID)
	cust := s.CreateTestCustomer(s.namespace, "create-stripe-code")

	s.Run("a new tax code is created inside the request transaction", func() {
		// when: a pending line with only a stripe.code is created
		rec := s.createPendingLines(cust.ID, []apiv1models.InvoicePendingLineCreate{
			s.flatLine("line", &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: "txcd_10000000"}}),
		})
		s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())

		// then: the response renders the tax code the stripe code maps to
		created := s.createResponse(rec)
		s.Require().Len(created.Lines, 1)
		line := created.Lines[0]
		s.Require().NotNil(line.TaxConfig)
		s.Require().NotEmpty(lo.FromPtr(line.TaxConfig.TaxCodeId))

		tc, err := s.TaxCodeEnv.Service.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
			Namespace: s.namespace,
			AppType:   app.AppTypeStripe,
			TaxCode:   "txcd_10000000",
		})
		s.Require().NoError(err)
		s.Equal(&apiv1models.TaxConfig{
			Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(tc.ID),
		}, line.TaxConfig)
	})

	s.Run("a failing request leaves no orphan tax code", func() {
		// given: a tax code the request will reference after deleting it, so the service
		// rejects the second line after the first line's stripe.code was already resolved
		before := s.taxCodeCount()
		s.condemnedTaxCode = s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, "txcd_20000000")
		s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: s.condemnedTaxCode.NamespacedID}))
		afterDelete := s.taxCodeCount()

		// when: the first line resolves a new stripe.code and the second references the
		// deleted tax code by ID
		rec := s.createPendingLines(cust.ID, []apiv1models.InvoicePendingLineCreate{
			s.flatLine("fine", &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: "txcd_20000000"}}),
			s.flatLine("deleted-ref", &apiv1models.TaxConfig{TaxCodeId: lo.ToPtr(s.condemnedTaxCode.ID)}),
		})

		// then: the request fails with the deleted-code issue and nothing was created or kept:
		// the count matches the state before the request (including the deleted row)
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Contains(rec.Body.String(), "billing_tax_code_deleted")
		s.Equal(afterDelete, s.taxCodeCount())
		s.NotEqual(before, s.taxCodeCount(), "precondition: the deleted row counts")
	})
}

func unitLinePriceForTaxConfigTest() *apiv1models.RateCardUsageBasedPrice {
	price := apiv1models.RateCardUsageBasedPrice{}
	_ = price.FromUnitPriceWithCommitments(apiv1models.UnitPriceWithCommitments{
		Amount: "1",
		Type:   apiv1models.UnitPriceWithCommitmentsTypeUnit,
	})

	return &price
}

func (s *PendingLineTaxConfigTestSuite) TestCreatePendingLineTaxCodeIDWins() {
	// given: a customer and a tax code with a stripe mapping
	s.ProvisionBillingProfile(s.T().Context(), s.namespace, s.appID)
	cust := s.CreateTestCustomer(s.namespace, "create-id-wins")
	stripeCode := "txcd_30000000"
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, stripeCode)

	// when: the line sends both the id and a different stripe.code
	rec := s.createPendingLines(cust.ID, []apiv1models.InvoicePendingLineCreate{
		s.flatLine("line", &apiv1models.TaxConfig{
			Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_99000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}),
	})
	s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())

	// then: the id wins and the response renders the tax code's own mapping
	created := s.createResponse(rec)
	s.Require().Len(created.Lines, 1)
	s.Equal(&apiv1models.TaxConfig{
		Stripe:    &apiv1models.StripeTaxConfig{Code: stripeCode},
		TaxCodeId: lo.ToPtr(taxCode.ID),
	}, created.Lines[0].TaxConfig)
}

// The create response renders the whole gathering invoice, so lines an earlier request added
// with another Tax Code must render too.
func (s *PendingLineTaxConfigTestSuite) TestCreatePendingLineRendersEarlierLinesTaxCodes() {
	// given: a gathering invoice holding a line with one tax code
	s.ProvisionBillingProfile(s.T().Context(), s.namespace, s.appID)
	cust := s.CreateTestCustomer(s.namespace, "create-renders-earlier-lines")
	firstTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, "txcd_30000000")
	secondTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, "txcd_30000001")

	firstRec := s.createPendingLines(cust.ID, []apiv1models.InvoicePendingLineCreate{
		s.flatLine("first", &apiv1models.TaxConfig{TaxCodeId: lo.ToPtr(firstTaxCode.ID)}),
	})
	s.Require().Equal(http.StatusCreated, firstRec.Code, firstRec.Body.String())

	// when: another request adds a line with a different tax code
	secondRec := s.createPendingLines(cust.ID, []apiv1models.InvoicePendingLineCreate{
		s.flatLine("second", &apiv1models.TaxConfig{TaxCodeId: lo.ToPtr(secondTaxCode.ID)}),
	})
	s.Require().Equal(http.StatusCreated, secondRec.Code, secondRec.Body.String())

	// then: the invoice in the response renders both lines with their own tax codes
	created := s.createResponse(secondRec)
	s.Require().NotNil(created.Invoice.Lines)

	stripeCodes := map[string]string{}
	for _, line := range *created.Invoice.Lines {
		s.Require().NotNil(line.TaxConfig, line.Name)
		s.Require().NotNil(line.TaxConfig.Stripe, line.Name)
		stripeCodes[line.Name] = line.TaxConfig.Stripe.Code
	}

	s.Equal(map[string]string{"first": "txcd_30000000", "second": "txcd_30000001"}, stripeCodes)
}

func (s *PendingLineTaxConfigTestSuite) TestUpdateInvoiceResolvesStripeCode() {
	ctx := s.T().Context()

	// given: a customer pinned to a non-auto-advancing profile, a gathering invoice with one
	// line without a tax config, and a tax code with a stripe mapping. The edit-diff
	// sanitizer resolves lines without a tax config to the provider default tax code.
	s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), s.namespace)
	profile := s.ProvisionBillingProfile(ctx, s.namespace, s.appID, billingtest.WithBillingProfileEditFn(func(input *billing.CreateProfileInput) {
		input.WorkflowConfig.Invoicing.AutoAdvance = false
	}))
	cust := s.CreateTestCustomer(s.namespace, "update-stripe-code")
	_, err := s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  cust.GetID(),
		ProfileID: profile.ID,
	})
	s.Require().NoError(err)

	stripeCode := "txcd_40000000"
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, stripeCode)

	createRec := s.createPendingLines(cust.ID, []apiv1models.InvoicePendingLineCreate{
		s.flatLine("line", nil),
	})
	s.Require().Equal(http.StatusCreated, createRec.Code, createRec.Body.String())
	invoiceID := s.createResponse(createRec).Invoice.Id

	// when: the invoice edit gives the line a stripe.code only
	gatheringInvoice := s.getInvoice(invoiceID)
	lines := s.echoLines(gatheringInvoice)
	rateCard := *lines[0].RateCard
	rateCard.TaxConfig = &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: stripeCode}}
	lines[0].TaxConfig = rateCard.TaxConfig
	lines[0].RateCard = &rateCard

	updated := s.updateInvoice(gatheringInvoice, lines)

	// then: the line resolves to the tax code and renders its stripe code
	updatedLines := lo.FromPtr(updated.Lines)
	s.Require().Len(updatedLines, 1)
	s.Require().NotNil(updatedLines[0].TaxConfig)
	s.Equal(&apiv1models.TaxConfig{
		Stripe:    &apiv1models.StripeTaxConfig{Code: stripeCode},
		TaxCodeId: lo.ToPtr(taxCode.ID),
	}, updatedLines[0].TaxConfig)

	// and the stored line references the tax code without an embedded stripe code
	stored, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
		Invoice: billing.InvoiceID{Namespace: s.namespace, ID: invoiceID},
		Expand:  billing.InvoiceExpandAll,
	})
	s.Require().NoError(err)
	gatheringStored, err := stored.AsGatheringInvoice()
	s.Require().NoError(err)
	storedLines := gatheringStored.Lines.OrEmpty()
	s.Require().Len(storedLines, 1)
	s.Require().NotNil(storedLines[0].TaxConfig)
	s.Require().NotNil(storedLines[0].TaxConfig.TaxCodeID)
	s.Equal(taxCode.ID, *storedLines[0].TaxConfig.TaxCodeID)
}

func (s *PendingLineTaxConfigTestSuite) echoLines(invoice apiv1models.Invoice) []apiv1models.InvoiceLineReplaceUpdate {
	out := make([]apiv1models.InvoiceLineReplaceUpdate, 0, len(lo.FromPtr(invoice.Lines)))
	for _, line := range lo.FromPtr(invoice.Lines) {
		out = append(out, apiv1models.InvoiceLineReplaceUpdate{
			Id:         lo.ToPtr(line.Id),
			Name:       line.Name,
			FeatureKey: line.FeatureKey,
			InvoiceAt:  line.InvoiceAt,
			Metadata:   line.Metadata,
			Period:     line.Period,
			Price:      line.Price,
			RateCard:   line.RateCard,
			TaxConfig:  line.TaxConfig,
		})
	}

	return out
}

func (s *PendingLineTaxConfigTestSuite) updateInvoice(invoice apiv1models.Invoice, lines []apiv1models.InvoiceLineReplaceUpdate) apiv1models.Invoice {
	raw, err := json.Marshal(apiv1models.InvoiceReplaceUpdate{
		Customer: apiv1models.BillingPartyReplaceUpdate{
			Name:      invoice.Customer.Name,
			Addresses: invoice.Customer.Addresses,
			TaxId:     nil,
		},
		Supplier: apiv1models.BillingPartyReplaceUpdate{
			Name:      invoice.Supplier.Name,
			Addresses: invoice.Supplier.Addresses,
			TaxId:     invoice.Supplier.TaxId,
		},
		Lines: lines,
		Workflow: apiv1models.InvoiceWorkflowReplaceUpdate{
			Workflow: apiv1models.InvoiceWorkflowSettingsReplaceUpdate{
				Invoicing: apiv1models.InvoiceWorkflowInvoicingSettingsReplaceUpdate{
					AutoAdvance: lo.FromPtr(invoice.Workflow.Workflow.Invoicing).AutoAdvance,
					DraftPeriod: lo.FromPtr(invoice.Workflow.Workflow.Invoicing).DraftPeriod,
					DueAfter:    lo.FromPtr(invoice.Workflow.Workflow.Invoicing).DueAfter,
				},
				Payment: lo.FromPtr(invoice.Workflow.Workflow.Payment),
			},
		},
	})
	s.Require().NoError(err)

	request := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	s.handler.UpdateInvoice().With(UpdateInvoiceParams{InvoiceID: invoice.Id}).ServeHTTP(recorder, request)
	s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

	var updated apiv1models.Invoice
	s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &updated))

	return updated
}
