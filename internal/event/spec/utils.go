package spec

import (
	"fmt"
	"strings"
)

func ComposeResourcePath(namespace string, items ...string) string {
	return fmt.Sprintf("//openmeter.io/namespace/%s/%s", namespace, strings.Join(items, "/"))
}
