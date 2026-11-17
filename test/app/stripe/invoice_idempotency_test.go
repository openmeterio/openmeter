package appstripe

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// idempotentStripeInvoices stands in for the part of Stripe the suite-wide mock hides: a create
// request is answered with the response stored for its idempotency key, even when the invoice in
// that response has been deleted since.
type idempotentStripeInvoices struct {
	createdByKey  map[string]*stripe.Invoice
	deleted       map[string]bool
	createKeys    []string
	createFailure error
}

func (s *StripeInvoiceTestSuite) stubIdempotentStripeInvoices() *idempotentStripeInvoices {
	fake := &idempotentStripeInvoices{
		createdByKey: map[string]*stripe.Invoice{},
		deleted:      map[string]bool{},
	}

	errNoSuchInvoice := errors.New("no such invoice")

	s.StripeAppClient.On("CreateInvoice", mock.Anything).Unset()

	var createInvoice *mock.Call
	createInvoice = s.StripeAppClient.
		On("CreateInvoice", mock.Anything).
		Run(func(args mock.Arguments) {
			key := args.Get(0).(stripeclient.CreateInvoiceInput).IdempotencyKey()
			fake.createKeys = append(fake.createKeys, key)

			if fake.createFailure != nil {
				createInvoice.ReturnArguments = mock.Arguments{(*stripe.Invoice)(nil), fake.createFailure}
				fake.createFailure = nil

				return
			}

			if _, found := fake.createdByKey[key]; !found {
				fake.createdByKey[key] = &stripe.Invoice{
					ID:       fmt.Sprintf("in_%d", len(fake.createdByKey)+1),
					Status:   stripe.InvoiceStatusDraft,
					Customer: &stripe.Customer{ID: defaultStripeCustomerID},
					Currency: "USD",
					Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
				}
			}

			createInvoice.ReturnArguments = mock.Arguments{fake.createdByKey[key], nil}
		})

	var getInvoice *mock.Call
	getInvoice = s.StripeAppClient.
		On("GetInvoice", mock.Anything).
		Run(func(args mock.Arguments) {
			stripeInvoiceID := args.Get(0).(stripeclient.GetInvoiceInput).StripeInvoiceID

			invoice, found := lo.Find(lo.Values(fake.createdByKey), func(invoice *stripe.Invoice) bool {
				return invoice.ID == stripeInvoiceID
			})
			if !found || fake.deleted[stripeInvoiceID] {
				getInvoice.ReturnArguments = mock.Arguments{(*stripe.Invoice)(nil), errNoSuchInvoice}

				return
			}

			getInvoice.ReturnArguments = mock.Arguments{invoice, nil}
		})

	s.StripeAppClient.
		On("DeleteInvoice", mock.Anything).
		Run(func(args mock.Arguments) {
			fake.deleted[args.Get(0).(stripeclient.DeleteInvoiceInput).StripeInvoiceID] = true
		}).
		Return(nil)

	var addInvoiceLines *mock.Call
	addInvoiceLines = s.StripeAppClient.
		On("AddInvoiceLines", mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(0).(stripeclient.AddInvoiceLinesInput)

			if fake.deleted[input.StripeInvoiceID] {
				addInvoiceLines.ReturnArguments = mock.Arguments{[]stripeclient.StripeInvoiceItemWithLineID(nil), errNoSuchInvoice}

				return
			}

			addInvoiceLines.ReturnArguments = mock.Arguments{
				lo.Map(input.Lines, func(line *stripe.InvoiceItemParams, idx int) stripeclient.StripeInvoiceItemWithLineID {
					return stripeclient.StripeInvoiceItemWithLineID{
						InvoiceItem: &stripe.InvoiceItem{
							ID:       fmt.Sprintf("ii_%s_%d", input.StripeInvoiceID, idx),
							Metadata: line.Metadata,
						},
						LineID: fmt.Sprintf("il_%s_%d", input.StripeInvoiceID, idx),
					}
				}),
				nil,
			}
		})

	return fake
}

func (s *StripeInvoiceTestSuite) TestCreateInvoiceIdempotencyKeyChangesAfterStripeInvoiceIsDeleted() {
	ctx := s.T().Context()
	namespace := "ns-invoice-idempotency-deleted"
	period := emptyInvoiceModeServicePeriod()

	clock.FreezeTime(period.To.Add(time.Minute))
	defer clock.UnFreeze()

	cust := s.setupEmptyInvoiceModeCustomer(ctx, namespace, billing.EmptyInvoiceModeOmit)
	defer s.StripeAppClient.Restore()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, namespace)
	defer apiRequestsTotal.Cleanup()

	stripeInvoices := s.stubIdempotentStripeInvoices()

	// given: a draft with a $10 flat fee and a usage line without usage, synced to Stripe
	invoice := s.createEmptyInvoiceModeDraft(ctx, cust,
		emptyInvoiceModeFlatFeeLine(namespace, period, 10),
		emptyInvoiceModeUsageLine(period, apiRequestsTotal.Feature.Key),
	)
	s.Require().Equal("in_1", invoice.ExternalIDs.Invoicing)

	// given: the flat fee is removed, so the Stripe invoice is deleted
	invoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
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
	s.Require().True(stripeInvoices.deleted["in_1"])
	s.Require().Empty(invoice.ExternalIDs.Invoicing)

	// when: a later edit adds a $20 fee, so the invoice has to be created in Stripe again
	invoice, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      invoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(i *billing.StandardInvoice) error {
			i.Lines.Append(billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
				Namespace:     namespace,
				InvoiceID:     i.ID,
				Period:        period,
				InvoiceAt:     period.To,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "Added fee",
				PerUnitAmount: alpacadecimal.NewFromInt(20),
				Currency:      currencyx.FiatCode(currency.USD),
				PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			}))

			return nil
		},
	})
	s.Require().NoError(err)

	// then: the create request does not reuse the key Stripe answers with the deleted invoice
	s.NotEqual(stripeInvoices.createKeys[0], lo.LastOrEmpty(stripeInvoices.createKeys))

	// then: the invoice is synced to a new Stripe invoice
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Equal("in_2", invoice.ExternalIDs.Invoicing)
	s.Equal(billing.InvoicingAppStatus{}, invoice.InvoicingAppStatus)
}
