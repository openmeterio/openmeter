package spec

import (
	"fmt"
	"strings"
)

// Entitlements
const (
	EntityEntitlement = "entitlement"
	EntitySubjectKey  = "subjectKey"
	EntityGrant       = "grant"
)

// Ingestion/Events
const (
	EntityEvent = "event"
)

func ComposeResourcePath(namespace string, items ...string) string {
	return fmt.Sprintf("//openmeter.io/namespace/%s/%s", namespace, strings.Join(items, "/"))
}
