package entitlementdriverv2

import (
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// getErrorEncoder mirrors the v1 driver's error encoder to keep behavior consistent
func getErrorEncoder() encoder.ErrorEncoder { return entitlementdriver.GetErrorEncoder() }
