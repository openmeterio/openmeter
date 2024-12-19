package billing

import (
	"context"
)

type UpsertResults struct {
	invoiceNumber string
	externalID    string

	lineExternalIDs         map[string]string
	LineDiscountExternalIDs map[string]string
}

func NewUpsertResults() *UpsertResults {
	return &UpsertResults{
		lineExternalIDs: make(map[string]string),
	}
}

func (u *UpsertResults) GetInvoiceNumber() (string, bool) {
	return u.invoiceNumber, u.invoiceNumber != ""
}

func (u *UpsertResults) SetInvoiceNumber(invoiceNumber string) {
	u.invoiceNumber = invoiceNumber
}

func (u *UpsertResults) GetExternalID() (string, bool) {
	return u.externalID, u.externalID != ""
}

func (u *UpsertResults) SetExternalID(externalID string) {
	u.externalID = externalID
}

func (u *UpsertResults) AddLineExternalID(lineID string, externalID string) {
	u.lineExternalIDs[lineID] = externalID
}

func (u *UpsertResults) GetLineExternalID(lineID string) (string, bool) {
	externalID, ok := u.lineExternalIDs[lineID]
	return externalID, ok
}

func (u *UpsertResults) GetLineExternalIDs() map[string]string {
	return u.lineExternalIDs
}

func (u *UpsertResults) AddLineDiscountExternalID(lineDiscountID string, externalID string) {
	u.LineDiscountExternalIDs[lineDiscountID] = externalID
}

func (u *UpsertResults) GetLineDiscountExternalID(lineDiscountID string) (string, bool) {
	externalID, ok := u.LineDiscountExternalIDs[lineDiscountID]
	return externalID, ok
}

func (u *UpsertResults) GetLineDiscountExternalIDs() map[string]string {
	return u.LineDiscountExternalIDs
}

type UpsertInvoiceResult = UpsertResults

func NewUpsertInvoiceResult() *UpsertInvoiceResult {
	return NewUpsertResults()
}

type FinalizeInvoiceResult struct {
	paymentExternalID string
}

func NewFinalizeInvoiceResult() *FinalizeInvoiceResult {
	return &FinalizeInvoiceResult{}
}

func (f *FinalizeInvoiceResult) GetPaymentExternalID() (string, bool) {
	return f.paymentExternalID, f.paymentExternalID != ""
}

func (f *FinalizeInvoiceResult) SetPaymentExternalID(paymentExternalID string) {
	f.paymentExternalID = paymentExternalID
}

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
