package notification

import "github.com/openmeterio/openmeter/api"

const (
	// AnnotationRuleTestEvent indicates that the event is generated as part of testing a notification rule
	AnnotationRuleTestEvent = "notification.rule.test"

	AnnotationEventFeatureKey = "event.feature.key"
	AnnotationEventFeatureID  = "event.feature.id"
	AnnotationEventSubjectKey = "event.subject.key"
	AnnotationEventSubjectID  = "event.subject.id"

	AnnotationEventDedupeHash = "event.balance.dedupe.hash"
)

type Annotations = api.Annotations
