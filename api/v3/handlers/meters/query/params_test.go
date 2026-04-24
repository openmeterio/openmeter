package query

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newTestMeter() meter.Meter {
	return meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
		},
		GroupBy: map[string]string{
			"region": "$.region",
			"zone":   "$.zone",
		},
	}
}

func noopCustomerResolver(_ context.Context, _ string, _ []string) ([]customer.Customer, error) {
	return nil, nil
}

func TestBuildQueryParams_Empty(t *testing.T) {
	m := newTestMeter()
	body := api.MeterQueryRequest{}

	params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
	require.NoError(t, err)
	assert.Nil(t, params.From)
	assert.Nil(t, params.To)
	assert.Nil(t, params.WindowSize)
	assert.Nil(t, params.WindowTimeZone)
	assert.Empty(t, params.GroupBy)
}

func TestBuildQueryParams_FromTo(t *testing.T) {
	m := newTestMeter()
	now := time.Now().UTC().Truncate(time.Second)
	body := api.MeterQueryRequest{
		From: &now,
		To:   lo.ToPtr(now.Add(time.Hour)),
	}

	params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
	require.NoError(t, err)
	assert.Equal(t, &now, params.From)
	assert.Equal(t, lo.ToPtr(now.Add(time.Hour)), params.To)
}

func TestBuildQueryParams_Granularity(t *testing.T) {
	m := newTestMeter()

	t.Run("valid granularity", func(t *testing.T) {
		gran := api.MeterQueryGranularity("PT1H")
		body := api.MeterQueryRequest{Granularity: &gran}

		params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.NoError(t, err)
		require.NotNil(t, params.WindowSize)
		assert.Equal(t, meter.WindowSizeHour, *params.WindowSize)
	})

	t.Run("invalid granularity", func(t *testing.T) {
		gran := api.MeterQueryGranularity("P1Y")
		body := api.MeterQueryRequest{Granularity: &gran}

		_, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.Error(t, err)
	})
}

func TestBuildQueryParams_TimeZone(t *testing.T) {
	m := newTestMeter()

	t.Run("valid timezone", func(t *testing.T) {
		body := api.MeterQueryRequest{TimeZone: lo.ToPtr("America/New_York")}

		params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.NoError(t, err)
		require.NotNil(t, params.WindowTimeZone)
		assert.Equal(t, "America/New_York", params.WindowTimeZone.String())
	})

	t.Run("invalid timezone", func(t *testing.T) {
		body := api.MeterQueryRequest{TimeZone: lo.ToPtr("Invalid/Zone")}

		_, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.Error(t, err)
	})
}

func TestBuildQueryParams_GroupByDimensions(t *testing.T) {
	m := newTestMeter()

	t.Run("valid dimensions", func(t *testing.T) {
		body := api.MeterQueryRequest{
			GroupByDimensions: &[]string{"subject", "region"},
		}

		params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.NoError(t, err)
		assert.Equal(t, []string{"subject", "region"}, params.GroupBy)
	})

	t.Run("invalid dimension", func(t *testing.T) {
		body := api.MeterQueryRequest{
			GroupByDimensions: &[]string{"nonexistent"},
		}

		_, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.Error(t, err)
	})
}

func TestBuildQueryParams_SubjectFilter(t *testing.T) {
	m := newTestMeter()

	t.Run("eq filter adds subject to group by", func(t *testing.T) {
		body := api.MeterQueryRequest{
			Filters: &api.MeterQueryFilters{
				Dimensions: &map[string]api.QueryFilterStringMapItem{
					DimensionSubject: {Eq: lo.ToPtr("user-1")},
				},
			},
		}

		params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.NoError(t, err)
		assert.Equal(t, []string{"user-1"}, params.FilterSubject)
		assert.Contains(t, params.GroupBy, DimensionSubject)
	})

	t.Run("does not duplicate subject in group by", func(t *testing.T) {
		body := api.MeterQueryRequest{
			GroupByDimensions: &[]string{"subject"},
			Filters: &api.MeterQueryFilters{
				Dimensions: &map[string]api.QueryFilterStringMapItem{
					DimensionSubject: {Eq: lo.ToPtr("user-1")},
				},
			},
		}

		params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.NoError(t, err)
		count := 0
		for _, g := range params.GroupBy {
			if g == DimensionSubject {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})
}

func TestBuildQueryParams_CustomerFilter(t *testing.T) {
	m := newTestMeter()

	t.Run("resolves customers and adds to group by", func(t *testing.T) {
		resolver := func(_ context.Context, _ string, ids []string) ([]customer.Customer, error) {
			var customers []customer.Customer
			for _, id := range ids {
				customers = append(customers, customer.Customer{
					ManagedResource: models.ManagedResource{
						NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
						ID:              id,
					},
				})
			}
			return customers, nil
		}

		body := api.MeterQueryRequest{
			Filters: &api.MeterQueryFilters{
				Dimensions: &map[string]api.QueryFilterStringMapItem{
					DimensionCustomerID: {In: &[]string{"c1", "c2"}},
				},
			},
		}

		params, err := BuildQueryParams(context.Background(), m, body, resolver)
		require.NoError(t, err)
		assert.Len(t, params.FilterCustomer, 2)
		assert.Contains(t, params.GroupBy, DimensionCustomerID)
	})
}

func TestBuildQueryParams_DimensionFilters(t *testing.T) {
	m := newTestMeter()

	t.Run("valid dimension filter", func(t *testing.T) {
		body := api.MeterQueryRequest{
			Filters: &api.MeterQueryFilters{
				Dimensions: &map[string]api.QueryFilterStringMapItem{
					"region": {Eq: lo.ToPtr("us-east-1")},
				},
			},
		}

		params, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.NoError(t, err)
		require.Contains(t, params.FilterGroupBy, "region")
	})

	t.Run("invalid dimension name", func(t *testing.T) {
		body := api.MeterQueryRequest{
			Filters: &api.MeterQueryFilters{
				Dimensions: &map[string]api.QueryFilterStringMapItem{
					"nonexistent": {Eq: lo.ToPtr("val")},
				},
			},
		}

		_, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.Error(t, err)
	})

	t.Run("multiple operators in dimension filter", func(t *testing.T) {
		body := api.MeterQueryRequest{
			Filters: &api.MeterQueryFilters{
				Dimensions: &map[string]api.QueryFilterStringMapItem{
					"region": {
						Eq:  lo.ToPtr("us-east"),
						Neq: lo.ToPtr("us-west"),
					},
				},
			},
		}

		_, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
		require.Error(t, err)
	})
}

func TestBuildQueryParams_MultipleSubjectFilterOperators(t *testing.T) {
	m := newTestMeter()
	body := api.MeterQueryRequest{
		Filters: &api.MeterQueryFilters{
			Dimensions: &map[string]api.QueryFilterStringMapItem{
				DimensionSubject: {
					Eq:  lo.ToPtr("a"),
					Neq: lo.ToPtr("b"),
				},
			},
		},
	}

	_, err := BuildQueryParams(context.Background(), m, body, noopCustomerResolver)
	require.Error(t, err)
}

func TestCustomersToStreaming(t *testing.T) {
	customers := []customer.Customer{
		{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ID:              "c1",
			},
		},
		{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ID:              "c2",
			},
		},
	}

	result := CustomersToStreaming(customers)
	require.Len(t, result, 2)
}
