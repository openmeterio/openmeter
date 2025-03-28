package metadata

import (
	"fmt"
	"strings"
)

// Entity types
const (
	EntityEntitlement  = "entitlement"
	EntitySubscription = "subscription"
	EntityInvoice      = "invoice"
	EntityCustomer     = "customer"
	EntitySubjectKey   = "subjectKey"
	EntityGrant        = "grant"
	EntityApp          = "app"
	EntityMeter        = "meter"
)

// Ingestion/Events
const (
	EntityEvent = "event"
)

func ComposeResourcePath(namespace string, items ...string) string {
	return ComposeResourcePathRaw(append([]string{"namespace", namespace}, items...)...)
}

func ComposeResourcePathRaw(items ...string) string {
	return fmt.Sprintf("//openmeter.io/%s", strings.Join(items, "/"))
}
