package provider

type OpenMeterReference struct{}

type StripeReference struct {
	InvoiceID string `json:"invoiceID,omitempty"`
}

type Reference struct {
	Meta

	OpenMeter OpenMeterReference `json:"openmeter,omitempty"`
	Stripe    StripeReference    `json:"stripe,omitempty"`
}
