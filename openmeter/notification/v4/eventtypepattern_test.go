package v4

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEventTypePattern(t *testing.T) {
	created := EventType{Name: "billing.invoice.created", Version: 1}
	updated := EventType{Name: "billing.invoice.updated", Version: 1}
	createdV2 := EventType{Name: "billing.invoice.created", Version: 2}
	billingx := EventType{Name: "billingx.invoice.created", Version: 1}
	triggered := EventType{Name: "entitlement.balance_threshold.triggered", Version: 1}

	all := []EventType{created, updated, createdV2, billingx, triggered}

	tests := []struct {
		pattern string
		matches []EventType
	}{
		{"v1.billing.invoice.created", []EventType{created}},
		{"v1.billing.invoice.*", []EventType{created, updated}},
		{"v1.billing.*", []EventType{created, updated}},
		{"v1.*", []EventType{created, updated, billingx, triggered}},
		{"v2.*", []EventType{createdV2}},
		{"v1.billing.invoice.created.*", nil},
		{"v3.billing.invoice.created", nil},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			p, err := ParseEventTypePattern(tt.pattern)
			require.NoError(t, err)
			require.Equal(t, tt.pattern, p.String())

			var matched []EventType

			for _, et := range all {
				if p.Matches(et) {
					matched = append(matched, et)
				}
			}

			require.Equal(t, tt.matches, matched)
		})
	}

	for _, invalid := range []string{
		"",
		"*",
		"v1",
		"v1.",
		"v*.billing.invoice.created",
		"v1.billing",
		"v1.billing*",
		"v1.*.created",
		"v1.billing.*.created",
		"v1.*.*",
		"v1.billing.**",
		"v1.Billing.*",
		"v1.billing..*",
		"v1..*",
		"!v1.billing.*",
	} {
		_, err := ParseEventTypePattern(invalid)
		require.True(t, models.IsGenericValidationError(err), "%q: %v", invalid, err)
	}
}
