package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

type UpsertResults struct {
	invoiceNumber string
	externalID    string

	lineExternalIDs         map[string]string
	lineDiscountExternalIDs map[string]string
}

func NewUpsertResults() *UpsertResults {
	return &UpsertResults{
		lineExternalIDs:         make(map[string]string),
		lineDiscountExternalIDs: make(map[string]string),
	}
}

func (u *UpsertResults) GetInvoiceNumber() (string, bool) {
	return u.invoiceNumber, u.invoiceNumber != ""
}

func (u *UpsertResults) SetInvoiceNumber(invoiceNumber string) *UpsertResults {
	u.invoiceNumber = invoiceNumber
	return u
}

func (u *UpsertResults) GetExternalID() (string, bool) {
	return u.externalID, u.externalID != ""
}

func (u *UpsertResults) SetExternalID(externalID string) *UpsertResults {
	u.externalID = externalID
	return u
}

func (u *UpsertResults) AddLineExternalID(lineID string, externalID string) *UpsertResults {
	u.lineExternalIDs[lineID] = externalID
	return u
}

func (u *UpsertResults) GetLineExternalID(lineID string) (string, bool) {
	externalID, ok := u.lineExternalIDs[lineID]
	return externalID, ok
}

func (u *UpsertResults) GetLineExternalIDs() map[string]string {
	return u.lineExternalIDs
}

func (u *UpsertResults) AddLineDiscountExternalID(lineDiscountID string, externalID string) *UpsertResults {
	u.lineDiscountExternalIDs[lineDiscountID] = externalID
	return u
}

func (u *UpsertResults) GetLineDiscountExternalID(lineDiscountID string) (string, bool) {
	externalID, ok := u.lineDiscountExternalIDs[lineDiscountID]
	return externalID, ok
}

func (u *UpsertResults) GetLineDiscountExternalIDs() map[string]string {
	return u.lineDiscountExternalIDs
}

type UpsertInvoiceResult = UpsertResults

func NewUpsertInvoiceResult() *UpsertInvoiceResult {
	return NewUpsertResults()
}

type FinalizeInvoiceResult struct {
	invoiceNumber     string
	paymentExternalID string
	sentToCustomerAt  mo.Option[time.Time]
}

func NewFinalizeInvoiceResult() *FinalizeInvoiceResult {
	return &FinalizeInvoiceResult{}
}

func (f *FinalizeInvoiceResult) GetPaymentExternalID() (string, bool) {
	return f.paymentExternalID, f.paymentExternalID != ""
}

func (f *FinalizeInvoiceResult) SetPaymentExternalID(paymentExternalID string) *FinalizeInvoiceResult {
	f.paymentExternalID = paymentExternalID
	return f
}

func (u *FinalizeInvoiceResult) GetInvoiceNumber() (string, bool) {
	return u.invoiceNumber, u.invoiceNumber != ""
}

func (f *FinalizeInvoiceResult) SetInvoiceNumber(invoiceNumber string) *FinalizeInvoiceResult {
	f.invoiceNumber = invoiceNumber
	return f
}

func (f *FinalizeInvoiceResult) GetSentToCustomerAt() (time.Time, bool) {
	return f.sentToCustomerAt.OrEmpty(), f.sentToCustomerAt.IsPresent()
}

func (f *FinalizeInvoiceResult) SetSentToCustomerAt(sentToCustomerAt time.Time) *FinalizeInvoiceResult {
	f.sentToCustomerAt = mo.Some(sentToCustomerAt)
	return f
}

func (f *FinalizeInvoiceResult) MergeIntoInvoice(invoice *Invoice) error {
	if paymentExternalID, ok := f.GetPaymentExternalID(); ok {
		invoice.ExternalIDs.Payment = paymentExternalID
	}

	if invoiceNumber, ok := f.GetInvoiceNumber(); ok {
		invoice.Number = invoiceNumber
	}

	if sentToCustomerAt, ok := f.GetSentToCustomerAt(); ok {
		invoice.SentToCustomerAt = &sentToCustomerAt
	}

	return nil
}

type PostAdvanceHookResult struct {
	trigger *InvoiceTriggerInput
}

func NewPostAdvanceHookResult() *PostAdvanceHookResult {
	return &PostAdvanceHookResult{}
}

func (p *PostAdvanceHookResult) InvokeTrigger(trigger InvoiceTriggerInput) *PostAdvanceHookResult {
	p.trigger = &trigger
	return p
}

func (p *PostAdvanceHookResult) GetTriggerToInvoke() *InvoiceTriggerInput {
	return p.trigger
}

// InvoicingApp is the interface that should be implemented by the app to handle the invoicing
//
// apps can also implement InvoicingAppPostAdvanceHook to perform additional actions after the invoice
// has been advanced

// Warning: The received invoice is
//   - read-only (e.g. any changes made to it are lost to prevent manipulation of the invoice state)
//   - reflects the current in memory state of the invoice, thus if you fetched from the db
//     an earlier version of the invoice will be passed, thus do not call any billingService methods
//     from these callbacks.
type InvoicingApp interface {
	// ValidateInvoice validates if the app can run for the given invoice
	ValidateInvoice(ctx context.Context, invoice Invoice) error

	// UpsertInvoice upserts the invoice on the remote system, the invoice is read-only, the app should not modify it
	// the recommended behavior is that the invoices FlattenLinesByID is used to get all lines, then the app should
	// syncronize all the fee lines and store the external IDs in the result.
	UpsertInvoice(ctx context.Context, invoice Invoice) (*UpsertInvoiceResult, error)

	// FinalizeInvoice finalizes the invoice on the remote system, starts the payment flow. It is safe to assume
	// that the state machine have already performed an upsert as part of this state transition.
	//
	// If the payment is handled by a decoupled implementation (different app or app has strict separation of concerns)
	// then the payment app will be called with FinalizePayment and that should return the external ID of the payment. (later)
	FinalizeInvoice(ctx context.Context, invoice Invoice) (*FinalizeInvoiceResult, error)

	// DeleteInvoice deletes the invoice on the remote system, the invoice is read-only, the app should not modify it
	// the invoice deletion is only invoked for non-finalized invoices.
	DeleteInvoice(ctx context.Context, invoice Invoice) error
}

