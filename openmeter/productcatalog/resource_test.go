package productcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_AsPath(t *testing.T) {
	tests := []struct {
		name         string
		resource     Resource
		expectedPath string
	}{
		{
			name: "no parent",
			resource: Resource{
				Parent:     nil,
				Key:        "test",
				Kind:       "plan",
				Attributes: nil,
			},
			expectedPath: "plan/test",
		},
		{
			name: "with parent",
			resource: Resource{
				Parent: &Resource{
					Parent:     nil,
					Key:        "default",
					Kind:       "namespace",
					Attributes: nil,
				},
				Key:        "premium",
				Kind:       "plan",
				Attributes: nil,
			},
			expectedPath: "namespace/default/plan/premium",
		},
		{
			name: "with attributes",
			resource: Resource{
				Parent: &Resource{
					Parent:     nil,
					Key:        "default",
					Kind:       "namespace",
					Attributes: nil,
				},
				Key:        "premium",
				Kind:       "plan",
				Attributes: map[string]any{"currency": "USD", "price": 99.99},
			},
			expectedPath: "namespace/default/plan/premium",
		},
		{
			name: "with multiple path segments in parent",
			resource: Resource{
				Parent: &Resource{
					Parent: &Resource{
						Parent:     nil,
						Key:        "default",
						Kind:       "namespace",
						Attributes: nil,
					},
					Key:        "premium",
					Kind:       "plan",
					Attributes: nil,
				},
				Key:        "trial",
				Kind:       "phase",
				Attributes: nil,
			},
			expectedPath: "namespace/default/plan/premium/phase/trial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.resource.AsPath()
			assert.Equalf(t, tt.expectedPath, path, "reource path mismatch")
		})
	}
}
