package httpdriver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type LegacyInvoiceDependencyHTTPSuite struct {
	billingtest.BaseSuite
}

func TestLegacyInvoiceDependencyHTTP(t *testing.T) {
	suite.Run(t, new(LegacyInvoiceDependencyHTTPSuite))
}

func (s *LegacyInvoiceDependencyHTTPSuite) dependencyRequest(h http.Handler, method string, body any) *httptest.ResponseRecorder {
	s.T().Helper()
	encoded, err := json.Marshal(body)
	s.Require().NoError(err)
	req := httptest.NewRequestWithContext(s.T().Context(), method, "/api/v1/billing/invoices", bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	s.Require().NotPanics(func() { h.ServeHTTP(response, req) }, "dependency failures must have an invoice response")
	s.T().Logf("%s invoice response: %d %s", method, response.Code, response.Body.String())

	return response
}

func (s *LegacyInvoiceDependencyHTTPSuite) requireVisibleDependencyIssue(invoice api.Invoice, lineID, code string) {
	s.T().Helper()
	s.Equal(string(billing.StandardInvoiceStatusDraftInvalidCreated), invoice.StatusDetails.ExtendedStatus)
	s.True(invoice.StatusDetails.Failed)
	s.NotNil(invoice.StatusDetails.AvailableActions.Retry)
	s.NotNil(invoice.StatusDetails.AvailableActions.Delete)
	s.Nil(invoice.IssuedAt)
	s.Require().NotNil(invoice.ValidationIssues)
	s.Require().Len(*invoice.ValidationIssues, 1)
	issue := (*invoice.ValidationIssues)[0]
	s.Equal(api.ValidationIssueSeverityCritical, issue.Severity)
	s.Equal(code, lo.FromPtr(issue.Code))
	s.Equal(fmt.Sprintf("/lines/%s", lineID), lo.FromPtr(issue.Field))
	s.Equal(string(billing.LineEngineValidationComponent(billing.LineEngineTypeInvoice)), issue.Component)
	s.NotEmpty(issue.Message)
}

func (s *LegacyInvoiceDependencyHTTPSuite) TestCustomerRetryWithBrokenDependency() {
	for _, dependency := range []string{"feature", "meter"} {
		s.Run(dependency, func() {
			// given: a valid legacy gathering line loses its feature or meter after recording 7 units at $2.
			fixture := s.newDependencyHTTPFixture()
			code := billing.ErrInvoiceLineFeatureNotFound.Code
			if dependency == "feature" {
				// Archival preserves historical resolution; physical loss models an unavailable legacy key.
				s.Require().NoError(s.DBClient.Feature.DeleteOneID(fixture.feature.ID).Exec(s.T().Context()))
			} else {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
				code = billing.ErrInvoiceLineFeatureHasNoMeters.Code
			}
			clock.FreezeTime(fixture.period.To.Add(2 * time.Hour))
			defer clock.UnFreeze()

			// when: the customer collects the due line, the response must expose a recoverable invoice.
			response := s.dependencyRequest(fixture.handler.InvoicePendingLinesAction(), http.MethodPost, api.InvoicePendingLinesActionInput{
				CustomerId: fixture.customerID, AsOf: &fixture.period.To,
			})
			s.Require().Equal(http.StatusCreated, response.Code, response.Body.String())
			var invoices []api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &invoices))
			s.Require().Len(invoices, 1)
			invoiceID := invoices[0].Id
			s.requireVisibleDependencyIssue(invoices[0], fixture.lineID, code)

			// then: list and detail reads must expose the same invoice and line-scoped issue.
			response = s.dependencyRequest(fixture.handler.ListInvoices().With(api.ListInvoicesParams{
				Customers: &[]string{fixture.customerID},
			}), http.MethodGet, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var listed api.InvoicePaginatedResponse
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &listed))
			s.Require().Len(listed.Items, 1)
			s.Equal(invoiceID, listed.Items[0].Id)
			s.requireVisibleDependencyIssue(listed.Items[0], fixture.lineID, code)
			get := fixture.handler.GetInvoice().With(GetInvoiceParams{InvoiceID: invoiceID})
			response = s.dependencyRequest(get, http.MethodGet, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var invoice api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &invoice))
			s.requireVisibleDependencyIssue(invoice, fixture.lineID, code)

			// when: the customer retries without repair, the line issue must remain visible.
			retry := fixture.handler.ProgressInvoice(InvoiceProgressActionRetry).With(ProgressInvoiceParams{InvoiceID: invoiceID})
			response = s.dependencyRequest(retry, http.MethodPost, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var unrepaired api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &unrepaired))
			s.requireVisibleDependencyIssue(unrepaired, fixture.lineID, code)

			// when: the customer restores the missing dependency and retries the same invoice.
			if dependency == "feature" {
				_, err := s.FeatureService.CreateFeature(s.T().Context(), feature.CreateFeatureInputs{
					Namespace: fixture.feature.Namespace, Name: fixture.feature.Name, Key: fixture.feature.Key, MeterID: fixture.feature.MeterID,
				})
				s.Require().NoError(err)
			} else {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{fixture.meter}))
			}
			response = s.dependencyRequest(retry, http.MethodPost, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var recovered api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &recovered))
			s.Equal(invoiceID, recovered.Id)
			s.Equal(string(billing.StandardInvoiceStatusDraftManualApprovalNeeded), recovered.StatusDetails.ExtendedStatus)
			s.Empty(lo.FromPtr(recovered.ValidationIssues))
			s.Equal("14", recovered.Totals.Total)

			// then: approval and a fresh read must show issuance for the recovered $14.
			response = s.dependencyRequest(fixture.handler.ProgressInvoice(InvoiceProgressActionApprove).With(ProgressInvoiceParams{InvoiceID: invoiceID}), http.MethodPost, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			response = s.dependencyRequest(get, http.MethodGet, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var issued api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &issued))
			s.Equal(invoiceID, issued.Id)
			s.NotNil(issued.IssuedAt)
			s.Empty(lo.FromPtr(issued.ValidationIssues))
			s.Equal("14", issued.Totals.Total)
		})
	}
}