type InvoicingAppPostAdvanceHook interface {
	// PostAdvanceInvoiceHook is called after the invoice has been advanced to the next stable state
	// (e.g. no next trigger is available)
	//
	// Can be used by the app to perform additional actions in case there are some post-processing steps
	// required on the invoice.
	PostAdvanceInvoiceHook(ctx context.Context, invoice Invoice) (*PostAdvanceHookResult, error)
}

// InvoicingAppAsyncSyncer is an optional interface that can be implemented by the app to support
// asynchronous syncing of the invoice (e.g. when we are receiving the payload such as with custominvoicing app)
type InvoicingAppAsyncSyncer interface {
	CanDraftSyncAdvance(invoice Invoice) (bool, error)
	CanIssuingSyncAdvance(invoice Invoice) (bool, error)
	// TODO: finalization check
}

// GetApp returns the app from the app entity
func GetApp(app app.App) (InvoicingApp, error) {
	customerApp, ok := app.(InvoicingApp)
	if !ok {
		return nil, AppError{
			AppID:   app.GetID(),
			AppType: app.GetType(),
			Err:     fmt.Errorf("is not an invoicing app"),
		}
	}

	return customerApp, nil
}

// MergeIntoInvoice merges the upsert invoice result into the invoice.
func (r UpsertInvoiceResult) MergeIntoInvoice(invoice *Invoice) error {
	// Let's merge the results into the invoice
	if invoiceNumber, ok := r.GetInvoiceNumber(); ok {
		invoice.Number = invoiceNumber
	}

	if externalID, ok := r.GetExternalID(); ok {
		invoice.ExternalIDs.Invoicing = externalID
	}

	var outErr error

	// Let's merge the line IDs
	if len(r.GetLineExternalIDs()) > 0 {
		flattenedLines := invoice.FlattenLinesByID()

		// Merge the line IDs
		for lineID, externalID := range r.GetLineExternalIDs() {
			if line, ok := flattenedLines[lineID]; ok {
				line.ExternalIDs.Invoicing = externalID
			} else {
				outErr = errors.Join(outErr, fmt.Errorf("line not found in invoice: %s", lineID))
			}
		}

		// Let's merge the line discount IDs
		dicountIDToExternalID := r.GetLineDiscountExternalIDs()

		for _, line := range flattenedLines {
			for idx, discount := range line.Discounts.Amount {
				if externalID, ok := dicountIDToExternalID[discount.ID]; ok {
					line.Discounts.Amount[idx].ExternalIDs.Invoicing = externalID
				}
			}

			for idx, discount := range line.Discounts.Usage {
				if externalID, ok := dicountIDToExternalID[discount.ID]; ok {
					line.Discounts.Usage[idx].ExternalIDs.Invoicing = externalID
				}
			}
		}
	}

	return outErr
}

type SyncInput interface {
	models.Validator

	MergeIntoInvoice(invoice *Invoice) error
	GetAdditionalMetadata() map[string]string
	GetInvoiceID() InvoiceID
}

var _ SyncInput = (*SyncDraftInvoiceInput)(nil)

type SyncDraftInvoiceInput struct {
	InvoiceID            InvoiceID
	UpsertInvoiceResults *UpsertInvoiceResult
	AdditionalMetadata   map[string]string
}

func (i SyncDraftInvoiceInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.AdditionalMetadata == nil {
		errs = append(errs, fmt.Errorf("additional metadata is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i SyncDraftInvoiceInput) MergeIntoInvoice(invoice *Invoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}
	if i.UpsertInvoiceResults != nil {
		return i.UpsertInvoiceResults.MergeIntoInvoice(invoice)
	}

	return nil
}

func (i SyncDraftInvoiceInput) GetAdditionalMetadata() map[string]string {
	return i.AdditionalMetadata
}

func (i SyncDraftInvoiceInput) GetInvoiceID() InvoiceID {
	return i.InvoiceID
}

var _ SyncInput = (*SyncIssuingInvoiceInput)(nil)

type SyncIssuingInvoiceInput struct {
	InvoiceID             InvoiceID
	FinalizeInvoiceResult *FinalizeInvoiceResult
	AdditionalMetadata    map[string]string
}

func (i SyncIssuingInvoiceInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.AdditionalMetadata == nil {
		errs = append(errs, fmt.Errorf("additional metadata is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i SyncIssuingInvoiceInput) MergeIntoInvoice(invoice *Invoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}

	if i.FinalizeInvoiceResult != nil {
		return i.FinalizeInvoiceResult.MergeIntoInvoice(invoice)
	}

	return nil
}

func (i SyncIssuingInvoiceInput) GetAdditionalMetadata() map[string]string {
	return i.AdditionalMetadata
}

func (i SyncIssuingInvoiceInput) GetInvoiceID() InvoiceID {
	return i.InvoiceID
}
