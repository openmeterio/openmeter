package appservice

const (
	StripeInvoiceIDAttributeName = "invoice.stripe_invoice_id"
	InvoiceIDAttributeName       = "invoice.id"
	InvoiceStatusAttributeName   = "invoice.status"
)

// LatestWebhookSchemaVersion is persisted on newly installed Stripe apps and identifies the
// webhook event set SetupWebhook registered. Bump it whenever that event list changes.
const LatestWebhookSchemaVersion = 2
