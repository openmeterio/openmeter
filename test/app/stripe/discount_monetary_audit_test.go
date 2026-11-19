package appstripe

import (
	"errors"
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// TestMonetaryAuditDiscountIdentitySurvivesUpdate isolates update-result identity
// after real billing calculation/persistence and successful initial Stripe SDK
// synchronization: USD 100 gross less USD 40 discount leaves USD 60 cash due.
//
// appinvoice.go:updateInvoice calls AddLineDiscountExternalID(discount.ID,
// line.ExternalIDs.Invoicing), returning the gross detail's provider ID for the
// discount. MergeIntoInvoice faithfully persists that incorrect association.
// A third synchronization consequently maps both USD -40 and USD 100 to the
// same provider item and leaves the original discount item unreferenced.
// The result must preserve the matched discount stripeLine.ID instead.
//
// The second synchronization assumes success at the existing Stripe client
// mock boundary to inspect its result. It does not send HTTP or teach the
// transport to accept invalid il_ IDs; the known invoice-item endpoint defect
// normally prevents this path from completing. The third call captures its
// plan and returns a sentinel before mutation. This is a masked identity
// defect, not evidence of finalized wrong collection or three partial invoices.
// Stripe distinguishes invoice lines from their underlying invoice items:
// https://docs.stripe.com/api/invoice-line-item/object?api-version=2024-06-20
//
// Reproduce: POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/app/stripe
// -run TestStripeInvoicing/TestMonetaryAuditDiscountIdentitySurvivesUpdate -count=1 -v
func (s *StripeInvoiceTestSuite) TestMonetaryAuditDiscountIdentitySurvivesUpdate() {
	// Given a persisted USD 60 invoice and correct initial provider identities.
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("stripe-discount-identity-monetary-audit")
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
	}
	clock.FreezeTime(period.To.Add(time.Hour))
	defer clock.UnFreeze()
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())
	s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)
	customerEntity := s.CreateTestCustomer(namespace, "synthetic-discount-identity")
	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customerEntity.GetID(),
		Currency: "USD",
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     namespace,
				Period:        period,
				InvoiceAt:     period.To,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "Synthetic discounted service",
				PerUnitAmount: alpacadecimal.NewFromInt(100),
				PaymentTerm:   productcatalog.InArrearsPaymentTerm,
				RateCardDiscounts: billing.Discounts{
					Percentage: &billing.PercentageDiscount{
						PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(40)},
					},
				},
			}),
		}),
	})
	s.Require().NoError(err)
	collected, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
		AsOf:     lo.ToPtr(period.To),
	})
	s.Require().NoError(err)
	s.Require().Len(collected, 1)
	stripeInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: collected[0].GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Equal(60.0, stripeInvoice.Totals.Total.InexactFloat64())
	s.Require().Empty(stripeInvoice.ValidationIssues)
	installedApp, err := s.Fixture.setupApp(ctx, namespace)
	s.Require().NoError(err)
	_, err = s.Fixture.setupAppCustomerData(ctx, installedApp, customerEntity)
	s.Require().NoError(err)
	installedStripeApp, ok := installedApp.(stripeapp.App)
	s.Require().True(ok)

	provider := &stripeMonetaryAuditTransport{
		items:     make(map[string][]*stripe.InvoiceItem),
		responses: make(map[string][]byte),
	}
	previousHTTPClient := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{}).(*stripe.BackendImplementation).HTTPClient
	stripe.SetHTTPClient(&http.Client{Transport: provider})
	defer stripe.SetHTTPClient(previousHTTPClient)
	realStripeApp, err := stripeapp.New(installedStripeApp.Meta, stripeapp.AppConfig{
		Logger:                 omtestutils.NewDiscardLogger(s.T()),
		AppService:             s.AppService,
		BillingService:         s.BillingService,
		StripeAppClientFactory: stripeclient.NewStripeAppClient,
		StripeAppService:       s.AppStripeService,
		SecretService:          s.SecretService,
	})
	s.Require().NoError(err)
	createdResult, err := realStripeApp.UpsertStandardInvoice(ctx, stripeInvoice)
	s.Require().NoError(err)
	s.Require().NoError(createdResult.MergeIntoInvoice(&stripeInvoice))
	s.Require().Equal(int64(6000), provider.invoiceTotal(stripeInvoice.ExternalIDs.Invoicing))
	initialDetails := stripeInvoice.GetLeafLinesWithResolvedTaxConfig()
	s.Require().Len(initialDetails, 1)
	s.Require().Len(initialDetails[0].AmountDiscounts, 1)
	discountID := initialDetails[0].AmountDiscounts[0].ID
	originalDiscountExternalID := initialDetails[0].AmountDiscounts[0].ExternalIDs.Invoicing
	grossExternalID := initialDetails[0].ExternalIDs.Invoicing
	s.Require().NotEmpty(originalDiscountExternalID)
	s.Require().NotEqual(grossExternalID, originalDiscountExternalID)
	providerLines := lo.Map(provider.items[stripeInvoice.ExternalIDs.Invoicing], func(item *stripe.InvoiceItem, _ int) *stripe.InvoiceLineItem {
		return &stripe.InvoiceLineItem{
			ID:          "il_" + item.ID,
			InvoiceItem: item,
			Amount:      item.Amount,
			Metadata:    item.Metadata,
		}
	})

	// When an unchanged update succeeds at the app/client boundary and is merged.
	s.StripeAppClient.On("UpdateInvoice", mock.Anything).
		Twice().Return(&stripe.Invoice{ID: stripeInvoice.ExternalIDs.Invoicing}, nil)
	s.StripeAppClient.On("ListInvoiceLineItems", stripeInvoice.ExternalIDs.Invoicing).
		Twice().Return(providerLines, nil)
	s.StripeAppClient.On("UpdateInvoiceLines", mock.Anything).
		Once().Return([]*stripe.InvoiceItem{}, nil)
	updatedResult, err := installedStripeApp.UpsertStandardInvoice(ctx, stripeInvoice)
	s.Require().NoError(err)
	s.Equal(originalDiscountExternalID, updatedResult.GetLineDiscountExternalIDs()[discountID], "the update result must retain the discount's provider identity")
	s.Require().NoError(updatedResult.MergeIntoInvoice(&stripeInvoice))
	updatedDetails := stripeInvoice.GetLeafLinesWithResolvedTaxConfig()
	s.Equal(originalDiscountExternalID, updatedDetails[0].AmountDiscounts[0].ExternalIDs.Invoicing, "merging must not attach the discount to its gross detail item")

	// Then a third mapping must still target two independent provider items.
	stopBeforeProviderMutation := errors.New("synthetic capture before provider execution")
	var plannedUpdates []*stripeclient.StripeInvoiceItemWithID
	s.StripeAppClient.On("UpdateInvoiceLines", mock.Anything).
		Once().Run(func(args mock.Arguments) {
		plannedUpdates = args.Get(0).(stripeclient.UpdateInvoiceLinesInput).Lines
	}).Return([]*stripe.InvoiceItem(nil), stopBeforeProviderMutation)
	_, err = installedStripeApp.UpsertStandardInvoice(ctx, stripeInvoice)
	s.Require().ErrorIs(err, stopBeforeProviderMutation)
	s.StripeAppClient.AssertExpectations(s.T())
	s.Require().Len(plannedUpdates, 2)
	for _, item := range plannedUpdates {
		s.T().Logf("third sync type=%s target=%s amount=%d", item.Metadata["om_line_type"], item.ID, lo.FromPtr(item.Amount))
	}

	s.NotEqual(plannedUpdates[0].ID, plannedUpdates[1].ID, "the USD 40 discount and USD 100 gross amount must never overwrite each other")
	s.Empty(provider.updateItemIDs, "the mock boundary must not execute invalid invoice-item requests")
	s.Require().Equal(int64(6000), provider.invoiceTotal(stripeInvoice.ExternalIDs.Invoicing))
}
