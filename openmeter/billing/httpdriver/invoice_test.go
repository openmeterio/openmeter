package httpdriver

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/oliveagle/jsonpath"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type InvoicingTestSuite struct {
	billingtest.BaseSuite
}

func TestInvoicingTax(t *testing.T) {
	suite.Run(t, new(InvoicingTestSuite))
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
		Customer: cust.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(
				billing.NewFlatFeeLineInput{
					Namespace:     namespace,
					Period:        billing.Period{Start: now, End: now.Add(time.Hour * 24)},
					InvoiceAt:     now.Add(time.Hour * 24),
					ManagedBy:     billing.ManuallyManagedLine,
					Name:          "Test item - USD",
					PerUnitAmount: alpacadecimal.NewFromFloat(100),
				},
			),
		},
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
	apiInvoice, err := MapStandardInvoiceToAPI(standardInvoice)
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
