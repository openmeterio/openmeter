package subscription

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	// AnnotationSubscriptionID is the ID of the subscription that created this entitlement
	AnnotationSubscriptionID = "subscription.id"

	AnnotationOwnerSubSystem = "subscription.owner"
)

const OwnerSubscriptionSubSystem = "subscription"

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

func (a annotationParser) ListOwnerSubSystems(annotations models.Annotations) []string {
	if annotations == nil {
		return nil
	}

	systems, ok := annotations[AnnotationOwnerSubSystem]
	if !ok {
		return nil
	}

	var systemsStr []string
	systemsArr, ok := systems.([]interface{})
	if !ok {
		return nil
	}

	for _, system := range systemsArr {
		systemStr, ok := system.(string)
		if !ok {
			return nil
		}
		systemsStr = append(systemsStr, systemStr)
	}

	return systemsStr
}

func (a annotationParser) AddOwnerSubSystem(annotations models.Annotations, system string) models.Annotations {
	if annotations == nil {
		annotations = models.Annotations{}
	}

	systems := a.ListOwnerSubSystems(annotations)
	systems = append(systems, system)
	annotations[AnnotationOwnerSubSystem] = systems

	return annotations
}

func (a annotationParser) RemoveOwnerSubSystem(annotations models.Annotations, system string) models.Annotations {
	if annotations == nil {
		return annotations
	}

	systems := a.ListOwnerSubSystems(annotations)
	systems = lo.Filter(systems, func(s string, _ int) bool {
		return s != system
	})
	annotations[AnnotationOwnerSubSystem] = systems

	return annotations
}
