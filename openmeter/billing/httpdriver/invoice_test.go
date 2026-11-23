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
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/oliveagle/jsonpath"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type InvoicingTestSuite struct {
	billingtest.BaseSuite
}

func TestInvoicingTax(t *testing.T) {
	suite.Run(t, new(InvoicingTestSuite))
}

func TestValidateAPIGenericInvoiceDeleteSupportedForGatheringInvoices(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		invoice   billing.GenericInvoice
		wantError bool
	}{
		{
			name: "active usage-based line is rejected",
			invoice: gatheringInvoiceDeleteSupportTestInvoice(
				gatheringInvoiceDeleteSupportTestLine(billing.LineEngineTypeChargeUsageBased, nil),
			),
			wantError: true,
		},
		{
			name: "deleted usage-based line is ignored",
			invoice: gatheringInvoiceDeleteSupportTestInvoice(
				gatheringInvoiceDeleteSupportTestLine(billing.LineEngineTypeChargeUsageBased, &now),
			),
		},
		{
			name: "flat fee line is supported",
			invoice: gatheringInvoiceDeleteSupportTestInvoice(
				gatheringInvoiceDeleteSupportTestLine(billing.LineEngineTypeChargeFlatFee, nil),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := billing.ValidateAPIGenericInvoiceDeleteSupported(tc.invoice)

			if !tc.wantError {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.ErrorIs(t, err, billing.ErrCannotUpdateChargeManagedLine)

			var validationErr billing.ValidationError
			require.ErrorAs(t, err, &validationErr)

			issues, conversionErr := billing.ToValidationIssues(err)
			require.NoError(t, conversionErr)
			require.Len(t, issues, 1)
			require.Equal(t, billing.ErrCannotUpdateChargeManagedLine.Code, issues[0].Code)
			require.Equal(t, billing.ErrCannotUpdateChargeManagedLine.Message, issues[0].Message)
			require.Equal(t, billing.ValidationIssueSeverityCritical, issues[0].Severity)
			require.Equal(t, billing.LineEngineValidationComponent(billing.LineEngineTypeChargeUsageBased), issues[0].Component)
		})
	}
}

func TestValidateAPIInvoiceDeleteSupportedIgnoresDeletedGatheringInvoice(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	invoice := gatheringInvoiceDeleteSupportTestInvoice(
		gatheringInvoiceDeleteSupportTestLine(billing.LineEngineTypeChargeUsageBased, nil),
	)
	invoice.DeletedAt = &now

	err := billing.ValidateAPIInvoiceDeleteSupported(billing.NewInvoice(*invoice))

	require.NoError(t, err)
}

func TestValidateAPIInvoiceDeleteSupportedSkipsStandardInvoices(t *testing.T) {
	t.Parallel()

	err := billing.ValidateAPIInvoiceDeleteSupported(billing.NewInvoice(billing.StandardInvoice{
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{{
			StandardLineBase: billing.StandardLineBase{
				Engine: billing.LineEngineTypeChargeUsageBased,
			},
			UsageBased: &billing.UsageBasedLine{},
		}}),
	}))

	require.NoError(t, err)
}

func TestValidateAPIInvoiceDeleteSupportedRejectsUsageBasedGatheringInvoice(t *testing.T) {
	t.Parallel()

	err := billing.ValidateAPIInvoiceDeleteSupported(billing.NewInvoice(*gatheringInvoiceDeleteSupportTestInvoice(
		gatheringInvoiceDeleteSupportTestLine(billing.LineEngineTypeChargeUsageBased, nil),
	)))

	require.Error(t, err)
	require.ErrorIs(t, err, billing.ErrCannotUpdateChargeManagedLine)
}

func gatheringInvoiceDeleteSupportTestInvoice(lines ...billing.GatheringLine) *billing.GatheringInvoice {
	return &billing.GatheringInvoice{
		Lines: billing.NewGatheringInvoiceLines(lines),
	}
}

