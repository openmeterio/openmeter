package appstripe

import (
	"context"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const (
	emptyInvoiceModeStripeInvoiceID = "in_empty_invoice_mode"
	emptyInvoiceModeStripeLineID    = "il_flat_fee"
	emptyInvoiceModeFlatFeeName     = "Flat fee"
)

var emptyInvoiceModes = []billing.EmptyInvoiceMode{
	billing.EmptyInvoiceModeCreate,
	billing.EmptyInvoiceModeOmit,
}

func (s *StripeInvoiceTestSuite) TestEmptyInvoiceModeDraftSync() {
	for _, mode := range emptyInvoiceModes {
		s.Run(string(mode), func() {
			ctx := s.T().Context()
			namespace := "ns-empty-invoice-mode-draft-sync-" + string(mode)
			period := emptyInvoiceModeServicePeriod()

			clock.FreezeTime(period.From)
			defer clock.UnFreeze()

			// given: a single usage-based line without usage at a $1 unit price
			cust := s.setupEmptyInvoiceModeCustomer(ctx, namespace, mode)
			defer s.StripeAppClient.Restore()

			apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, namespace)
			defer apiRequestsTotal.Cleanup()

			_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: cust.GetID(),
				Currency: currencyx.FiatCode(currency.USD),
				Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
					emptyInvoiceModeUsageLine(period, apiRequestsTotal.Feature.Key),
				}),
			})
			s.Require().NoError(err)

			clock.FreezeTime(period.To.Add(time.Minute))

			// when: the draft invoice is synced to Stripe
			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: cust.GetID(),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			invoice := invoices[0]

			s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
			s.Empty(invoice.ValidationIssues)
			s.Equal(float64(0), invoice.Totals.Total.InexactFloat64())

			switch mode {
			case billing.EmptyInvoiceModeCreate:
				// then: the empty invoice still gets a Stripe invoice
				s.StripeAppClient.AssertNumberOfCalls(s.T(), "CreateInvoice", 1)
				s.Equal(emptyInvoiceModeStripeInvoiceID, invoice.ExternalIDs.Invoicing)
			case billing.EmptyInvoiceModeOmit:
				// then: no Stripe invoice is created and the invoice has no Stripe reference
				s.StripeAppClient.AssertNumberOfCalls(s.T(), "CreateInvoice", 0)
				s.Empty(invoice.ExternalIDs.Invoicing)
			}
		})
	}
}

