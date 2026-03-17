package billing

import (
	"context"
	"fmt"
	"slices"
)

type LifecycleHandler string

const (
	DefaultLifecycleHandler LifecycleHandler = "default"
	ChargesLifecycleHandler LifecycleHandler = "charges"
)

func (h LifecycleHandler) Values() []string {
	return []string{
		string(DefaultLifecycleHandler),
		string(ChargesLifecycleHandler),
	}
}

func (h LifecycleHandler) Validate() error {
	if !slices.Contains(h.Values(), string(h)) {
		return fmt.Errorf("invalid lifecycle handler: %s", h)
	}

	return nil
}

type IsLineBillableResult struct {
	IsBillable      bool
	ValidationError error
}

type AreLinesBillableResult []IsLineBillableResult

type InvoiceLifecycleHandler interface {
	// AreLinesBillable is used to determine if the lines are billable as of the given time.
	AreLinesBillable(ctx context.Context, invoice GatheringInvoice, lines GatheringLines) (AreLinesBillableResult, error)

	// HandleInvoiceCreation is used to handle the creation of the invoice line when creating the standard invoice.
	HandleInvoiceCreation(ctx context.Context, line GatheringLines) (StandardLines, error)

	// HandleCollectionSnapshot is invoked when the draft.collecting state is entered.
	HandleCollectionSnapshot(ctx context.Context, lines StandardLines) (StandardLines, error)

	// HandleInvoiceIssued is invoked when the invoice is issued.
	HandleInvoiceIssued(ctx context.Context, lines StandardLines) error

	// HandlePaymentAuthorized is invoked when the payment is authorized.
	HandlePaymentAuthorized(ctx context.Context, lines StandardLines) error

	// HandlePaymentSettled is invoked when the payment is settled.
	HandlePaymentSettled(ctx context.Context, lines StandardLines) error
}
