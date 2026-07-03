package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestMeterCacheHealBound(t *testing.T) {
	// dirty window above the floor: bound reduces to two refresh intervals
	assert.Equal(t, 20*time.Minute, meterCacheHealBound(time.Hour, 10*time.Minute))

	// dirty window floored at one hour: bound is what the floor leaves after age and one
	// interval
	assert.Equal(t, 45*time.Minute, meterCacheHealBound(10*time.Minute, 5*time.Minute))
}

func TestMeterCacheStaticReject(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	kathmandu, err := time.LoadLocation("Asia/Kathmandu")
	require.NoError(t, err)
	lordHowe, err := time.LoadLocation("Australia/Lord_Howe")
	require.NoError(t, err)

	parse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return ts
	}

	baseMeter := meter.Meter{
		Key:           "meter1",
		EventType:     "event1",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       map[string]string{"group1": "$.group1"},
	}

	baseCache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	baseParams := func() streaming.QueryParams {
		return streaming.QueryParams{
			Cachable:   true,
			From:       lo.ToPtr(parse("2025-01-01T00:00:00Z")),
			To:         lo.ToPtr(parse("2025-02-01T00:00:00Z")),
			WindowSize: lo.ToPtr(meter.WindowSizeHour),
		}
	}

	tests := []struct {
		name string

		meter   func(m meter.Meter) meter.Meter
		params  func(p streaming.QueryParams) streaming.QueryParams
		cache   func(c CacheConfig) CacheConfig
		decimal *bool

		want cacheRejectReason
	}{
		{
			name: "windowed query is admitted",
			want: cacheRejectReasonNone,
		},
		{
			name: "total with from is admitted (G10)",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = nil
				return p
			},
			want: cacheRejectReasonNone,
		},
		{
			name: "window size above grain is admitted",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = lo.ToPtr(meter.WindowSizeMonth)
				return p
			},
			want: cacheRejectReasonNone,
		},
		{
			name: "whole hour offset timezone is admitted",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowTimeZone = newYork
				return p
			},
			want: cacheRejectReasonNone,
		},
		{
			name: "empty stored at filter is admitted",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.FilterStoredAt = &filter.FilterTimeUnix{}
				return p
			},
			want: cacheRejectReasonNone,
		},
		{
			name: "not opted in",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.Cachable = false
				return p
			},
			want: cacheRejectReasonNotOptedIn,
		},
		{
			name: "cache disabled",
			cache: func(c CacheConfig) CacheConfig {
				c.Enabled = false
				return c
			},
			want: cacheRejectReasonCacheDisabled,
		},
		{
			name:    "decimal precision disabled",
			decimal: lo.ToPtr(false),
			want:    cacheRejectReasonDecimalDisabled,
		},
		{
			name: "missing to",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.To = nil
				return p
			},
			want: cacheRejectReasonNoTo,
		},
		{
			name: "total without from",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = nil
				p.From = nil
				return p
			},
			want: cacheRejectReasonTotalWithoutFrom,
		},
		{
			name: "window size below grain",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = lo.ToPtr(meter.WindowSizeMinute)
				return p
			},
			want: cacheRejectReasonWindowBelowGrain,
		},
		{
			name: "second window size",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = lo.ToPtr(meter.WindowSizeSecond)
				return p
			},
			want: cacheRejectReasonWindowBelowGrain,
		},
		{
			name: "invalid grain",
			cache: func(c CacheConfig) CacheConfig {
				c.WindowSize = CacheGrain("week")
				return c
			},
			want: cacheRejectReasonInvalidGrain,
		},
		{
			name: "fractional offset timezone",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowTimeZone = kathmandu
				return p
			},
			want: cacheRejectReasonTimezone,
		},
		{
			// Lord Howe uses +10:30 in standard time and +11:00 in DST: a range inside
			// the DST period is admissible, one covering standard time is not — the
			// weekly offset sampling must see the fractional period.
			name: "half hour standard time zone inside whole hour dst period is admitted",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowTimeZone = lordHowe
				p.From = lo.ToPtr(parse("2025-01-05T00:00:00Z"))
				p.To = lo.ToPtr(parse("2025-01-20T00:00:00Z"))
				return p
			},
			want: cacheRejectReasonNone,
		},
		{
			name: "half hour standard time period is rejected by sampling",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowTimeZone = lordHowe
				p.From = lo.ToPtr(parse("2025-01-05T00:00:00Z"))
				p.To = lo.ToPtr(parse("2025-06-20T00:00:00Z"))
				return p
			},
			want: cacheRejectReasonTimezone,
		},
		{
			name: "non utc timezone with unbounded from",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowTimeZone = newYork
				p.From = nil
				return p
			},
			want: cacheRejectReasonTimezone,
		},
		{
			name: "day grain requires utc",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = lo.ToPtr(meter.WindowSizeDay)
				p.WindowTimeZone = newYork
				return p
			},
			cache: func(c CacheConfig) CacheConfig {
				c.WindowSize = CacheGrainDay
				return c
			},
			want: cacheRejectReasonDayGrainTimezone,
		},
		{
			name: "day grain with utc is admitted",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.WindowSize = lo.ToPtr(meter.WindowSizeDay)
				return p
			},
			cache: func(c CacheConfig) CacheConfig {
				c.WindowSize = CacheGrainDay
				return c
			},
			want: cacheRejectReasonNone,
		},
		{
			name: "customer filter",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.FilterCustomer = []streaming.Customer{nil}
				return p
			},
			want: cacheRejectReasonFilterCustomer,
		},
		{
			name: "customer id group by",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.GroupBy = []string{"customer_id"}
				return p
			},
			want: cacheRejectReasonCustomerIDGroupBy,
		},
		{
			name: "client id progress tracking",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.ClientID = lo.ToPtr("client-1")
				return p
			},
			want: cacheRejectReasonClientID,
		},
		{
			name: "stored at filter",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.FilterStoredAt = &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{Lt: lo.ToPtr(parse("2025-01-15T00:00:00Z"))},
				}
				return p
			},
			want: cacheRejectReasonFilterStoredAt,
		},
		{
			name: "group by outside the meter",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.GroupBy = []string{"group2"}
				return p
			},
			want: cacheRejectReasonGroupByUnknownKey,
		},
		{
			name: "filter group by outside the meter",
			params: func(p streaming.QueryParams) streaming.QueryParams {
				p.FilterGroupBy = map[string]filter.FilterString{"group2": {Eq: lo.ToPtr("a")}}
				return p
			},
			want: cacheRejectReasonFilterGroupByUnknown,
		},
		{
			name: "reserved alias group by key",
			meter: func(m meter.Meter) meter.Meter {
				m.GroupBy = map[string]string{"namespace": "$.namespace"}
				return m
			},
			want: cacheRejectReasonReservedAlias,
		},
		{
			name: "reserved suffix group by key",
			meter: func(m meter.Meter) meter.Meter {
				m.GroupBy = map[string]string{"custom_value": "$.custom"}
				return m
			},
			want: cacheRejectReasonReservedAlias,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseMeter
			if tt.meter != nil {
				m = tt.meter(m)
			}

			params := baseParams()
			if tt.params != nil {
				params = tt.params(params)
			}

			cache := baseCache
			if tt.cache != nil {
				cache = tt.cache(cache)
			}

			decimal := true
			if tt.decimal != nil {
				decimal = *tt.decimal
			}

			assert.Equal(t, tt.want, meterCacheStaticReject(m, params, cache, decimal))
		})
	}
}

