package subscription

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	// AnnotationSubscriptionID is the ID of the subscription that created this entitlement
	AnnotationSubscriptionID = "subscription.id"

	AnnotationOwnerSubSystem = "subscription.owner"

	AnnotationBooleanEntitlementCount = "subscription.entitlement.boolean.count"
)

const OwnerSubscriptionSubSystem = "subscription"

type annotationParser struct{}

var AnnotationParser = annotationParser{}

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

func (a annotationParser) AddOwnerSubSystem(annotations models.Annotations, system string) (models.Annotations, error) {
	if annotations == nil {
		return nil, errors.New("annotations are nil")
	}

	systems := a.ListOwnerSubSystems(annotations)
	if lo.Contains(systems, system) {
		return annotations, nil
	}

	systems = append(systems, system)
	annotations[AnnotationOwnerSubSystem] = systems

	return annotations, nil
}

func (a annotationParser) GetBooleanEntitlementCount(annotations models.Annotations) int {
	count, ok := annotations[AnnotationBooleanEntitlementCount]
	if !ok {
		return 0
	}

	countInt, ok := count.(int)
	if !ok {
		countFloat, ok := count.(float64)
		if !ok {
			return 0
		}

		countInt = int(countFloat)
	}

	return countInt
}

func (a annotationParser) SetBooleanEntitlementCount(annotations models.Annotations, count int) (models.Annotations, error) {
	if annotations == nil {
		return nil, errors.New("annotations are nil")
	}
	annotations[AnnotationBooleanEntitlementCount] = count
	return annotations, nil
}