func gatheringInvoiceDeleteSupportTestLine(engine billing.LineEngineType, deletedAt *time.Time) billing.GatheringLine {
	return gatheringInvoiceDeleteSupportTestLineWithID("", engine, deletedAt)
}

func gatheringInvoiceDeleteSupportTestLineWithID(id string, engine billing.LineEngineType, deletedAt *time.Time) billing.GatheringLine {
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			Engine: engine,
		},
	}
	line.ID = id
	line.DeletedAt = deletedAt

	return line
}

func (s *InvoicingTestSuite) TestGatheringInvoiceSerialization() {
	namespace := "ns-invoice-serialization"
	ctx := s.T().Context()

	appSandbox := s.InstallSandboxApp(s.T(), namespace)

	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			Currency:     lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, appSandbox.GetID())
	now := clock.Now()

	// Let's provision a gathering invoice with a single flat fee line
	res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Customer:     cust.GetID(),
		Currency:     currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(
				billing.NewFlatFeeLineInput{
					Namespace:     namespace,
					Period:        timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
					InvoiceAt:     now.Add(time.Hour * 24),
					ManagedBy:     billing.ManuallyManagedLine,
					Name:          "Test item - USD",
					PerUnitAmount: alpacadecimal.NewFromFloat(100),
				},
			),
		}),
	})
	s.NoError(err)

	// Let's get the invoice
	invoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
		Invoice: res.Invoice.GetInvoiceID(),
		Expand: billing.InvoiceExpands{
			billing.InvoiceExpandLines,
			billing.InvoiceExpandCalculateGatheringInvoiceWithLiveData,
		},
	})
	s.NoError(err)

	// The invoice should be a standard invoice, with status == gathering
	standardInvoice, err := invoice.AsStandardInvoice()
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusGathering, standardInvoice.Status)

	// Let's serialize the invoice
	apiInvoice, err := MapStandardInvoiceToAPI(standardInvoice, nil)
	s.NoError(err)

	// Let's deserialize the invoice
	apiInvoiceJSON, err := json.MarshalIndent(apiInvoice, "", "  ")
	s.NoError(err)

	s.T().Logf("serialized invoice: %s", string(apiInvoiceJSON))

	var parsedInvoice any
	err = json.Unmarshal(apiInvoiceJSON, &parsedInvoice)
	s.NoError(err)

	expects := []struct {
		Name           string
		Path           string
		Paths          []string
		ExpectError    bool
		ValueValidator func(member any) error
	}{
		// TODO: TypeSpec seem to mark metadata nullable, so this will fail
		//{
		//	Name: "empty metadata should be omitted",
		//	Paths: []string{
		//		"$.lines[0].children[0].metadata",
		//		"$.lines[0].metadata",
		//		"$.metadata",
		//	},
		//	ExpectError: true,
		//},
		{
			Name:        "empty validation issues should be omitted",
			Path:        "$.validationIssues",
			ExpectError: true,
		},
		{
			Name: "empty external ids should be omitted",
			Paths: []string{
				"$.externalIds",
				"$.lines[0].externalIds",
				"$.lines[0].children[0].externalIds",
			},
			ExpectError: true,
		},
		{
			Name: "featureKey should be omitted for flat fee lines not associated with a feature",
			Paths: []string{
				"$.lines[0].featureKey",
				"$.lines[0].children[0].featureKey",
			},
			ExpectError: true,
		},
		{
			Name: "preLinePeriodQuantity and meteredPreLinePeriodQuantity should be omitted when 0",
			Paths: []string{
				"$.lines[0].preLinePeriodQuantity",
				"$.lines[0].meteredPreLinePeriodQuantity",
				"$.lines[0].children[0].preLinePeriodQuantity",
				"$.lines[0].children[0].meteredPreLinePeriodQuantity",
			},
			ExpectError: true,
		},
		{
			Name: "meteredQuantity should be omitted when equal to quantity",
			Paths: []string{
				"$.lines[0].meteredQuantity",
				"$.lines[0].children[0].meteredQuantity",
			},
			ExpectError: true,
		},
		{
			Name: "lineIDs must be present",
			Paths: []string{
				"$.lines[0].id",
				"$.lines[0].children[0].id",
			},
			ValueValidator: func(member any) error {
				strValue, err := toString(member)
				if err != nil {
					return err
				}

				_, err = ulid.Parse(strValue)

				return err
			},
		},
		{
			Name:        "draft until should not be present",
			Path:        "$.draftUntil",
			ExpectError: true,
		},
		{
			Name: "collection at should be present",
			Path: "$.collectionAt",
			ValueValidator: func(member any) error {
				timeString, err := toString(member)
				if err != nil {
					return err
				}

				_, err = time.Parse(time.RFC3339, timeString)

				return err
			},
		},
		{
			Name:        "customer address should not be present if empty",
			Path:        "$.customer.addresses",
			ExpectError: true,
		},
	}

	for _, expect := range expects {
		s.Run(expect.Name, func() {
			paths := expect.Paths
			if expect.Path != "" {
				paths = append(paths, expect.Path)
			}

			for _, path := range paths {
				member, err := jsonpath.JsonPathLookup(parsedInvoice, path)
				if expect.ExpectError {
					s.Error(err, "path: %s", path)
				} else {
					s.NoError(err, "path: %s", path)
					if expect.ValueValidator != nil {
						s.NoError(expect.ValueValidator(member), "path: %s", path)
					}
				}
			}
		})
	}
}

