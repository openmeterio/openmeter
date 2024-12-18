package appstripe

import (
	"context"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"
)

// TestCreateInvoice tests stripe app behavior when creating an invoice.
func (s *AppHandlerTestSuite) TestCreateInvoice(ctx context.Context, t *testing.T) {
	app, customer, customerData := s.setupAppWithCustomer(ctx, t)
	stripeClient := s.Env.StripeClient()

	// Create a new invoice for the customer.
	invoicingApp, err := billing.GetApp(app)
	require.NoError(t, err)

	invoice := billing.Invoice{
		InvoiceBase: billing.InvoiceBase{
			Namespace: s.namespace,
			Customer: billing.InvoiceCustomer{
				CustomerID:       customer.ID,
				Name:             customer.Name,
				BillingAddress:   customer.BillingAddress,
				Timezone:         customer.Timezone,
				UsageAttribution: customer.UsageAttribution,
			},
			Currency: "USD",
		},
		// TODO: Lines
	}

	// Mock the stripe client to return the created invoice.
	stripeClient.SetMockInvoice(&stripe.Invoice{
		ID: "stripe-invoice-id",
		Customer: &stripe.Customer{
			ID: customerData.StripeCustomerID,
		},
		Currency: "USD",
	})

	// Create the invoice.
	results, err := invoicingApp.UpsertInvoice(ctx, invoice)
	require.NoError(t, err)

	// Assert external ID is set.
	externalId, ok := results.GetExternalID()
	require.True(t, ok, "external ID is not set")

	require.Equal(t, "stripe-invoice-id", externalId)

	// Assert customer ID is set.
	// Assert currency is set.
	// Assert due date is set.
}
