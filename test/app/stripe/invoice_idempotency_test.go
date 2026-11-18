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
	responseByKey map[string]stripeCreateResponse
	created       int
	deleted       map[string]bool
	createKeys    []string
	// failNext makes the next request of a client method fail. A failed create is stored for its
	// key like any other response.
	failNext map[string]error
}

type stripeCreateResponse struct {
	invoice *stripe.Invoice
	err     error
}

var errNoSuchStripeInvoice = errors.New("no such invoice")

func (f *idempotentStripeInvoices) takeFailure(method string) error {
	err := f.failNext[method]
	delete(f.failNext, method)

	return err
}

func (f *idempotentStripeInvoices) create(key string) (*stripe.Invoice, error) {
	f.createKeys = append(f.createKeys, key)

	response, stored := f.responseByKey[key]
	if !stored {
		response.err = f.takeFailure("CreateInvoice")

		if response.err == nil {
			f.created++
			response.invoice = &stripe.Invoice{
				ID:       fmt.Sprintf("in_%d", f.created),
				Status:   stripe.InvoiceStatusDraft,
				Customer: &stripe.Customer{ID: defaultStripeCustomerID},
				Currency: "USD",
				Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
			}
		}

		f.responseByKey[key] = response
	}

	return response.invoice, response.err
}

func (f *idempotentStripeInvoices) get(stripeInvoiceID string) (*stripe.Invoice, error) {
	if err := f.takeFailure("GetInvoice"); err != nil {
		return nil, err
	}

	if f.deleted[stripeInvoiceID] {
		return nil, stripeclient.NewStripeInvoiceNotFoundError(errNoSuchStripeInvoice)
	}

	return &stripe.Invoice{ID: stripeInvoiceID, Status: stripe.InvoiceStatusDraft}, nil
}

func (f *idempotentStripeInvoices) delete(stripeInvoiceID string) error {
	if err := f.takeFailure("DeleteInvoice"); err != nil {
		return err
	}

	f.deleted[stripeInvoiceID] = true

	return nil
}

func (f *idempotentStripeInvoices) addLines(input stripeclient.AddInvoiceLinesInput) ([]stripeclient.StripeInvoiceItemWithLineID, error) {
	if f.deleted[input.StripeInvoiceID] {
		return nil, errNoSuchStripeInvoice
	}

	return lo.Map(input.Lines, func(line *stripe.InvoiceItemParams, idx int) stripeclient.StripeInvoiceItemWithLineID {
		return stripeclient.StripeInvoiceItemWithLineID{
			InvoiceItem: &stripe.InvoiceItem{
				ID:       fmt.Sprintf("ii_%s_%d", input.StripeInvoiceID, idx),
				Metadata: line.Metadata,
			},
			LineID: fmt.Sprintf("il_%s_%d", input.StripeInvoiceID, idx),
		}
	}), nil
}

// respondWith lets a stubbed call compute its return values from the arguments it was called with.
func respondWith(call *mock.Call, respond func(mock.Arguments) mock.Arguments) {
	call.Run(func(args mock.Arguments) {
		call.ReturnArguments = respond(args)
	})
}

func (s *StripeInvoiceTestSuite) stubIdempotentStripeInvoices() *idempotentStripeInvoices {
	fake := &idempotentStripeInvoices{
		responseByKey: map[string]stripeCreateResponse{},
		deleted:       map[string]bool{},
		failNext:      map[string]error{},
	}

	s.StripeAppClient.On("CreateInvoice", mock.Anything).Unset()

	respondWith(s.StripeAppClient.On("CreateInvoice", mock.Anything), func(args mock.Arguments) mock.Arguments {
		invoice, err := fake.create(args.Get(0).(stripeclient.CreateInvoiceInput).IdempotencyKey())

		return mock.Arguments{invoice, err}
	})

	respondWith(s.StripeAppClient.On("GetInvoice", mock.Anything), func(args mock.Arguments) mock.Arguments {
		invoice, err := fake.get(args.Get(0).(stripeclient.GetInvoiceInput).StripeInvoiceID)

		return mock.Arguments{invoice, err}
	})

	respondWith(s.StripeAppClient.On("DeleteInvoice", mock.Anything), func(args mock.Arguments) mock.Arguments {
		return mock.Arguments{fake.delete(args.Get(0).(stripeclient.DeleteInvoiceInput).StripeInvoiceID)}
	})

	respondWith(s.StripeAppClient.On("AddInvoiceLines", mock.Anything), func(args mock.Arguments) mock.Arguments {
		lines, err := fake.addLines(args.Get(0).(stripeclient.AddInvoiceLinesInput))

		return mock.Arguments{lines, err}
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

	// then: reading back the replayed invoice fails, which increments the generation
	generation, err := s.AppStripeService.GetLatestInvoiceGeneration(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(1, generation)

	// then: the create request is repeated with a key Stripe has no stored response for
	s.Require().Len(stripeInvoices.createKeys, 3)
	s.Equal(stripeInvoices.createKeys[0], stripeInvoices.createKeys[1])
	s.NotEqual(stripeInvoices.createKeys[0], stripeInvoices.createKeys[2])

	// then: the invoice is synced to a new Stripe invoice
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Equal("in_2", invoice.ExternalIDs.Invoicing)
	s.Equal(billing.InvoicingAppStatus{}, invoice.InvoicingAppStatus)
}

func (s *StripeInvoiceTestSuite) TestCreateInvoiceIdempotencyKeyChangesAfterFailedCreate() {
	ctx := s.T().Context()
	namespace := "ns-invoice-idempotency-failed-create"
	period := emptyInvoiceModeServicePeriod()

	clock.FreezeTime(period.To.Add(time.Minute))
	defer clock.UnFreeze()

	cust := s.setupEmptyInvoiceModeCustomer(ctx, namespace, billing.EmptyInvoiceModeCreate)
	defer s.StripeAppClient.Restore()

	stripeInvoices := s.stubIdempotentStripeInvoices()

	// given: Stripe rejects the first create request, and would keep answering its key the same way
	stripeInvoices.failNext["CreateInvoice"] = errors.New("rate limited")

	// when: the draft is synced to Stripe
	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			emptyInvoiceModeFlatFeeLine(namespace, period, 10),
		}),
	})
	s.Require().NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]

	// then: the sync fails and the generation is incremented
	s.Require().Equal(billing.StandardInvoiceStatusDraftSyncFailed, invoice.Status)
	s.Require().Empty(invoice.ExternalIDs.Invoicing)

	generation, err := s.AppStripeService.GetLatestInvoiceGeneration(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Equal(1, generation)

	// when: the sync is retried
	invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)

	// then: the create request uses a new key and the invoice is synced
	s.Require().Len(stripeInvoices.createKeys, 2)
	s.NotEqual(stripeInvoices.createKeys[0], stripeInvoices.createKeys[1])
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Equal("in_1", invoice.ExternalIDs.Invoicing)
}

