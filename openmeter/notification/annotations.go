package notification

const (
	// AnnotationRuleTestEvent indicates that the event is generated as part of testing a notification rule
	AnnotationRuleTestEvent = "notification.rule.test"

	AnnotationEventFeatureKey = "event.feature.key"
	AnnotationEventFeatureID  = "event.feature.id"
	AnnotationEventSubjectKey = "event.subject.key"
	AnnotationEventSubjectID  = "event.subject.id"

	// TODO[later]: deprecate this annotation and use a generic one
	AnnotationBalanceEventDedupeHash = "event.balance.dedupe.hash"

	AnnotationEventInvoiceID     = "event.invoice.id"
	AnnotationEventInvoiceNumber = "event.invoice.number"
)
