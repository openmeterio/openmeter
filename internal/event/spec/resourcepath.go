package spec

import (
	"fmt"
	"strings"
)

const (
	EntityEntitlement = "entitlement"
	EntitySubjectKey  = "subjectKey"
	EntityGrant       = "grant"
)

func ComposeResourcePath(namespace string, items ...string) string {
	return fmt.Sprintf("//openmeter.io/namespace/%s/%s", namespace, strings.Join(items, "/"))
}
