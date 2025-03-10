package clickhouse

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func GetMockConnector() (*Connector, *MockClickHouse) {
	mockClickhouse := NewMockClickHouse()

	config := Config{
		Logger:                                slog.Default(),
		ClickHouse:                            mockClickhouse,
		Database:                              "testdb",
		EventsTableName:                       "events",
		ProgressManager:                       progressmanager.NewMockProgressManager(),
		QueryCacheEnabled:                     true,
		QueryCacheMinimumCacheableQueryPeriod: 3 * 24 * time.Hour,
		QueryCacheMinimumCacheableUsageAge:    24 * time.Hour,
	}

	connector := &Connector{config: config}

	return connector, mockClickhouse
}

// TestIsQueryCachable tests the isQueryCachable function
func TestIsQueryCachable(t *testing.T) {
	now := time.Now().UTC()

	connector, _ := GetMockConnector()

	tests := []struct {
		name         string
		meter        meter.Meter
		params       streaming.QueryParams
		wantCachable bool
	}{
		{
			name: "cachable is false",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				From: lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:   lo.ToPtr(now),
			},
			wantCachable: false,
		},
		{
			name: "no from time",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				Cachable: true,
				To:       lo.ToPtr(now),
			},
			wantCachable: false,
		},
		{
			name: "duration too short",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-2 * 24 * time.Hour)), // Only 2 days, less than minCachableDuration
				To:       lo.ToPtr(now),
			},
			wantCachable: false,
		},
		{
			name: "non cachable aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationUniqueCount,
			},
			params: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			wantCachable: false,
		},
		{
			name: "cachable sum query",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			wantCachable: true,
		},
		{
			name: "cachable count query",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationCount,
			},
			params: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			wantCachable: true,
		},
		{
			name: "cachable min query",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationMin,
			},
			params: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			wantCachable: true,
		},
		{
			name: "cachable max query",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationMax,
			},
			params: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			wantCachable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.isQueryCachable(tt.meter, tt.params)
			assert.Equal(t, tt.wantCachable, result)
		})
	}
}

// Integration test for queryMeterCached
func TestConnector_QueryMeterCached(t *testing.T) {
	connector, mockCH := GetMockConnector()

	// Setup test data
	now := time.Now().UTC()
	queryFrom := now.Add(-7 * 24 * time.Hour)
	queryTo := now

	hash := "test-hash"

	meter := meterpkg.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: "test-namespace",
			},
			ID:   "test-meter",
			Name: "test-meter",
		},
		Key:           "test-meter",
		Aggregation:   meterpkg.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	originalQueryMeter := queryMeter{
		Database:  "testdb",
		Namespace: "test-namespace",
		Meter:     meter,
		From:      &queryFrom,
		To:        &queryTo,
	}

	// Mock for lookupCachedMeterRows
	cachedStart := queryFrom
	currentCacheEnd := queryTo.Add(-5 * 24 * time.Hour).Truncate(time.Hour * 24)
	cachedEnd := queryTo.Add(-24 * time.Hour).Truncate(time.Hour * 24)

	mockRows1 := NewMockRows()
	mockCH.On("Query", mock.Anything, mock.AnythingOfType("string"), []interface{}{
		"test-hash",
		"test-namespace",
		// We query for the full cached period
		cachedStart.Unix(),
		cachedEnd.Unix(),
	}).Return(mockRows1, nil).Once()

	// Setup rows to return from cache
	mockRows1.On("Next").Return(true).Once()
	mockRows1.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(0).([]interface{})
		*(dest[0].(*time.Time)) = cachedStart
		*(dest[1].(*time.Time)) = currentCacheEnd
		*(dest[2].(*float64)) = 100.0
	}).Return(nil)
	mockRows1.On("Next").Return(false)
	mockRows1.On("Err").Return(nil)
	mockRows1.On("Close").Return(nil)

	// Mock the SQL query for loading new data to the cache
	mockRows2 := NewMockRows()
	mockCH.On("Query", mock.Anything, mock.AnythingOfType("string"), []interface{}{
		"test-namespace",
		"",
		// We query for the period we don't have cache but could have
		currentCacheEnd.Unix(),
		cachedEnd.Unix(),
	}).Return(mockRows2, nil).Once()

	// Setup rows to return new data that can be cached
	mockRows2.On("Next").Return(true).Once()
	mockRows2.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(0).([]interface{})

		*(dest[0].(*time.Time)) = currentCacheEnd
		*(dest[1].(*time.Time)) = cachedEnd
		*(dest[2].(**float64)) = lo.ToPtr(50.0)
	}).Return(nil)
	mockRows2.On("Next").Return(false)
	mockRows2.On("Err").Return(nil)
	mockRows2.On("Close").Return(nil)

	// Store new cachable data in cache
	mockCH.On("Exec", mock.Anything, mock.AnythingOfType("string"), []interface{}{
		// Called with the new data
		"test-hash",
		"test-namespace",
		currentCacheEnd,
		cachedEnd,
		50.0,
		"", // subject
		map[string]string{},
	}).Return(nil).Once()

	// Execute query with caching
	resultQueryMeter, results, err := connector.queryMeterCached(context.Background(), hash, originalQueryMeter)

	require.NoError(t, err)
	assert.Len(t, results, 2) // Should have both cached and fresh rows

	// Result query meter should have From set to the end of cached period
	require.Nil(t, resultQueryMeter.From)
	require.NotNil(t, resultQueryMeter.FromExclusive)
	assert.Equal(t, cachedEnd, *resultQueryMeter.FromExclusive)

	// Validate combined results
	assert.Equal(t, []meterpkg.MeterQueryRow{
		{
			WindowStart: cachedStart,
			WindowEnd:   currentCacheEnd,
			Value:       100.0,
			GroupBy:     map[string]*string{},
		},
		{
			WindowStart: currentCacheEnd,
			WindowEnd:   cachedEnd,
			Value:       50.0,
			GroupBy:     map[string]*string{},
		},
	}, results)

	mockCH.AssertExpectations(t)
	mockRows1.AssertExpectations(t)
	mockRows2.AssertExpectations(t)
}

