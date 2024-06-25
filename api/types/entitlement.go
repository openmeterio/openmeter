package types

// CreateEntitlementJSONBodyType defines parameters for CreateEntitlement.
//
// ENUM: "metered", "static", "boolean"
type CreateEntitlementJSONBodyType string

const (
	CreateEntitlementJSONBodyTypeMetered CreateEntitlementJSONBodyType = "metered"
	CreateEntitlementJSONBodyTypeStatic  CreateEntitlementJSONBodyType = "static"
	CreateEntitlementJSONBodyTypeBoolean CreateEntitlementJSONBodyType = "boolean"
)
