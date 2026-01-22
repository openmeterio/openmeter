package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
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

type UpsertStandardInvoiceResult = UpsertResults

func NewUpsertStandardInvoiceResult() *UpsertStandardInvoiceResult {
	return NewUpsertResults()
}

type FinalizeStandardInvoiceResult struct {
	invoiceNumber     string
	paymentExternalID string
	sentToCustomerAt  mo.Option[time.Time]
}

func NewFinalizeStandardInvoiceResult() *FinalizeStandardInvoiceResult {
	return &FinalizeStandardInvoiceResult{}
}

func (f *FinalizeStandardInvoiceResult) GetPaymentExternalID() (string, bool) {
	return f.paymentExternalID, f.paymentExternalID != ""
}

func (f *FinalizeStandardInvoiceResult) SetPaymentExternalID(paymentExternalID string) *FinalizeStandardInvoiceResult {
	f.paymentExternalID = paymentExternalID
	return f
}

func (u *FinalizeStandardInvoiceResult) GetInvoiceNumber() (string, bool) {
	return u.invoiceNumber, u.invoiceNumber != ""
}

func (f *FinalizeStandardInvoiceResult) SetInvoiceNumber(invoiceNumber string) *FinalizeStandardInvoiceResult {
	f.invoiceNumber = invoiceNumber
	return f
}

func (f *FinalizeStandardInvoiceResult) GetSentToCustomerAt() (time.Time, bool) {
	return f.sentToCustomerAt.OrEmpty(), f.sentToCustomerAt.IsPresent()
}

func (f *FinalizeStandardInvoiceResult) SetSentToCustomerAt(sentToCustomerAt time.Time) *FinalizeStandardInvoiceResult {
	f.sentToCustomerAt = mo.Some(sentToCustomerAt)
	return f
}

func (f *FinalizeStandardInvoiceResult) MergeIntoInvoice(invoice *StandardInvoice) error {
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
	// ValidateStandardInvoice validates if the app can run for the given invoice
	ValidateStandardInvoice(ctx context.Context, invoice StandardInvoice) error

	// UpsertStandardInvoice upserts the invoice on the remote system, the invoice is read-only, the app should not modify it
	// the recommended behavior is that the invoices FlattenLinesByID is used to get all lines, then the app should
	// synchronize all the fee lines and store the external IDs in the result.
	UpsertStandardInvoice(ctx context.Context, invoice StandardInvoice) (*UpsertStandardInvoiceResult, error)

	// FinalizeStandardInvoice finalizes the invoice on the remote system, starts the payment flow. It is safe to assume
	// that the state machine have already performed an upsert as part of this state transition.
	//
	// If the payment is handled by a decoupled implementation (different app or app has strict separation of concerns)
	// then the payment app will be called with FinalizePayment and that should return the external ID of the payment. (later)
	FinalizeStandardInvoice(ctx context.Context, invoice StandardInvoice) (*FinalizeStandardInvoiceResult, error)

	// DeleteStandardInvoice deletes the invoice on the remote system, the invoice is read-only, the app should not modify it
	// the invoice deletion is only invoked for non-finalized invoices.
	DeleteStandardInvoice(ctx context.Context, invoice StandardInvoice) error
}

type InvoicingAppPostAdvanceHook interface {
	// PostAdvanceInvoiceHook is called after the invoice has been advanced to the next stable state
	// (e.g. no next trigger is available)
	//
	// Can be used by the app to perform additional actions in case there are some post-processing steps
	// required on the invoice.
	PostAdvanceStandardInvoiceHook(ctx context.Context, invoice StandardInvoice) (*PostAdvanceHookResult, error)
}

