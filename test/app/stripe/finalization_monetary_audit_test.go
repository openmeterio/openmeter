package appstripe

import (
	"net/http"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v80"

	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// This suite uses the production Stripe client factory for every invoice
// lifecycle operation. Only installation is mocked; HTTP requests from the
// real Stripe SDK are handled by the deterministic transport in this package.
type stripeFinalizationMonetaryAuditSuite struct {
	billingtest.BaseSuite
	Fixture *Fixture
}

func TestStripeFinalizationMonetaryAudit(t *testing.T) {
	suite.Run(t, &stripeFinalizationMonetaryAuditSuite{})
}

func (s *stripeFinalizationMonetaryAuditSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	logger := omtestutils.NewDiscardLogger(s.T())
	secretService, err := secretservice.New(secretservice.Config{Adapter: secretadapter.New()})
	s.Require().NoError(err)

	installationClient := &StripeClientMock{}
	stripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:               s.DBClient,
		AppService:           s.AppService,
		CustomerService:      s.CustomerService,
		SecretService:        secretService,
		WebhookSecretService: secretService,
		StripeClientFactory: func(config stripeclient.StripeClientConfig) (stripeclient.StripeClient, error) {
			return installationClient, nil
		},
		StripeAppClientFactory: stripeclient.NewStripeAppClient,
		Logger:                 logger,
	})
	s.Require().NoError(err)

	webhookURLGenerator, err := appstripeservice.NewBaseURLWebhookURLGenerator("http://localhost:8888")
	s.Require().NoError(err)
	_, err = appstripeservice.New(appstripeservice.Config{
		Adapter:             stripeAdapter,
		AppService:          s.AppService,
		SecretService:       secretService,
		BillingService:      s.BillingService,
		Logger:              logger,
		Publisher:           eventbus.NewMock(s.T()),
		WebhookURLGenerator: webhookURLGenerator,
	})
	s.Require().NoError(err)
	s.Fixture = NewFixture(s.AppService, s.CustomerService, installationClient, &StripeAppClientMock{})
}

// TestApprovalPreservesProviderTotalAndReachesFinalization follows a USD 100
// service with a USD 40 discount through actual collection, persisted draft
// synchronization, approval, and the mandatory issuing synchronization.
//
// Stripe exposes distinct invoice-line IDs (il_) and invoice-item IDs (ii_):
// https://docs.stripe.com/api/invoice-line-item/object?api-version=2024-06-20
// Updating/deleting an invoice item requires its ii_ ID:
// https://docs.stripe.com/api/invoiceitems/update?api-version=2024-06-20
// https://docs.stripe.com/api/invoiceitems/delete?api-version=2024-06-20
// The transport rejects il_ IDs on those endpoints instead of masking the bug.
//
// In appinvoice.go, getStripeUpdateInvoiceItemParams and its discount/credit
// equivalents assign stripeLine.ID; client.UpdateInvoiceLines sends that ID
// to InvoiceItems.Update. The removal list similarly contains line IDs. Keep
// the line IDs for OpenMeter's external references, but address invoice-item
// endpoints with stripeLine.InvoiceItem.ID (or use the invoice-line API).
//
// The unchanged, no-failure case shows that approval fails even without an
// earlier retry. After partial-create retry, the draft may contain a
// duplicate discount, but issuing must reconcile it before finalization. On
// the current implementation the invalid update ID fails before orphan removal
// or finalization, leaving issuing.failed and a retryable receivable. This is
// blocked collection, not evidence that Stripe actually collected USD 20.
//
// Reproduce: POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/app/stripe
// -run TestStripeFinalizationMonetaryAudit -count=1 -v
func (s *stripeFinalizationMonetaryAuditSuite) TestApprovalPreservesProviderTotalAndReachesFinalization() {
	for _, tc := range []struct {
		name            string
		failItemAttempt int
	}{
		{name: "unchanged draft without provider failure"},
		{name: "draft after partial creation retry", failItemAttempt: 2},
	} {
		s.Run(tc.name, func() {
			// Given a real Stripe-backed billing profile and a discounted invoice.
			ctx := s.T().Context()
			namespace := s.GetUniqueNamespace("stripe-finalization-monetary-audit")
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(period.To.Add(time.Hour))
			defer clock.UnFreeze()

			provider := &stripeMonetaryAuditTransport{
				failItemAttempt: tc.failItemAttempt,
				items:           make(map[string][]*stripe.InvoiceItem),
				responses:       make(map[string][]byte),
			}
			previousHTTPClient := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{}).(*stripe.BackendImplementation).HTTPClient
			stripe.SetHTTPClient(&http.Client{Transport: provider})
			defer stripe.SetHTTPClient(previousHTTPClient)

			stripeApp, customerEntity, _, err := s.Fixture.setupAppWithCustomer(ctx, namespace)
			s.Require().NoError(err)
			s.ProvisionBillingProfile(ctx, namespace, stripeApp.GetID(), billingtest.WithManualApproval())
			s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)

			_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
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
						Currency:      "USD",
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(40)},
							},
						},
					}),
				}),
			})
			s.Require().NoError(err)

			collectedInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: customerEntity.GetID(),
				AsOf:     lo.ToPtr(period.To),
			})
			s.Require().NoError(err)
			s.Require().Len(collectedInvoices, 1)
			draftInvoice := collectedInvoices[0]
			if tc.failItemAttempt > 0 {
				s.Require().Equal(billing.StandardInvoiceStatusDraftSyncFailed, draftInvoice.Status)
				draftInvoice, err = s.BillingService.RetryInvoice(ctx, draftInvoice.GetInvoiceID())
				s.Require().NoError(err)
			}

			persistedDraft, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: draftInvoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.Require().Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, persistedDraft.Status)
			s.Require().Equal(60.0, persistedDraft.Totals.Total.InexactFloat64())
			s.Require().NotEmpty(persistedDraft.ExternalIDs.Invoicing)
			s.Require().Empty(persistedDraft.ValidationIssues)
			s.T().Logf("draft provider subtotal=%d local total=%s", provider.invoiceTotal(persistedDraft.ExternalIDs.Invoicing), persistedDraft.Totals.Total)

			// When approval performs the mandatory upsert before finalization.
			_, err = s.BillingService.ApproveInvoice(ctx, persistedDraft.GetInvoiceID())
			s.Require().NoError(err)
			persistedApproved, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: persistedDraft.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.T().Logf("approval status=%s updated item IDs=%v deleted item IDs=%v finalize calls=%d issues=%+v",
				persistedApproved.Status, provider.updateItemIDs, provider.deleteItemIDs, provider.finalizeCalls, persistedApproved.ValidationIssues)

			// Then Stripe should be ready to collect exactly the persisted USD 60.
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, persistedApproved.Status)
			s.Empty(persistedApproved.ValidationIssues)
			s.Equal(1, provider.finalizeCalls, "valid invoice approval must reach provider finalization")
			s.Equal(int64(6000), provider.invoiceTotal(persistedApproved.ExternalIDs.Invoicing), "issuing must reconcile any orphaned discount before collection")
		})
	}
}