func TestConnector_LookupCachedMeterRows(t *testing.T) {
	connector, mockCH := GetMockConnector()

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	queryMeter := queryMeter{
		Database:  "testdb",
		Namespace: "test-namespace",
		From:      &from,
		To:        &to,
	}

	// Test successful lookup
	expectedQuery := "SELECT window_start, window_end, value, subject, group_by FROM testdb.meterqueryrow_cache WHERE hash = ? AND namespace = ? AND window_start >= ? AND window_end <= ? ORDER BY window_start"
	expectedArgs := []interface{}{"test-hash", "test-namespace", from.Unix(), to.Unix()}

	// Mock query execution
	mockRows := NewMockRows()
	mockCH.On("Query", mock.Anything, expectedQuery, expectedArgs).Return(mockRows, nil)

	// Setup rows to return one value
	windowStart := from
	windowEnd := from.Add(time.Hour)
	value := 42.0
	subject := "test-subject"
	groupBy := map[string]string{"group1": "value1"}

	mockRows.On("Next").Return(true).Once()
	mockRows.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(0).([]interface{})
		*(dest[0].(*time.Time)) = windowStart
		*(dest[1].(*time.Time)) = windowEnd
		*(dest[2].(*float64)) = value
		*(dest[3].(*string)) = subject
		*(dest[4].(*map[string]string)) = groupBy
	}).Return(nil)
	mockRows.On("Next").Return(false)
	mockRows.On("Err").Return(nil)
	mockRows.On("Close").Return(nil)

	// Execute the lookup
	rows, err := connector.lookupCachedMeterRows(context.Background(), "test-hash", queryMeter)

	require.NoError(t, err)
	require.Len(t, rows, 1)

	// Verify row values
	assert.Equal(t, windowStart, rows[0].WindowStart)
	assert.Equal(t, windowEnd, rows[0].WindowEnd)
	assert.Equal(t, value, rows[0].Value)
	require.NotNil(t, rows[0].Subject)
	assert.Equal(t, subject, *rows[0].Subject)
	require.Contains(t, rows[0].GroupBy, "group1")
	require.NotNil(t, rows[0].GroupBy["group1"])
	assert.Equal(t, "value1", *rows[0].GroupBy["group1"])

	mockCH.AssertExpectations(t)
	mockRows.AssertExpectations(t)
}

