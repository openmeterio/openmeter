package meta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionReferenceEqual(t *testing.T) {
	reference := SubscriptionReference{
		SubscriptionID: "subscription-id",
		PhaseID:        "phase-id",
		ItemID:         "item-id",
	}

	tests := []struct {
		name     string
		left     SubscriptionReference
		right    SubscriptionReference
		expected bool
	}{
		{name: "equal", left: reference, right: reference, expected: true},
		{
			name: "different",
			left: reference,
			right: SubscriptionReference{
				SubscriptionID: "subscription-id",
				PhaseID:        "phase-id",
				ItemID:         "other-item-id",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.left.Equal(tt.right))
		})
	}
}