func toString(member any) (string, error) {
	switch v := member.(type) {
	case string:
		return v, nil
	case *string:
		return *v, nil
	default:
		return "", fmt.Errorf("expected string, got %T", member)
	}
}

func (s *InvoicingTestSuite) TestSimulateInvoiceHTTP() {
	namespace := "ns-simulate-invoice-http"
	ctx := s.T().Context()

	// given:
	// - a customer with a billing profile
	// - a simulate invoice request with one flat price line
	appSandbox := s.InstallSandboxApp(s.T(), namespace)

	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Key:      lo.ToPtr("test-customer"),
			Name:     "Test Customer",
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
		},
	})
	s.Require().NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, appSandbox.GetID())

	price := apiv1models.RateCardUsageBasedPrice{}
	s.NoError(price.FromFlatPriceWithPaymentTerm(apiv1models.FlatPriceWithPaymentTerm{
		Amount: "10",
		Type:   apiv1models.FlatPriceWithPaymentTermTypeFlat,
	}))

	period := invoiceLineTestPeriod()
	body, err := json.Marshal(apiv1models.InvoiceSimulationInput{
		Currency: apiv1models.CurrencyCode(currency.USD),
		Lines: []apiv1models.InvoiceSimulationLine{
			{
				Name:      "simulated flat fee",
				Period:    apiv1models.Period(period),
				InvoiceAt: period.To,
				RateCard:  &apiv1models.InvoiceUsageBasedRateCard{Price: &price},
			},
		},
	})
	s.NoError(err)

	h := &handler{
		service:          s.BillingService,
		namespaceDecoder: namespacedriver.StaticNamespaceDecoder(namespace),
		taxCodeResolver:  s.TaxCodeResolver,
	}

	// when:
	// - the request goes through the HTTP handler and the billing service
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/billing/customers/"+cust.ID+"/invoices/simulate", bytes.NewReader(body))
	h.SimulateInvoice().With(SimulateInvoiceParams{CustomerID: cust.ID}).ServeHTTP(w, r)

	// then:
	// - the service rates the line through the invoicing engine and returns the simulated invoice
	s.Require().Equal(http.StatusOK, w.Code, w.Body.String())

	var invoice apiv1models.Invoice
	s.NoError(json.Unmarshal(w.Body.Bytes(), &invoice))
	s.Len(lo.FromPtr(invoice.Lines), 1)
	s.Equal(10.0, alpacadecimal.RequireFromString(invoice.Totals.Total).InexactFloat64())
}

