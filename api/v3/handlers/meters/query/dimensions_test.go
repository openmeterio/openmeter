package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIsReservedDimension(t *testing.T) {
	assert.True(t, IsReservedDimension("subject"))
	assert.True(t, IsReservedDimension("customer_id"))
	assert.False(t, IsReservedDimension("region"))
	assert.False(t, IsReservedDimension(""))
}

func TestIsSupportedGroupByDimension(t *testing.T) {
	m := meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "test"},
		},
		GroupBy: map[string]string{
			"region": "$.region",
			"zone":   "$.zone",
		},
	}

	tests := []struct {
		name      string
		dimension string
		want      bool
	}{
		{name: "reserved subject", dimension: "subject", want: true},
		{name: "reserved customer_id", dimension: "customer_id", want: true},
		{name: "valid group by", dimension: "region", want: true},
		{name: "valid group by zone", dimension: "zone", want: true},
		{name: "unknown dimension", dimension: "unknown", want: false},
		{name: "empty dimension", dimension: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSupportedGroupByDimension(m, tt.dimension))
		})
	}
}
