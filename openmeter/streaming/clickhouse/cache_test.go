package clickhouse

import (
	"context"
	"errors"
	"log/slog"
	"sort"
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

// TestCanQueryBeCached tests the canQueryBeCached function
func TestCanQueryBeCached(t *testing.T) {
	now := time.Now().UTC()

	connector, _ := GetMockConnector()

	tests := []struct {
		name           string
		meterDef       meter.Meter
		queryParams    streaming.QueryParams
		expectCachable bool
	}{
		{
			name: "cachable is false",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				From: lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:   lo.ToPtr(now),
			},
			expectCachable: false,
		},
		{
			name: "no from time",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				To:       lo.ToPtr(now),
			},
			expectCachable: false,
		},
		{
			name: "duration too short",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-2 * 24 * time.Hour)), // Only 2 days, less than minCachableDuration
				To:       lo.ToPtr(now),
			},
			expectCachable: false,
		},
		{
			name: "non cachable aggregation",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationUniqueCount,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			expectCachable: false,
		},
		{
			name: "cachable sum query",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			expectCachable: true,
		},
		{
			name: "cachable count query",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationCount,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			expectCachable: true,
		},
		{
			name: "cachable min query",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationMin,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			expectCachable: true,
		},
		{
			name: "cachable max query",
			meterDef: meter.Meter{
				Aggregation: meter.MeterAggregationMax,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(now.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			expectCachable: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := connector.canQueryBeCached(testCase.meterDef, testCase.queryParams)
			assert.Equal(t, testCase.expectCachable, result)
		})
	}
}

