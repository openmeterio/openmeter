package feature

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestGetLastFeatures(t *testing.T) {
	tcs := []struct {
		name     string
		features []Feature
		expected map[string]string
	}{
		{
			name: "single-active",
			features: []Feature{
				{ID: "id-active", ArchivedAt: nil, Key: "feature-1-active"},
			},
			expected: map[string]string{"feature-1-active": "id-active"},
		},
		{
			name: "single-archived",
			features: []Feature{
				{ID: "id-archived", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-1-archived"},
			},
			expected: map[string]string{"feature-1-archived": "id-archived"},
		},
		{
			name: "multi-archived",
			features: []Feature{
				{ID: "id-archived", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-1"},
				{ID: "id-active", ArchivedAt: nil, Key: "feature-1"},
			},
			expected: map[string]string{"feature-1": "id-active"},
		},
		{
			name: "archived-ordering",
			features: []Feature{
				{ID: "id-archived-1", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-1"},
				{ID: "id-archived-2", ArchivedAt: lo.ToPtr(time.Now().Add(5 * time.Second)), Key: "feature-1"},
			},
			expected: map[string]string{"feature-1": "id-archived-2"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out := getLastFeatures(tc.features)

			featureKeyToID := map[string]string{}
			for key, feat := range out {
				featureKeyToID[key] = feat.ID
			}

			require.Equal(t, tc.expected, featureKeyToID)
		})
	}
}
