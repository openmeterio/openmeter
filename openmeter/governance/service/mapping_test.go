package service

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/governance"
)

func TestMapEntitlementToAccess(t *testing.T) {
	tests := []struct {
		name          string
		value         entitlement.EntitlementValue
		wantHasAccess bool
		wantCode      *governance.ReasonCode
	}{
		{
			name:          "metered with balance — has access",
			value:         &meteredentitlement.MeteredEntitlementValue{Balance: 10},
			wantHasAccess: true,
		},
		{
			name:          "metered exhausted — usage limit reached",
			value:         &meteredentitlement.MeteredEntitlementValue{Balance: 0},
			wantHasAccess: false,
			wantCode:      lo.ToPtr(governance.ReasonCodeUsageLimitReached),
		},
		{
			// BooleanEntitlementValue is always HasAccess=true; the gateway returns
			// NoAccessValue when the entitlement is inactive/not in plan.
			name:          "boolean — has access",
			value:         &booleanentitlement.BooleanEntitlementValue{},
			wantHasAccess: true,
		},
		{
			// StaticEntitlementValue is always HasAccess=true.
			name:          "static — has access",
			value:         &staticentitlement.StaticEntitlementValue{Config: `{"limit":100}`},
			wantHasAccess: true,
		},
		{
			// NoAccessValue is returned when the entitlement is inactive (not in current period).
			name:          "no access value — feature unavailable",
			value:         &entitlement.NoAccessValue{},
			wantHasAccess: false,
			wantCode:      lo.ToPtr(governance.ReasonCodeFeatureUnavailable),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapEntitlementToAccess(tc.value)

			assert.Equal(t, tc.wantHasAccess, got.HasAccess)

			if tc.wantCode != nil {
				if assert.NotNil(t, got.Reason) {
					assert.Equal(t, *tc.wantCode, got.Reason.Code)
				}
			} else {
				assert.Nil(t, got.Reason)
			}
		})
	}
}
