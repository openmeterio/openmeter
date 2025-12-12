package entitlement

type Access struct {
	// Map of featureKey to entitlement value + ID
	Entitlements map[string]EntitlementValueWithId
}

type EntitlementValueWithId struct {
	Type  EntitlementType
	Value EntitlementValue
	ID    string
}