func (s *StripeInvoiceTestSuite) TestEmptyInvoiceModeDraftEdit() {
	for _, mode := range emptyInvoiceModes {
		s.Run(string(mode), func() {
			ctx := s.T().Context()
			namespace := "ns-empty-invoice-mode-draft-edit-" + string(mode)
			period := emptyInvoiceModeServicePeriod()

			clock.FreezeTime(period.From)
			defer clock.UnFreeze()

			// given: a draft invoice waiting for manual approval with a $10 flat fee and a
			// usage-based line without usage, already synced to Stripe
			cust := s.setupEmptyInvoiceModeCustomer(ctx, namespace, mode)
			defer s.StripeAppClient.Restore()

			apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, namespace)
			defer apiRequestsTotal.Cleanup()

			s.StripeAppClient.
				On("AddInvoiceLines", mock.Anything).
				Return([]stripeclient.StripeInvoiceItemWithLineID{}, nil)

			_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: cust.GetID(),
				Currency: currencyx.FiatCode(currency.USD),
				Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
					billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
						Namespace:     namespace,
						Period:        period,
						InvoiceAt:     period.To,
						ManagedBy:     billing.ManuallyManagedLine,
						Name:          emptyInvoiceModeFlatFeeName,
						PerUnitAmount: alpacadecimal.NewFromInt(10),
						Currency:      currencyx.FiatCode(currency.USD),
						PaymentTerm:   productcatalog.InArrearsPaymentTerm,
					}),
					emptyInvoiceModeUsageLine(period, apiRequestsTotal.Feature.Key),
				}),
			})
			s.Require().NoError(err)

			clock.FreezeTime(period.To.Add(time.Minute))

			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: cust.GetID(),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			invoice := invoices[0]

			s.Require().Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
			s.Require().Equal(float64(10), invoice.Totals.Total.InexactFloat64())
			s.StripeAppClient.AssertNumberOfCalls(s.T(), "CreateInvoice", 1)
			s.StripeAppClient.AssertNumberOfCalls(s.T(), "AddInvoiceLines", 1)
			s.Require().Equal(emptyInvoiceModeStripeInvoiceID, invoice.ExternalIDs.Invoicing)

			stripeInvoice := &stripe.Invoice{
				ID:       emptyInvoiceModeStripeInvoiceID,
				Customer: &stripe.Customer{ID: defaultStripeCustomerID},
				Currency: "USD",
				Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
			}

			s.StripeAppClient.
				On("UpdateInvoice", mock.Anything).
				Return(stripeInvoice, nil)
			s.StripeAppClient.
				On("ListInvoiceLineItems", emptyInvoiceModeStripeInvoiceID).
				Return([]*stripe.InvoiceLineItem{{ID: emptyInvoiceModeStripeLineID}}, nil)
			s.StripeAppClient.
				On("RemoveInvoiceLines", mock.Anything).
				Return(nil)
			s.StripeAppClient.
				On("DeleteInvoice", mock.Anything).
				Return(nil)

			// when: the flat fee is removed, leaving only the zero usage line
			invoice, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
				Invoice:      invoice.GetInvoiceID(),
				ChangeSource: billing.ChangeSourceAPIRequest,
				EditFn: func(i *billing.StandardInvoice) error {
					flatFee, found := lo.Find(i.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
						return line.Name == emptyInvoiceModeFlatFeeName
					})
					if !found {
						return errors.New("flat fee line not found")
					}

					flatFee.DeletedAt = lo.ToPtr(clock.Now())

					return nil
				},
			})
			s.Require().NoError(err)

			s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
			s.Equal(1, invoice.Lines.NonDeletedLineCount())
			s.Equal(float64(0), invoice.Totals.Total.InexactFloat64())

			switch mode {
			case billing.EmptyInvoiceModeCreate:
				// then: the Stripe invoice is kept and only the flat fee line is removed from it
				s.StripeAppClient.AssertNumberOfCalls(s.T(), "UpdateInvoice", 1)
				s.StripeAppClient.AssertCalled(s.T(), "RemoveInvoiceLines", stripeclient.RemoveInvoiceLinesInput{
					StripeInvoiceID: emptyInvoiceModeStripeInvoiceID,
					Lines:           []string{emptyInvoiceModeStripeLineID},
				})
				s.StripeAppClient.AssertNumberOfCalls(s.T(), "DeleteInvoice", 0)
				s.Equal(emptyInvoiceModeStripeInvoiceID, invoice.ExternalIDs.Invoicing)
			case billing.EmptyInvoiceModeOmit:
				// then: the Stripe invoice is deleted and its reference is removed from the invoice
				s.StripeAppClient.AssertCalled(s.T(), "DeleteInvoice", stripeclient.DeleteInvoiceInput{
					StripeInvoiceID: emptyInvoiceModeStripeInvoiceID,
				})
				s.StripeAppClient.AssertNumberOfCalls(s.T(), "DeleteInvoice", 1)
				s.Empty(invoice.ExternalIDs.Invoicing)
			}
		})
	}
}

// setupEmptyInvoiceModeCustomer creates a Stripe-billed customer whose profile uses the given
// empty invoice mode and keeps invoices in draft until manually approved. Stripe call history
// is reset because the suite shares a single client mock.
func (s *StripeInvoiceTestSuite) setupEmptyInvoiceModeCustomer(ctx context.Context, namespace string, mode billing.EmptyInvoiceMode) *customer.Customer {
	s.T().Helper()

	stripeApp, err := s.Fixture.setupApp(ctx, namespace)
	s.Require().NoError(err)

	cust := s.CreateTestCustomer(namespace, "test")

	customerData, err := s.Fixture.setupAppCustomerData(ctx, stripeApp, cust)
	s.Require().NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, stripeApp.GetID(),
		billingtest.WithManualApproval(),
		billingtest.WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
			profile.WorkflowConfig.Invoicing.EmptyInvoiceMode = mode
		}),
	)
	s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)

	s.StripeAppClient.Calls = nil

	s.StripeAppClient.
		On("GetCustomer", customerData.StripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: customerData.StripeCustomerID,
			DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
				ID: "pm_123",
				BillingAddress: &models.Address{
					Country: lo.ToPtr(models.CountryCode("US")),
				},
			},
		}, nil)

	s.StripeAppClient.
		On("CreateInvoice", mock.Anything).
		Return(&stripe.Invoice{
			ID:       emptyInvoiceModeStripeInvoiceID,
			Customer: &stripe.Customer{ID: customerData.StripeCustomerID},
			Currency: "USD",
			Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
		}, nil)

	return cust
}

func emptyInvoiceModeServicePeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: time.Date(2024, 9, 2, 12, 13, 14, 0, time.UTC),
		To:   time.Date(2024, 9, 3, 12, 13, 14, 0, time.UTC),
	}
}

func emptyInvoiceModeUsageLine(period timeutil.ClosedPeriod, featureKey string) billing.GatheringLine {
	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Usage",
			}),
			ServicePeriod: period,
			InvoiceAt:     period.To,
			ManagedBy:     billing.ManuallyManagedLine,
			FeatureKey:    featureKey,
			Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			})),
		},
	}
}