func (s *InvoicingTestSuite) TestSimulateInvoiceResolvesStripeCodeReadOnly() {
	namespace := "ns-simulate-taxconfig"
	ctx := s.T().Context()

	// given: a customer with a billing profile and a known Stripe-mapped tax code
	appSandbox := s.InstallSandboxApp(s.T(), namespace)

	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Key:      lo.ToPtr("test-customer"),
			Name:     "Test Customer",
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
		},
	})
	s.Require().NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, appSandbox.GetID())

	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	h := &handler{
		service:          s.BillingService,
		namespaceDecoder: namespacedriver.StaticNamespaceDecoder(namespace),
		logger:           testutils.NewDiscardLogger(s.T()),
		taxCodeResolver:  s.TaxCodeResolver,
	}

	period := invoiceLineTestPeriod()
	price := apiv1models.RateCardUsageBasedPrice{}
	s.NoError(price.FromFlatPriceWithPaymentTerm(apiv1models.FlatPriceWithPaymentTerm{
		Amount: "10",
		Type:   apiv1models.FlatPriceWithPaymentTermTypeFlat,
	}))

	// The simulation reads: it must not create the Tax Code an unknown stripe.code would
	// resolve to, so deleted rows included, the count can't move.

	s.Run("known stripe code renders its id and stripe code", func() {
		// given: a line with only stripe.code
		before := s.taxCodeCount(namespace)
		// when: the invoice is simulated
		body, err := json.Marshal(apiv1models.InvoiceSimulationInput{
			Currency: apiv1models.CurrencyCode(currency.USD),
			Lines:    []apiv1models.InvoiceSimulationLine{s.simulationTaxConfigLine(&price, period, &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: "txcd_10000000"}})},
		})
		s.Require().NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/billing/customers/"+cust.ID+"/invoices/simulate", bytes.NewReader(body))
		h.SimulateInvoice().With(SimulateInvoiceParams{CustomerID: cust.ID}).ServeHTTP(w, r)

		// then: the line shows the tax code's id and stripe code, and nothing was created
		s.Require().Equal(http.StatusOK, w.Code, w.Body.String())

		var invoice apiv1models.Invoice
		s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &invoice))
		s.Require().Len(lo.FromPtr(invoice.Lines), 1)
		s.Require().NotNil((*invoice.Lines)[0].TaxConfig)
		s.Equal(&apiv1models.TaxConfig{
			Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, (*invoice.Lines)[0].TaxConfig)

		s.Equal(before, s.taxCodeCount(namespace))
	})

	s.Run("unknown stripe code creates nothing and renders no stripe code", func() {
		// given: a line with a stripe.code no tax code maps to
		before := s.taxCodeCount(namespace)

		// when: the invoice is simulated
		body, err := json.Marshal(apiv1models.InvoiceSimulationInput{
			Currency: apiv1models.CurrencyCode(currency.USD),
			Lines:    []apiv1models.InvoiceSimulationLine{s.simulationTaxConfigLine(&price, period, &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: "txcd_99000000"}})},
		})
		s.Require().NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/billing/customers/"+cust.ID+"/invoices/simulate", bytes.NewReader(body))
		h.SimulateInvoice().With(SimulateInvoiceParams{CustomerID: cust.ID}).ServeHTTP(w, r)

		// then: the request succeeds, the line carries no tax code reference and no stripe code
		s.Require().Equal(http.StatusOK, w.Code, w.Body.String())

		var invoice apiv1models.Invoice
		s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &invoice))
		s.Require().Len(lo.FromPtr(invoice.Lines), 1)
		s.Require().NotNil((*invoice.Lines)[0].TaxConfig)
		s.Nil((*invoice.Lines)[0].TaxConfig.Stripe)
		s.Nil((*invoice.Lines)[0].TaxConfig.TaxCodeId)

		s.Equal(before, s.taxCodeCount(namespace))
	})

	s.Run("tax code id wins over stripe code", func() {
		// given: a line echoing both the id and a foreign stripe code
		// when: the invoice is simulated
		body, err := json.Marshal(apiv1models.InvoiceSimulationInput{
			Currency: apiv1models.CurrencyCode(currency.USD),
			Lines: []apiv1models.InvoiceSimulationLine{s.simulationTaxConfigLine(&price, period, &apiv1models.TaxConfig{
				Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_99000000"},
				TaxCodeId: lo.ToPtr(taxCode.ID),
			})},
		})
		s.Require().NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/billing/customers/"+cust.ID+"/invoices/simulate", bytes.NewReader(body))
		h.SimulateInvoice().With(SimulateInvoiceParams{CustomerID: cust.ID}).ServeHTTP(w, r)

		// then: the id wins and the stripe code comes from the tax code's mapping
		s.Require().Equal(http.StatusOK, w.Code, w.Body.String())

		var invoice apiv1models.Invoice
		s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &invoice))
		s.Require().Len(lo.FromPtr(invoice.Lines), 1)
		s.Require().NotNil((*invoice.Lines)[0].TaxConfig)
		s.Equal(lo.ToPtr(taxCode.ID), (*invoice.Lines)[0].TaxConfig.TaxCodeId)
		s.Equal(&apiv1models.StripeTaxConfig{Code: "txcd_10000000"}, (*invoice.Lines)[0].TaxConfig.Stripe)
	})

	s.Run("mismatched taxConfig and rateCard.taxConfig is rejected", func() {
		// given: a line whose deprecated taxConfig differs from its rate card's
		line := s.simulationTaxConfigLine(&price, period, &apiv1models.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)})
		line.TaxConfig = &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: "txcd_10000000"}}

		// when: the invoice is simulated
		body, err := json.Marshal(apiv1models.InvoiceSimulationInput{
			Currency: apiv1models.CurrencyCode(currency.USD),
			Lines:    []apiv1models.InvoiceSimulationLine{line},
		})
		s.Require().NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/billing/customers/"+cust.ID+"/invoices/simulate", bytes.NewReader(body))
		h.SimulateInvoice().With(SimulateInvoiceParams{CustomerID: cust.ID}).ServeHTTP(w, r)

		// then: the request fails with a validation error naming the mismatch
		s.Require().Equal(http.StatusBadRequest, w.Code, w.Body.String())
		s.Contains(w.Body.String(), "taxConfig must be equal to rateCard.taxConfig")
	})

	s.Run("malformed stripe code is rejected", func() {
		// when: the invoice is simulated with a malformed stripe code
		body, err := json.Marshal(apiv1models.InvoiceSimulationInput{
			Currency: apiv1models.CurrencyCode(currency.USD),
			Lines:    []apiv1models.InvoiceSimulationLine{s.simulationTaxConfigLine(&price, period, &apiv1models.TaxConfig{Stripe: &apiv1models.StripeTaxConfig{Code: "invalid_code"}})},
		})
		s.Require().NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/billing/customers/"+cust.ID+"/invoices/simulate", bytes.NewReader(body))
		h.SimulateInvoice().With(SimulateInvoiceParams{CustomerID: cust.ID}).ServeHTTP(w, r)

		// then: the request fails with a validation error
		s.Require().Equal(http.StatusBadRequest, w.Code, w.Body.String())
	})
}

func (s *InvoicingTestSuite) taxCodeCount(namespace string) int {
	count, err := s.DBClient.TaxCode.Query().Where(taxcodedb.Namespace(namespace)).Count(s.T().Context())
	s.Require().NoError(err)

	return count
}

func (s *InvoicingTestSuite) simulationTaxConfigLine(price *apiv1models.RateCardUsageBasedPrice, period timeutil.ClosedPeriod, taxConfig *apiv1models.TaxConfig) apiv1models.InvoiceSimulationLine {
	return apiv1models.InvoiceSimulationLine{
		Name:      "simulated flat fee",
		Period:    apiv1models.Period(period),
		InvoiceAt: period.To,
		RateCard:  &apiv1models.InvoiceUsageBasedRateCard{Price: price, TaxConfig: taxConfig},
	}
}