// fakeCountRow implements driver.Row for the gate's marker overlap scan.
type fakeCountRow struct {
	count uint64
	err   error
}

func (r fakeCountRow) Err() error { return r.err }

func (r fakeCountRow) ScanStruct(any) error { return fmt.Errorf("not implemented") }

func (r fakeCountRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	*(dest[0].(*uint64)) = r.count

	return nil
}

func TestMeterCacheGateEligibility(t *testing.T) {
	baseMeter := meter.Meter{
		Key:           "meter1",
		EventType:     "event1",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	cache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	hash := meterHash(baseMeter, cache.WindowSize)

	now := time.Now().UTC()
	lastSuccess := now.Add(-time.Minute)
	refreshStart := lastSuccess.Add(-5 * time.Second)

	healthyState := func() meterCacheViewState {
		return meterCacheViewState{
			Exists:     true,
			MetadataOK: true,
			Metadata: meterCacheMVMetadata{
				MeterKey:     baseMeter.Key,
				EventType:    baseMeter.EventType,
				MeterHash:    formatCacheHash(hash),
				DDLHash:      formatCacheHash(hash),
				BackfilledAt: lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			LastSuccessTime:       &lastSuccess,
			LastSuccessDurationMS: lo.ToPtr(uint64(5000)),
		}
	}

	newGate := func(state meterCacheViewState, unhealedMarkers uint64) *meterCacheGate {
		mockCH := NewMockClickHouse()
		mockCH.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).
			Return(fakeCountRow{count: unhealedMarkers})

		gate := &meterCacheGate{
			logger:                 slog.Default(),
			clickhouse:             mockCH,
			database:               "openmeter",
			cache:                  cache,
			enableDecimalPrecision: true,
			viewStateTTL:           meterCacheViewStateTTL,
			viewStates:             map[string]meterCacheViewStateEntry{},
		}

		gate.fetchViewState = func(ctx context.Context, viewName string) (meterCacheViewState, error) {
			return state, nil
		}

		return gate
	}

	baseQuery := func() (queryMeter, streaming.QueryParams) {
		from := now.Add(-5 * time.Hour)
		to := now

		params := streaming.QueryParams{
			Cachable:   true,
			From:       &from,
			To:         &to,
			WindowSize: lo.ToPtr(meter.WindowSizeHour),
		}

		return queryMeter{
			Database:               "openmeter",
			EventsTableName:        "om_events",
			Namespace:              "my_namespace",
			Meter:                  baseMeter,
			From:                   params.From,
			To:                     params.To,
			WindowSize:             params.WindowSize,
			EnableDecimalPrecision: true,
		}, params
	}

	t.Run("healthy stamped view with healed markers is eligible", func(t *testing.T) {
		gate := newGate(healthyState(), 0)
		query, params := baseQuery()

		bounds, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		require.Equal(t, cacheRejectReasonNone, reason)

		wantBounds, ok, err := meterCacheBounds(query.From, *query.To, refreshStart, cache.MinimumUsageAge, cache.WindowSize)
		require.NoError(t, err)
		require.True(t, ok)
		require.NotNil(t, bounds.CacheLo)
		assert.True(t, bounds.CacheLo.Equal(*wantBounds.CacheLo))
		assert.True(t, bounds.CacheHi.Equal(wantBounds.CacheHi))
	})

	t.Run("static reject short circuits before any lookup", func(t *testing.T) {
		gate := newGate(healthyState(), 0)
		gate.fetchViewState = func(ctx context.Context, viewName string) (meterCacheViewState, error) {
			t.Fatal("view state must not be fetched for statically rejected queries")
			return meterCacheViewState{}, nil
		}

		query, params := baseQuery()
		params.Cachable = false

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonNotOptedIn, reason)
	})

	t.Run("missing view", func(t *testing.T) {
		gate := newGate(meterCacheViewState{Exists: false}, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonViewMissing, reason)
	})

	t.Run("unparseable metadata is foreign", func(t *testing.T) {
		state := healthyState()
		state.MetadataOK = false

		gate := newGate(state, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonViewForeign, reason)
	})

	t.Run("same shape sibling meter owning the view is foreign", func(t *testing.T) {
		state := healthyState()
		state.Metadata.MeterKey = "meter2"

		gate := newGate(state, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonViewForeign, reason)
	})

	t.Run("unstamped backfill (G3)", func(t *testing.T) {
		state := healthyState()
		state.Metadata.BackfilledAt = nil

		gate := newGate(state, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonBackfillUnstamped, reason)
	})

	t.Run("refresh exception", func(t *testing.T) {
		state := healthyState()
		state.Exception = "boom"

		gate := newGate(state, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonViewException, reason)
	})

	t.Run("never refreshed view is stale", func(t *testing.T) {
		state := healthyState()
		state.LastSuccessTime = nil
		state.LastSuccessDurationMS = nil

		gate := newGate(state, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonViewStale, reason)
	})

	t.Run("refresh older than three intervals is stale", func(t *testing.T) {
		state := healthyState()
		state.LastSuccessTime = lo.ToPtr(now.Add(-31 * time.Minute))

		gate := newGate(state, 0)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonViewStale, reason)
	})

	t.Run("range without a settled bucket is empty", func(t *testing.T) {
		gate := newGate(healthyState(), 0)
		query, params := baseQuery()

		from := now.Add(-30 * time.Minute)
		params.From = &from
		query.From = &from

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonEmptyCacheRange, reason)
	})

	t.Run("unhealed markers send the whole query live (G1)", func(t *testing.T) {
		gate := newGate(healthyState(), 1)
		query, params := baseQuery()

		_, reason, err := gate.cacheEligibility(t.Context(), query, params)
		require.NoError(t, err)
		assert.Equal(t, cacheRejectReasonUnhealedMarkers, reason)
	})

	t.Run("view state fetch failure is an error", func(t *testing.T) {
		gate := newGate(healthyState(), 0)
		gate.fetchViewState = func(ctx context.Context, viewName string) (meterCacheViewState, error) {
			return meterCacheViewState{}, fmt.Errorf("boom")
		}

		query, params := baseQuery()

		_, _, err := gate.cacheEligibility(t.Context(), query, params)
		require.Error(t, err)
	})
}