// InvoicingAppAsyncSyncer is an optional interface that can be implemented by the app to support
// asynchronous syncing of the invoice (e.g. when we are receiving the payload such as with custominvoicing app)
type InvoicingAppAsyncSyncer interface {
	CanDraftSyncAdvance(invoice StandardInvoice) (bool, error)
	CanIssuingSyncAdvance(invoice StandardInvoice) (bool, error)
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
func (r UpsertStandardInvoiceResult) MergeIntoInvoice(invoice *StandardInvoice) error {
	// Let's merge the results into the invoice
	if invoiceNumber, ok := r.GetInvoiceNumber(); ok {
		invoice.Number = invoiceNumber
	}

	if externalID, ok := r.GetExternalID(); ok {
		invoice.ExternalIDs.Invoicing = externalID
	}

	if !invoice.Lines.IsPresent() {
		return errors.New("invoice has no expanded lines")
	}

	var outErr error

	// Let's merge the line IDs
	lineIDToExternalID := r.GetLineExternalIDs()
	dicountIDToExternalID := r.GetLineDiscountExternalIDs()

	lines := invoice.Lines.OrEmpty()

	for _, line := range lines {
		if externalID, ok := lineIDToExternalID[line.ID]; ok {
			line.ExternalIDs.Invoicing = externalID
			delete(lineIDToExternalID, line.ID)
		}

		foundIDs := line.SetDiscountExternalIDs(dicountIDToExternalID)
		for _, id := range foundIDs {
			delete(dicountIDToExternalID, id)
		}

		for idx, detailedLine := range line.DetailedLines {
			if externalID, ok := lineIDToExternalID[detailedLine.ID]; ok {
				line.DetailedLines[idx].ExternalIDs.Invoicing = externalID
				delete(lineIDToExternalID, detailedLine.ID)
			}

			foundIDs := line.DetailedLines[idx].SetDiscountExternalIDs(dicountIDToExternalID)
			for _, id := range foundIDs {
				delete(dicountIDToExternalID, id)
			}
		}
	}

	if len(lineIDToExternalID) > 0 {
		outErr = errors.Join(outErr, fmt.Errorf("some lines were not found in the invoice: ids=[%s]", strings.Join(lo.Keys(lineIDToExternalID), ", ")))
	}

	if len(dicountIDToExternalID) > 0 {
		outErr = errors.Join(outErr, fmt.Errorf("some line discounts were not found in the invoice: ids=[%s]", strings.Join(lo.Keys(dicountIDToExternalID), ", ")))
	}

	return outErr
}

type SyncInput interface {
	models.Validator

	ValidateWithInvoice(invoice StandardInvoice) error
	MergeIntoInvoice(invoice *StandardInvoice) error
	GetAdditionalMetadata() map[string]string
	GetInvoiceID() InvoiceID
}

var _ SyncInput = (*SyncDraftStandardInvoiceInput)(nil)

type SyncDraftStandardInvoiceInput struct {
	InvoiceID            InvoiceID
	UpsertInvoiceResults *UpsertStandardInvoiceResult
	AdditionalMetadata   map[string]string
	InvoiceValidator     func(invoice StandardInvoice) error
}

func (i SyncDraftStandardInvoiceInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.AdditionalMetadata == nil {
		errs = append(errs, fmt.Errorf("additional metadata is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i SyncDraftStandardInvoiceInput) MergeIntoInvoice(invoice *StandardInvoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}
	if i.UpsertInvoiceResults != nil {
		return i.UpsertInvoiceResults.MergeIntoInvoice(invoice)
	}

	return nil
}

func (i SyncDraftStandardInvoiceInput) GetAdditionalMetadata() map[string]string {
	return i.AdditionalMetadata
}

func (i SyncDraftStandardInvoiceInput) GetInvoiceID() InvoiceID {
	return i.InvoiceID
}

func (i SyncDraftStandardInvoiceInput) ValidateWithInvoice(invoice StandardInvoice) error {
	if i.InvoiceValidator != nil {
		return i.InvoiceValidator(invoice)
	}

	return nil
}

var _ SyncInput = (*SyncIssuingStandardInvoiceInput)(nil)

type SyncIssuingStandardInvoiceInput struct {
	InvoiceID             InvoiceID
	FinalizeInvoiceResult *FinalizeStandardInvoiceResult
	AdditionalMetadata    map[string]string
	InvoiceValidator      func(invoice StandardInvoice) error
}

func (i SyncIssuingStandardInvoiceInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.AdditionalMetadata == nil {
		errs = append(errs, fmt.Errorf("additional metadata is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i SyncIssuingStandardInvoiceInput) MergeIntoInvoice(invoice *StandardInvoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}

	if i.FinalizeInvoiceResult != nil {
		return i.FinalizeInvoiceResult.MergeIntoInvoice(invoice)
	}

	return nil
}

func (i SyncIssuingStandardInvoiceInput) GetAdditionalMetadata() map[string]string {
	return i.AdditionalMetadata
}

func (i SyncIssuingStandardInvoiceInput) GetInvoiceID() InvoiceID {
	return i.InvoiceID
}

func (i SyncIssuingStandardInvoiceInput) ValidateWithInvoice(invoice StandardInvoice) error {
	if i.InvoiceValidator != nil {
		return i.InvoiceValidator(invoice)
	}

	return nil
}
