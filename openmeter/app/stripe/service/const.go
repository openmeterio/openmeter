package appservice

type StripeLogAttributeName string

const (
	StripeInvoiceIDAttributeName StripeLogAttributeName = "stripe_invoice_id"
	InvoiceIDAttributeName       StripeLogAttributeName = "invoice_id"
	InvoiceStatusAttributeName   StripeLogAttributeName = "invoice_status"
)
