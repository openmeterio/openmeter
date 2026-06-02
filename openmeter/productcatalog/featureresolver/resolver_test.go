package featureresolver_test

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func Test_NamespacedFeatureResolver(t *testing.T) {
	// Setup test environment
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := pctestutils.NewTestNamespace(t)

	// Setup meter repository
	err := env.Meter.ReplaceMeters(t.Context(), pctestutils.NewTestMeters(t, namespace))
	require.NoError(t, err, "replacing meters must not fail")

	result, err := env.Meter.ListMeters(t.Context(), meter.ListMetersParams{
		Page: pagination.Page{
			PageSize:   1000,
			PageNumber: 1,
		},
		Namespace: namespace,
	})
	require.NoErrorf(t, err, "listing meters must not fail")

	meters := result.Items
	require.NotEmptyf(t, meters, "list of Meters must not be empty")

	// Set a feature for each meter
	features := make([]feature.Feature, 0, len(meters))
	for _, m := range meters {
		input := pctestutils.NewTestFeatureFromMeter(t, &m)

		feat, err := env.Feature.CreateFeature(t.Context(), input)
		require.NoErrorf(t, err, "creating feature must not fail")
		require.NotNil(t, feat, "feature must not be empty")

		features = append(features, feat)
	}
	require.NotEmptyf(t, features, "list of Features must not be empty")
	require.Lenf(t, features, len(meters), "list of Features must have the same length as the list of Meters")

	resolver, err := featureresolver.New(env.Feature)
	require.NoError(t, err, "creating feature resolver must not fail")

	namespacedResolver := resolver.WithNamespace(namespace)

	t.Run("Resolve", func(t *testing.T) {
		tests := []struct {
			name          string
			featureID     *string
			featureKey    *string
			expectedError error
		}{
			{
				name:          "nil",
				featureID:     nil,
				featureKey:    nil,
				expectedError: errors.New("feature id or key is required"),
			},
			{
				name:          "by id",
				featureID:     &features[0].ID,
				featureKey:    nil,
				expectedError: nil,
			},
			{
				name:          "by key",
				featureID:     nil,
				featureKey:    &features[0].Key,
				expectedError: nil,
			},
			{
				name:          "by both id and key",
				featureID:     &features[0].ID,
				featureKey:    &features[0].Key,
				expectedError: nil,
			},
			{
				name:          "by non-existing id",
				featureID:     lo.ToPtr("abracadabraId"),
				featureKey:    nil,
				expectedError: new(models.GenericNotFoundError),
			},
			{
				name:          "by non-existing key",
				featureID:     nil,
				featureKey:    lo.ToPtr("abracadabraKey"),
				expectedError: new(models.GenericNotFoundError),
			},
			{
				name:          "by non-existing id and key",
				featureID:     lo.ToPtr("abracadabraId"),
				featureKey:    lo.ToPtr("abracadabraKey"),
				expectedError: new(models.GenericNotFoundError),
			},
			{
				name:          "mismatched id and key",
				featureID:     &features[0].ID,
				featureKey:    &features[1].Key,
				expectedError: new(models.GenericConflictError),
			},
			{
				name:          "id is actually a key",
				featureID:     &features[0].Key,
				featureKey:    nil,
				expectedError: new(models.GenericConflictError),
			},
			{
				name:          "key is actually an id",
				featureID:     nil,
				featureKey:    &features[0].ID,
				expectedError: new(models.GenericConflictError),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var f *feature.Feature

				f, err = namespacedResolver.Resolve(t.Context(), test.featureID, test.featureKey)
				if test.expectedError != nil {
					assert.ErrorAsf(t, err, &test.expectedError, "expected error %v", test.expectedError)
				} else {
					require.NoErrorf(t, err, "expected no error: %v", err)

					if test.featureID != nil {
						assert.Equalf(t, *test.featureID, f.ID, "resolved feature id must be equal to the one we set")
					}

					if test.featureKey != nil {
						assert.Equalf(t, *test.featureKey, f.Key, "resolved feature key must be equal to the one we set")
					}
				}
			})
		}
	})

	t.Run("BatchResolve", func(t *testing.T) {
		testBatch := map[string]bool{
			features[0].ID:  true,
			features[0].Key: true,
			features[1].ID:  true,
			features[1].Key: true,
			features[2].ID:  true,
			features[2].Key: true,
			"abracadabra":   false,
		}

		idOrKeys := lo.MapToSlice(testBatch, func(key string, _ bool) string {
			return key
		})

		var resolved map[string]*feature.Feature

		resolved, err = namespacedResolver.BatchResolve(t.Context(), idOrKeys...)
		require.NoErrorf(t, err, "expected no error: %v", err)

		for k, ok := range testBatch {
			if ok {
				assert.NotNilf(t, resolved[k], "resolved feature must not be nil")
				assert.True(t, resolved[k].ID == k || resolved[k].Key == k, "resolved feature id or key must be equal to the one we set")
			} else {
				assert.Nilf(t, resolved[k], "resolved feature must be nil")
			}
		}
	})
}
