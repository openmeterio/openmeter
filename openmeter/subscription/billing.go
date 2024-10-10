package subscription

import (
	"context"
	"time"
)

type BillingAdapter interface {
	// LastInvoicedAt returns when a given customer was last invoiced.
	// An invoice is closed for these purposes if the contents can no longer be edited this no further charges can be added or modigied.
	LastInvoicedAt(ctx context.Context, customerId string) (time.Time, error)

	// TriggerInvoicing triggers the invoicing process for a given customer and subscription.
	// This has to happen if the subscription is changed in a way that cannot be reflected in the current invoice.
	//
	// TODO: think through when this needs to be called and what can fit on the same invoice.
	TriggerInvoicing(ctx context.Context, customerId string, subscriptionId string) error

	// StartNewInvoice starts a new invoice for a given customer and subscription.
	StartNewInvoice(ctx context.Context, customerId string, subscriptionId string) error
}
