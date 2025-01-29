package httpdriver

type StripeLogAttributeName string

const (
	StripeEventIDAttributeName   StripeLogAttributeName = "stripe_event_id"
	StripeEventTypeAttributeName StripeLogAttributeName = "stripe_event_type"
	AppIDAttributeName           StripeLogAttributeName = "app_id"
)