func TestMeterCacheGateViewStateMemoization(t *testing.T) {
	fetches := 0

	gate := &meterCacheGate{
		logger:       slog.Default(),
		viewStateTTL: meterCacheViewStateTTL,
		viewStates:   map[string]meterCacheViewStateEntry{},
	}

	gate.fetchViewState = func(ctx context.Context, viewName string) (meterCacheViewState, error) {
		fetches++
		return meterCacheViewState{Exists: true}, nil
	}

	// given:
	// - two lookups within the TTL
	// then:
	// - only one fetch hits the backend (G13)
	_, err := gate.viewState(t.Context(), "view1")
	require.NoError(t, err)
	_, err = gate.viewState(t.Context(), "view1")
	require.NoError(t, err)
	assert.Equal(t, 1, fetches)

	// given:
	// - a different view
	// then:
	// - the memo is per view
	_, err = gate.viewState(t.Context(), "view2")
	require.NoError(t, err)
	assert.Equal(t, 2, fetches)

	// given:
	// - the entry aged past the TTL
	// then:
	// - the next lookup re-fetches
	entry := gate.viewStates["view1"]
	entry.fetchedAt = time.Now().Add(-gate.viewStateTTL - time.Second)
	gate.viewStates["view1"] = entry

	_, err = gate.viewState(t.Context(), "view1")
	require.NoError(t, err)
	assert.Equal(t, 3, fetches)
}