func (s *StripeInvoiceTestSuite) TestCreateInvoiceIdempotencyKeyIsKeptAfterFailedRead() {
	ctx := s.T().Context()
	namespace := "ns-invoice-idempotency-failed-read"
	period := emptyInvoiceModeServicePeriod()

	clock.FreezeTime(period.To.Add(time.Minute))
	defer clock.UnFreeze()

	cust := s.setupEmptyInvoiceModeCustomer(ctx, namespace, billing.EmptyInvoiceModeCreate)
	defer s.StripeAppClient.Restore()

	stripeInvoices := s.stubIdempotentStripeInvoices()

	// given: Stripe creates the invoice, but reading it back fails without saying it is gone
	stripeInvoices.failNext["GetInvoice"] = errors.New("stripe is unavailable")

	// when: the draft is synced to Stripe
	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			emptyInvoiceModeFlatFeeLine(namespace, period, 10),
		}),
	})
	s.Require().NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]

	// then: the sync fails without giving up the Stripe invoice
	s.Require().Equal(billing.StandardInvoiceStatusDraftSyncFailed, invoice.Status)
	s.Require().Len(stripeInvoices.createKeys, 1)

	generation, err := s.AppStripeService.GetLatestInvoiceGeneration(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Equal(0, generation)

	// when: the sync is retried
	invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)

	// then: the same key gives back the invoice created the first time
	s.Require().Len(stripeInvoices.createKeys, 2)
	s.Equal(stripeInvoices.createKeys[0], stripeInvoices.createKeys[1])
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Equal("in_1", invoice.ExternalIDs.Invoicing)
}

func (s *StripeInvoiceTestSuite) TestInvoiceGenerationIncrementsAfterFailedDelete() {
	ctx := s.T().Context()
	namespace := "ns-invoice-idempotency-failed-delete"
	period := emptyInvoiceModeServicePeriod()

	clock.FreezeTime(period.To.Add(time.Minute))
	defer clock.UnFreeze()

	cust := s.setupEmptyInvoiceModeCustomer(ctx, namespace, billing.EmptyInvoiceModeOmit)
	defer s.StripeAppClient.Restore()

	stripeInvoices := s.stubIdempotentStripeInvoices()

	// given: a draft with a $10 flat fee, synced to Stripe
	invoice := s.createEmptyInvoiceModeDraft(ctx, cust, emptyInvoiceModeFlatFeeLine(namespace, period, 10))
	s.Require().Equal("in_1", invoice.ExternalIDs.Invoicing)

	// when: the flat fee is removed, but Stripe fails to delete the invoice
	stripeInvoices.failNext["DeleteInvoice"] = errors.New("stripe is unavailable")

	invoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      invoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(i *billing.StandardInvoice) error {
			for _, line := range i.Lines.OrEmpty() {
				line.DeletedAt = lo.ToPtr(clock.Now())
			}

			return nil
		},
	})
	s.Require().NoError(err)

	// then: the sync fails and the generation is incremented
	s.Equal(billing.StandardInvoiceStatusDraftSyncFailed, invoice.Status)

	generation, err := s.AppStripeService.GetLatestInvoiceGeneration(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(1, generation)
}
