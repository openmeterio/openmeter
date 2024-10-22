package entitlement

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/framework/annotations"
)

var SystemManagedEntitlementAnnotation annotations.Annotation = annotations.Annotation{
	Key:   "openmeter.io/entitlements/system-managed",
	Value: "true",
}

type guardSystemManagedEntitlements struct{}

// FIXME: this is kindof ugly and sholdn't be used as a pattern, lets improve on it
// TODO: this check should be added to user initated operations...
var EntitlementAnnotationGuard annotations.Parser[Entitlement, any] = (*guardSystemManagedEntitlements)(nil)

func (p *guardSystemManagedEntitlements) Parse(val Entitlement) (any, error) {
	annotations := val.AnnotatedModel.Metadata

	// Check if the entitlement is system managed
	if val, ok := annotations[SystemManagedEntitlementAnnotation.Key]; ok {
		if val == SystemManagedEntitlementAnnotation.Value {
			return nil, &ForbiddenError{
				Message: "System managed entitlements cannot be modified",
			}
		} else {
			return nil, fmt.Errorf("Invalid value for system managed entitlement annotation (%s): %s", SystemManagedEntitlementAnnotation.Key, val)
		}
	}

	return nil, nil
}
