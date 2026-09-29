package notification

const (
	// AnnotationRuleTestEvent indicates that the event is generated as part of testing a notification rule
	AnnotationRuleTestEvent = "notification.rule.test"

	AnnotationEventFeatureKey = "event.feature.key"
	AnnotationEventFeatureID  = "event.feature.id"
	AnnotationEventSubjectKey = "event.subject.key"
	AnnotationEventSubjectID  = "event.subject.id"

	AnnotationEventCustomerID  = "event.customer.id"
	AnnotationEventCustomerKey = "event.customer.key"

	// TODO[later]: deprecate this annotation and use a generic one
	AnnotationBalanceEventDedupeHash = "event.balance.dedupe.hash"

	AnnotationEventInvoiceID     = "event.invoice.id"
	AnnotationEventInvoiceNumber = "event.invoice.number"

	AnnotationEventResendTimestamp = "event.resend.timestamp"

	// AnnotationChannelProviderDisabledTimestamp records when the webhook provider was observed to have
	// disabled the endpoint backing this channel. The provider disables endpoints on its own after a
	// prolonged delivery failure and never pushes that decision back to us, so without this marker the
	// channel would keep reporting itself as enabled while every event sent through it fails immediately.
	// Updating the channel through the API clears the annotation and re-enables the endpoint at the provider.
	AnnotationChannelProviderDisabledTimestamp = "channel.provider.disabled.timestamp"
)