// Integration test for executeQueryWithCaching
func TestConnector_ExecuteQueryWithCaching(t *testing.T) {
	connector, mockClickHouse := GetMockConnector()

	// Setup test data
	now := time.Now().UTC()
	queryFrom := now.Add(-7 * 24 * time.Hour)
	queryTo := now

	queryHash := "test-hash"

	testMeter := meterpkg.Meter{
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
		Meter:     testMeter,
		From:      &queryFrom,
		To:        &queryTo,
	}

	// Mock for fetchCachedMeterRows
	cachedStart := queryFrom
	currentCacheEnd := queryTo.Add(-5 * 24 * time.Hour).Truncate(time.Hour * 24)
	cachedEnd := queryTo.Add(-24 * time.Hour).Truncate(time.Hour * 24)

	mockRows1 := NewMockRows()
	mockClickHouse.On("Query", mock.Anything, mock.AnythingOfType("string"), []interface{}{
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
	mockClickHouse.On("Query", mock.Anything, mock.AnythingOfType("string"), []interface{}{
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
	mockClickHouse.On("Exec", mock.Anything, mock.AnythingOfType("string"), []interface{}{
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
	resultQueryMeter, results, err := connector.executeQueryWithCaching(context.Background(), queryHash, originalQueryMeter)

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

	mockClickHouse.AssertExpectations(t)
	mockRows1.AssertExpectations(t)
	mockRows2.AssertExpectations(t)
}

func TestConnector_FetchCachedMeterRows(t *testing.T) {
	connector, mockClickHouse := GetMockConnector()

	now := time.Now().UTC()
	fromTime := now.Add(-24 * time.Hour)
	toTime := now

	testQueryMeter := queryMeter{
		Database:  "testdb",
		Namespace: "test-namespace",
		From:      &fromTime,
		To:        &toTime,
	}

	// Test successful lookup
	expectedQuery := "SELECT window_start, window_end, value, subject, group_by FROM testdb.meterqueryrow_cache WHERE hash = ? AND namespace = ? AND window_start >= ? AND window_end <= ? ORDER BY window_start"
	expectedArgs := []interface{}{"test-hash", "test-namespace", fromTime.Unix(), toTime.Unix()}

	// Mock query execution
	mockRows := NewMockRows()
	mockClickHouse.On("Query", mock.Anything, expectedQuery, expectedArgs).Return(mockRows, nil)

	// Setup rows to return one value
	windowStart := fromTime
	windowEnd := fromTime.Add(time.Hour)
	rowValue := 42.0
	subject := "test-subject"
	groupBy := map[string]string{"group1": "value1"}

	mockRows.On("Next").Return(true).Once()
	mockRows.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(0).([]interface{})
		*(dest[0].(*time.Time)) = windowStart
		*(dest[1].(*time.Time)) = windowEnd
		*(dest[2].(*float64)) = rowValue
		*(dest[3].(*string)) = subject
		*(dest[4].(*map[string]string)) = groupBy
	}).Return(nil)
	mockRows.On("Next").Return(false)
	mockRows.On("Err").Return(nil)
	mockRows.On("Close").Return(nil)

	// Execute the lookup
	cachedRows, err := connector.fetchCachedMeterRows(context.Background(), "test-hash", testQueryMeter)

	require.NoError(t, err)
	require.Len(t, cachedRows, 1)

	// Verify row values
	assert.Equal(t, windowStart, cachedRows[0].WindowStart)
	assert.Equal(t, windowEnd, cachedRows[0].WindowEnd)
	assert.Equal(t, rowValue, cachedRows[0].Value)
	require.NotNil(t, cachedRows[0].Subject)
	assert.Equal(t, subject, *cachedRows[0].Subject)
	require.Contains(t, cachedRows[0].GroupBy, "group1")
	require.NotNil(t, cachedRows[0].GroupBy["group1"])
	assert.Equal(t, "value1", *cachedRows[0].GroupBy["group1"])

	mockClickHouse.AssertExpectations(t)
	mockRows.AssertExpectations(t)
}

// TestStoreCachedMeterRows tests the storeCachedMeterRows function
func TestConnector_StoreCachedMeterRows(t *testing.T) {
	connector, mockClickHouse := GetMockConnector()

	now := time.Now().UTC()
	subject := "test-subject"
	groupValue := "group-value"

	testQueryMeter := queryMeter{
		Database:  "testdb",
		Namespace: "test-namespace",
	}

	testQueryRows := []meterpkg.MeterQueryRow{
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
	mockClickHouse.On("Exec", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)

	err := connector.storeCachedMeterRows(context.Background(), "test-hash", testQueryMeter, testQueryRows)

	require.NoError(t, err)
	mockClickHouse.AssertExpectations(t)
}

// TestCreateRemainingQueryFactory tests the createRemainingQueryFactory function
func TestCreateRemainingQueryFactory(t *testing.T) {
	connector, _ := GetMockConnector()

	now := time.Now().UTC()
	fromTime := now.Add(-4 * 24 * time.Hour)
	toTime := now

	originalQuery := queryMeter{
		From: &fromTime,
		To:   &toTime,
	}

	factory := connector.createRemainingQueryFactory(originalQuery)

	// Test the factory with a cached query meter
	cachedToTime := now.Add(-1 * 24 * time.Hour)
	cachedQuery := queryMeter{
		From: &fromTime,
		To:   &cachedToTime,
	}

	resultQuery := factory(cachedQuery)

	// Should have the cached To as the new From
	require.Nil(t, resultQuery.From)
	assert.Equal(t, cachedToTime, *resultQuery.FromExclusive)
	// Original To should be preserved
	assert.Equal(t, toTime, *resultQuery.To)
}

// TestPrepareCacheableQueryPeriod tests the prepareCacheableQueryPeriod function
func TestPrepareCacheableQueryPeriod(t *testing.T) {
	connector, _ := GetMockConnector()

	now := time.Now().UTC()
	tests := []struct {
		name           string
		originalQuery  queryMeter
		expectError    bool
		expectedFrom   *time.Time
		expectedTo     *time.Time
		expectedWindow *meter.WindowSize
	}{
		{
			name: "missing from time",
			originalQuery: queryMeter{
				To: lo.ToPtr(now),
			},
			expectError: true,
		},
		{
			name: "cache `to` should be truncated to complete days and be a day before the original `to`",
			originalQuery: queryMeter{
				From: lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:   lo.ToPtr(now.Add(-12 * time.Hour)), // Less than config.minCacheableToAge
			},
			expectError:    false,
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
			expectError:    false,
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
			expectError:    false,
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
			expectError:    false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meter.WindowSizeHour),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := connector.prepareCacheableQueryPeriod(testCase.originalQuery)

			if testCase.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if testCase.expectedFrom != nil {
				assert.Equal(t, testCase.expectedFrom.Truncate(time.Second), result.From.Truncate(time.Second))
			} else {
				assert.Nil(t, result.From)
			}

			if testCase.expectedTo != nil {
				assert.Equal(t, testCase.expectedTo.Truncate(time.Second), result.To.Truncate(time.Second))
			} else {
				assert.Nil(t, result.To)
			}

			if testCase.expectedWindow != nil {
				assert.Equal(t, *testCase.expectedWindow, *result.WindowSize)
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

func TestInvalidateCache(t *testing.T) {
	connector, mockClickHouse := GetMockConnector()

	// Test case: single namespace
	mockClickHouse.On("Exec", mock.Anything, "DELETE FROM testdb.meterqueryrow_cache WHERE namespace IN (?)", []interface{}{
		[]string{"test-namespace-1"},
	}).Return(nil).Once()
	err := connector.invalidateCache(context.Background(), []string{"test-namespace-1"})
	require.NoError(t, err)

	// Test case: multiple namespaces
	mockClickHouse.On("Exec", mock.Anything, "DELETE FROM testdb.meterqueryrow_cache WHERE namespace IN (?)", []interface{}{
		[]string{"test-namespace-1", "test-namespace-2"},
	}).Return(nil).Once()
	err = connector.invalidateCache(context.Background(), []string{"test-namespace-1", "test-namespace-2"})
	require.NoError(t, err)

	// Test case: error from database
	mockClickHouse.On("Exec", mock.Anything, "DELETE FROM testdb.meterqueryrow_cache WHERE namespace IN (?)", []interface{}{
		[]string{"error-namespace"},
	}).Return(errors.New("database error")).Once()
	err = connector.invalidateCache(context.Background(), []string{"error-namespace"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete from cache: database error")

	mockClickHouse.AssertExpectations(t)
}

// TestFindNamespacesToInvalidateCache tests the findNamespacesToInvalidateCache function
func TestFindNamespacesToInvalidateCache(t *testing.T) {
	connector, _ := GetMockConnector()

	// Current time for the test
	now := time.Now().UTC()

	// Set up our test cache age parameter - from the test connector it's 24 hours
	minCacheableAge := connector.config.QueryCacheMinimumCacheableUsageAge

	// Test cases
	tests := []struct {
		name               string
		rawEvents          []streaming.RawEvent
		expectedNamespaces []string
	}{
		{
			name:               "empty events",
			rawEvents:          []streaming.RawEvent{},
			expectedNamespaces: []string{},
		},
		{
			name: "single event, not old enough",
			rawEvents: []streaming.RawEvent{
				{
					Namespace: "test-namespace-1",
					Time:      now.Add(-minCacheableAge + time.Hour), // Not old enough, no need to invalidate cache
				},
			},
			expectedNamespaces: []string{},
		},
		{
			name: "single event, old enough",
			rawEvents: []streaming.RawEvent{
				{
					Namespace: "test-namespace-1",
					Time:      now.Add(-minCacheableAge - time.Hour), // Old enough, need to invalidate cache
				},
			},
			expectedNamespaces: []string{"test-namespace-1"},
		},
		{
			name: "multiple events, different namespaces, all old enough",
			rawEvents: []streaming.RawEvent{
				{
					Namespace: "test-namespace-1",
					Time:      now.Add(-minCacheableAge - time.Hour), // Old enough, need to invalidate cache
				},
				{
					Namespace: "test-namespace-2",
					Time:      now.Add(-minCacheableAge - time.Hour), // Old enough, need to invalidate cache
				},
			},
			expectedNamespaces: []string{"test-namespace-1", "test-namespace-2"},
		},
		{
			name: "multiple events, same namespace, all old enough",
			rawEvents: []streaming.RawEvent{
				{
					Namespace: "test-namespace-1",
					Time:      now.Add(-minCacheableAge - time.Hour), // Old enough, need to invalidate cache
				},
				{
					Namespace: "test-namespace-1", // Duplicate namespace
					Time:      now.Add(-minCacheableAge - 2*time.Hour),
				},
			},
			expectedNamespaces: []string{"test-namespace-1"}, // Should be deduplicated
		},
		{
			name: "mixed ages and namespaces",
			rawEvents: []streaming.RawEvent{
				{
					Namespace: "test-namespace-1",
					Time:      now.Add(-minCacheableAge - time.Hour), // Old enough
				},
				{
					Namespace: "test-namespace-2",
					Time:      now.Add(-minCacheableAge + time.Hour), // Not old enough
				},
				{
					Namespace: "test-namespace-3",
					Time:      now.Add(-minCacheableAge - 2*time.Hour), // Old enough
				},
				{
					Namespace: "test-namespace-1",                      // Duplicate namespace
					Time:      now.Add(-minCacheableAge - 3*time.Hour), // Old enough
				},
			},
			expectedNamespaces: []string{"test-namespace-1", "test-namespace-3"}, // test-namespace-2 not included, duplicates removed
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := connector.findNamespacesToInvalidateCache(testCase.rawEvents)

			// Since the order of the namespaces isn't guaranteed, we need to sort both slices
			// before comparison to ensure consistent test results
			sort.Strings(result)
			sort.Strings(testCase.expectedNamespaces)

			assert.Equal(t, testCase.expectedNamespaces, result)
		})
	}
}