func (s *LegacyInvoiceDependencyHTTPSuite) TestCustomerDeleteWithBrokenDependency() {
	for _, dependency := range []string{"feature", "meter"} {
		s.Run(dependency, func() {
			// given: a valid legacy gathering line loses its feature or meter after recording 7 units at $2.
			fixture := s.newDependencyHTTPFixture()
			code := billing.ErrInvoiceLineFeatureNotFound.Code
			if dependency == "feature" {
				// Archival preserves historical resolution; physical loss models an unavailable legacy key.
				s.Require().NoError(s.DBClient.Feature.DeleteOneID(fixture.feature.ID).Exec(s.T().Context()))
			} else {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
				code = billing.ErrInvoiceLineFeatureHasNoMeters.Code
			}
			clock.FreezeTime(fixture.period.To.Add(2 * time.Hour))
			defer clock.UnFreeze()

			// when: the customer collects the due line, the response must expose a recoverable invoice.
			response := s.dependencyRequest(fixture.handler.InvoicePendingLinesAction(), http.MethodPost, api.InvoicePendingLinesActionInput{
				CustomerId: fixture.customerID, AsOf: &fixture.period.To,
			})
			s.Require().Equal(http.StatusCreated, response.Code, response.Body.String())
			var invoices []api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &invoices))
			s.Require().Len(invoices, 1)
			invoiceID := invoices[0].Id
			s.requireVisibleDependencyIssue(invoices[0], fixture.lineID, code)

			// then: list and detail reads must expose the same invoice and line-scoped issue.
			response = s.dependencyRequest(fixture.handler.ListInvoices().With(api.ListInvoicesParams{
				Customers: &[]string{fixture.customerID},
			}), http.MethodGet, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var listed api.InvoicePaginatedResponse
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &listed))
			s.Require().Len(listed.Items, 1)
			s.Equal(invoiceID, listed.Items[0].Id)
			s.requireVisibleDependencyIssue(listed.Items[0], fixture.lineID, code)
			get := fixture.handler.GetInvoice().With(GetInvoiceParams{InvoiceID: invoiceID})
			response = s.dependencyRequest(get, http.MethodGet, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var invoice api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &invoice))
			s.requireVisibleDependencyIssue(invoice, fixture.lineID, code)

			// when: the customer deletes without repairing the dependency, subsequent reads must retain the deletion.
			response = s.dependencyRequest(fixture.handler.DeleteInvoice().With(DeleteInvoiceParams{InvoiceID: invoiceID}), http.MethodDelete, nil)
			s.Require().Equal(http.StatusNoContent, response.Code, response.Body.String())
			response = s.dependencyRequest(get, http.MethodGet, nil)
			s.Require().Equal(http.StatusOK, response.Code, response.Body.String())
			var deleted api.Invoice
			s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &deleted))
			s.Equal(invoiceID, deleted.Id)
			s.Equal(string(billing.StandardInvoiceStatusDeleted), deleted.StatusDetails.ExtendedStatus)
			s.NotNil(deleted.DeletedAt)
			s.Nil(deleted.StatusDetails.AvailableActions.Advance)
		})
	}
}

type legacyDependencyHTTPFixture struct {
	handler    *handler
	customerID string
	lineID     string
	feature    feature.Feature
	meter      meter.Meter
	period     timeutil.ClosedPeriod
}

func (s *LegacyInvoiceDependencyHTTPSuite) newDependencyHTTPFixture() legacyDependencyHTTPFixture {
	s.T().Helper()
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("legacy-dependency-http")
	app := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, app.GetID(), billingtest.WithManualApproval(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")))
	cust := s.CreateTestCustomer(ns, "legacy-http-subject")
	s.Require().NotNil(cust)
	testFeature := s.SetupApiRequestsTotalFeature(ctx, ns)
	s.T().Cleanup(testFeature.Cleanup)
	meters, err := s.MeterAdapter.ListMeters(ctx, meter.ListMetersParams{Namespace: ns})
	s.Require().NoError(err)
	s.Require().Len(meters.Items, 1)
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.FreezeTime(period.From)
	defer clock.UnFreeze()
	created, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(), Currency: currencyx.FiatCode("USD"),
		Lines: []billing.GatheringLine{{GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: "usage"}),
			Engine:          billing.LineEngineTypeInvoice, ManagedBy: billing.ManuallyManagedLine,
			ServicePeriod: period, InvoiceAt: period.To, FeatureKey: testFeature.Feature.Key,
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)}),
		}}},
	})
	s.Require().NoError(err)
	s.Require().Len(created.Lines, 1)
	s.Equal(billing.LineEngineTypeInvoice, created.Lines[0].Engine)
	s.Nil(created.Lines[0].ChargeID)
	s.MockStreamingConnector.AddSimpleEvent(meters.Items[0].Key, 7, period.From.Add(time.Hour))
	h := &handler{
		service: s.BillingService, appService: s.AppService,
		namespaceDecoder: namespacedriver.StaticNamespaceDecoder(ns), logger: testutils.NewLogger(s.T()),
	}
	return legacyDependencyHTTPFixture{
		handler:    h,
		customerID: cust.ID,
		lineID:     created.Lines[0].ID,
		feature:    testFeature.Feature,
		meter:      meters.Items[0],
		period:     period,
	}
}
