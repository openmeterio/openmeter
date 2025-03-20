package entitlementdriver

const (
	AnnotationEntitlementManaged = "openmeter.entitlement.managed"
)

func Annotate(meta *map[string]string, key string, value string) {
	if meta == nil {
		meta = &map[string]string{}
	}

	if *meta == nil {
		*meta = map[string]string{}
	}

	(*meta)[key] = value
}
