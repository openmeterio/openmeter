package subscription

import "github.com/openmeterio/openmeter/pkg/models"

const (
	// AnnotationSubscriptionID is the ID of the subscription that created this entitlement
	AnnotationSubscriptionID = "subscription.id"
)

type annotationParser struct{}

var AnnotationParser = annotationParser{}

func (a annotationParser) GetSubscriptionID(annotations models.Annotations) (string, bool) {
	subId, ok := annotations[AnnotationSubscriptionID]
	if !ok {
		return "", false
	}

	subIdStr, ok := subId.(string)
	if !ok {
		return "", false
	}

	return subIdStr, true
}

func (a annotationParser) HasSubscription(annotations models.Annotations) bool {
	subId, ok := annotations[AnnotationSubscriptionID]
	return ok && subId != nil
}