// TestInsertRowsToCache tests the insertRowsToCache function
func TestConnector_InsertRowsToCache(t *testing.T) {
	connector, mockCH := GetMockConnector()

	now := time.Now().UTC()
	subject := "test-subject"
	groupValue := "group-value"

	queryMeter := queryMeter{
		Database:  "testdb",
		Namespace: "test-namespace",
	}

	queryRows := []meterpkg.MeterQueryRow{
		{
			WindowStart: now,
			WindowEnd:   now.Add(time.Hour),
			Value:       42.0,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": &groupValue,
			},
		},
	}

	// We don't need to check the exact SQL, just that the Exec is called with something
	mockCH.On("Exec", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)

	err := connector.insertRowsToCache(context.Background(), "test-hash", queryMeter, queryRows)

	require.NoError(t, err)
	mockCH.AssertExpectations(t)
}

// TestRemainingQueryMeterFactory tests the remainingQueryMeterFactory function
func TestRemainingQueryMeterFactory(t *testing.T) {
	connector, _ := GetMockConnector()

	now := time.Now().UTC()
	from := now.Add(-4 * 24 * time.Hour)
	to := now

	originalQuery := queryMeter{
		From: &from,
		To:   &to,
	}

	factory := connector.remainingQueryMeterFactory(originalQuery)

	// Test the factory with a cached query meter
	cachedTo := now.Add(-1 * 24 * time.Hour)
	cachedQuery := queryMeter{
		From: &from,
		To:   &cachedTo,
	}

	resultQuery := factory(cachedQuery)

	// Should have the cached To as the new From
	require.Nil(t, resultQuery.From)
	assert.Equal(t, cachedTo, *resultQuery.FromExclusive)
	// Original To should be preserved
	assert.Equal(t, to, *resultQuery.To)
}

// TestGetQueryMeterForCachedPeriod tests the getQueryMeterForCachedPeriod function
func TestGetQueryMeterForCachedPeriod(t *testing.T) {
	connector, _ := GetMockConnector()

	now := time.Now().UTC()
	tests := []struct {
		name           string
		originalQuery  queryMeter
		expectedError  bool
		expectedFrom   *time.Time
		expectedTo     *time.Time
		expectedWindow *meter.WindowSize
	}{
		{
			name: "missing from time",
			originalQuery: queryMeter{
				To: lo.ToPtr(now),
			},
			expectedError: true,
		},
		{
			name: "cache `to` should be truncated to complete days and be a day before the original `to`",
			originalQuery: queryMeter{
				From: lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:   lo.ToPtr(now.Add(-12 * time.Hour)), // Less than config.minCacheableToAge
			},
			expectedError:  false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meter.WindowSizeDay),
		},
		{
			name: "cache `to` should be truncated to complete day",
			originalQuery: queryMeter{
				From: lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:   lo.ToPtr(now.Add(-36 * time.Hour)), // Less than minCacheableToAge
			},
			expectedError:  false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
			expectedTo:     lo.ToPtr(now.Add(-36 * time.Hour).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meter.WindowSizeDay),
		},
		{
			name: "set window size if not provided",
			originalQuery: queryMeter{
				From: lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:   lo.ToPtr(now.Add(-12 * time.Hour)),
			},
			expectedError:  false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meter.WindowSizeDay),
		},
		{
			name: "use provided window size",
			originalQuery: queryMeter{
				From:       lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:         lo.ToPtr(now.Add(-12 * time.Hour)),
				WindowSize: lo.ToPtr(meter.WindowSizeHour),
			},
			expectedError:  false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meter.WindowSizeHour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := connector.getQueryMeterForCachedPeriod(tt.originalQuery)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.expectedFrom != nil {
				assert.Equal(t, tt.expectedFrom.Truncate(time.Second), result.From.Truncate(time.Second))
			} else {
				assert.Nil(t, result.From)
			}

			if tt.expectedTo != nil {
				assert.Equal(t, tt.expectedTo.Truncate(time.Second), result.To.Truncate(time.Second))
			} else {
				assert.Nil(t, result.To)
			}

			if tt.expectedWindow != nil {
				assert.Equal(t, *tt.expectedWindow, *result.WindowSize)
			} else {
				assert.Nil(t, result.WindowSize)
			}

			// Verify To is truncated to complete days
			assert.Equal(t, result.To.Hour(), 0)
			assert.Equal(t, result.To.Minute(), 0)
			assert.Equal(t, result.To.Second(), 0)
		})
	}
}
