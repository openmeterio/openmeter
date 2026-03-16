package creditpurchase

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
)

// CreditPurchaseHandler is the interface for handling credit purchase charges.
// It is used to handle the different types of credit purchase charges (promotional, external, invoice).
//
// Promotional credit purchases are handled by the OnPromotionalCreditPurchase method only.
//
// Cost basis > 0 credit purchases are handled by the OnCreditPurchaseInitiated method, which is the initial call.
// Happy path:
// - OnCreditPurchaseInitiated is called
// - OnCreditPurchasePaymentAuthorized is called
// - OnCreditPurchasePaymentSettled is called
//
// Failed payment can occur either after the OnCreditPurchaseInitiated or after the OnCreditPurchasePaymentAuthorized call.

type Handler interface {
	// Promotional credit handler methods (cost basis == 0)
	// ----------------------------------------------------

	// OnPromotionalCreditPurchase is called when a promotional credit purchase is created (e.g. costbasis is 0)
	// For promotional credit purchases we don't call any of the payment handler methods.
	OnPromotionalCreditPurchase(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// Credit purchase handler methods (cost basis > 0)
	// ------------------------------------------------

	// OnCreditPurchaseInitiated is called when a credit purchase is initiated that is going to be settled by
	// a payment (either external or a standard invoice)
	// Initial call
	OnCreditPurchaseInitiated(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// OnCreditPurchasePaymentAuthorized is called when a credit purchase payment is authorized for a credit
	// purchase.
	OnCreditPurchasePaymentAuthorized(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// OnCreditPurchasePaymentSettled is called when a credit purchase payment is settled for a credit
	// purchase.
	OnCreditPurchasePaymentSettled(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)
}
