package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestCanQueryBeCached tests the canQueryBeCached function
func TestCanQueryBeCached(t *testing.T) {
	now := time.Now().UTC()

	// Use a date before the minimum cacheable usage age
	to := now.Add(-24 * time.Hour)

	getConnector := func(opts ...MockConnectorOption) *Connector {
		connector, _ := GetMockConnector(t, opts...)
		return connector
	}

	tests := []struct {
		name           string
		connector      *Connector
		namespace      string
		meterDef       meterpkg.Meter
		queryParams    streaming.QueryParams
		expectCachable bool
	}{
		{
			name:      "cachable is false",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				From: lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:   lo.ToPtr(to),
			},
			expectCachable: false,
		},
		{
			name:      "namespace template does not match",
			connector: getConnector(WithQueryCacheNamespaceTemplate("^test-")),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: false,
		},
		{
			name:      "namespace template matches",
			connector: getConnector(WithQueryCacheNamespaceTemplate("^test-[a-z]+$")),
			namespace: "test-namespace",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: true,
		},
		{
			name:      "no from time",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				To:       lo.ToPtr(to),
			},
			expectCachable: false,
		},
		{
			name:      "from age is before minimum cacheable usage age",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				// Now is younger than the minimum cacheable usage age
				From: lo.ToPtr(now),
				To:   lo.ToPtr(now),
			},
			expectCachable: false,
		},
		{
			name:      "duration too short",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-2 * 24 * time.Hour)), // Only 2 days, less than minCachableDuration
				To:       lo.ToPtr(to),
			},
			expectCachable: false,
		},
		{
			name:      "non cachable aggregation",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationUniqueCount,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: false,
		},
		{
			name:      "cachable sum query",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: true,
		},
		{
			name:      "cachable sum query with to set to now",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(now),
			},
			expectCachable: true,
		},
		{
			name:      "cachable count query",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationCount,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: true,
		},
		{
			name:      "cachable min query",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationMin,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: true,
		},
		{
			name:      "cachable max query",
			connector: getConnector(),
			namespace: "default",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationMax,
			},
			queryParams: streaming.QueryParams{
				Cachable: true,
				From:     lo.ToPtr(to.Add(-4 * 24 * time.Hour)),
				To:       lo.ToPtr(to),
			},
			expectCachable: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := testCase.connector.canQueryBeCached(testCase.namespace, testCase.meterDef, testCase.queryParams)
			assert.Equal(t, testCase.expectCachable, result)
		})
	}
}

// Integration test for executeQueryWithCaching
func TestConnector_ExecuteQueryWithCaching(t *testing.T) {
	connector, mockClickHouse := GetMockConnector(t)

	// Setup test data
	now := time.Now().UTC().Truncate(time.Hour * 24)
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
		queryTo.Unix(),
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

	// FIXME: materialize all the 49 rows with zero values
	// Store new cachable data in cache
	// mockClickHouse.On("Exec", mock.Anything, mock.AnythingOfType("string"), []interface{}{
	// 	// Called with the new data
	// 	"test-hash",
	// 	"test-namespace",
	// 	currentCacheEnd,
	// 	cachedEnd,
	// 	50.0,
	// 	"", // subject
	// 	map[string]string{},
	// }).Return(nil).Once()

	mockClickHouse.On("Exec", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Once()

	// Execute query with caching
	resultRows, err := connector.executeQueryWithCaching(context.Background(), queryHash, originalQueryMeter)
	require.NoError(t, err)

	// Validate rows returned
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
	}, resultRows)

	mockClickHouse.AssertExpectations(t)
	mockRows1.AssertExpectations(t)
	mockRows2.AssertExpectations(t)
}

// Integration test for executeQueryWithCaching when the query is covered by the cache and no remaining query is needed
func TestConnector_ExecuteQueryWithCaching_QueryCovered_NoRemainingQuery(t *testing.T) {
	connector, mockClickHouse := GetMockConnector(t)

	// Setup test data
	now := time.Now().UTC().Truncate(time.Hour * 24)
	queryFrom := now.Add(-7 * 24 * time.Hour).Truncate(time.Hour * 24)
	queryTo := now.Add(-6 * 24 * time.Hour).Truncate(time.Hour * 24)

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
	currentCacheEnd := queryTo
	cachedEnd := currentCacheEnd

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
	}).Return(nil).Once()
	mockRows1.On("Next").Return(false)
	mockRows1.On("Err").Return(nil)
	mockRows1.On("Close").Return(nil)

	// Execute query with caching
	resultRows, err := connector.executeQueryWithCaching(context.Background(), queryHash, originalQueryMeter)
	require.NoError(t, err)

	// Validate combined results
	assert.Equal(t, []meterpkg.MeterQueryRow{
		{
			WindowStart: cachedStart,
			WindowEnd:   currentCacheEnd,
			Value:       100.0,
			GroupBy:     map[string]*string{},
		},
	}, resultRows)

	mockClickHouse.AssertExpectations(t)
	mockRows1.AssertExpectations(t)
}

func TestConnector_FetchCachedMeterRows(t *testing.T) {
	connector, mockClickHouse := GetMockConnector(t)

	now := time.Now().UTC().Truncate(time.Hour * 24)
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
	connector, mockClickHouse := GetMockConnector(t)

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

// TestPrepareCacheableQueryPeriod tests the prepareCacheableQueryPeriod function
func TestPrepareCacheableQueryPeriod(t *testing.T) {
	connector, _ := GetMockConnector(t)

	now := time.Now().UTC()
	tests := []struct {
		name           string
		originalQuery  queryMeter
		expectError    bool
		expectedFrom   *time.Time
		expectedTo     *time.Time
		expectedWindow *meterpkg.WindowSize
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
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour).Truncate(time.Hour * 24)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meterpkg.WindowSizeDay),
		},
		{
			name: "cache `to` should be truncated to complete day",
			originalQuery: queryMeter{
				From: lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:   lo.ToPtr(now.Add(-36 * time.Hour)), // Less than minCacheableToAge
			},
			expectError:    false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour).Truncate(time.Hour * 24)),
			expectedTo:     lo.ToPtr(now.Add(-36 * time.Hour).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meterpkg.WindowSizeDay),
		},
		{
			name: "set window size if not provided",
			originalQuery: queryMeter{
				From: lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
				To:   lo.ToPtr(now.Add(-12 * time.Hour)),
			},
			expectError:    false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour).Truncate(time.Hour * 24)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour * 24)),
			expectedWindow: lo.ToPtr(meterpkg.WindowSizeDay),
		},
		{
			name: "use provided window size should round down to the window size",
			originalQuery: queryMeter{
				From:       lo.ToPtr(now.Add(-7 * 24 * time.Hour).Truncate(time.Hour * 24)),
				To:         lo.ToPtr(now.Add(-12 * time.Hour)),
				WindowSize: lo.ToPtr(meterpkg.WindowSizeHour),
			},
			expectError:    false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour).Truncate(time.Hour * 24)),
			expectedTo:     lo.ToPtr(now.Add(-connector.config.QueryCacheMinimumCacheableUsageAge).Truncate(time.Hour)),
			expectedWindow: lo.ToPtr(meterpkg.WindowSizeHour),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cachedQuery, err := connector.prepareCacheableQueryPeriod(testCase.originalQuery)

			if testCase.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if testCase.expectedFrom != nil {
				assert.Equal(t, testCase.expectedFrom.Truncate(time.Second), cachedQuery.From.Truncate(time.Second))
			} else {
				assert.Nil(t, cachedQuery.From)
			}

			if testCase.expectedTo != nil {
				assert.Equal(t, testCase.expectedTo.Truncate(time.Second), cachedQuery.To.Truncate(time.Second))
			} else {
				assert.Nil(t, cachedQuery.To)
			}

			if testCase.expectedWindow != nil {
				assert.Equal(t, *testCase.expectedWindow, *cachedQuery.WindowSize)
			} else {
				assert.Nil(t, cachedQuery.WindowSize)
			}

			// Verify To is truncated
			assert.Equal(t, cachedQuery.To.Minute(), 0)
			assert.Equal(t, cachedQuery.To.Second(), 0)
		})
	}
}

func TestInvalidateCache(t *testing.T) {
	connector, mockClickHouse := GetMockConnector(t)

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
	connector, _ := GetMockConnector(t)

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

// TestGetMockConnectorOptions verifies that the connector options work correctly
func TestGetMockConnectorOptions(t *testing.T) {
	template := "custom_template"
	connector, _ := GetMockConnector(t, WithQueryCacheNamespaceTemplate(template))
	assert.Equal(t, template, connector.config.QueryCacheNamespaceTemplate)
}

func TestQueryParamsHash(t *testing.T) {
	tests := []struct {
		name  string
		query streaming.QueryParams
		want  string
	}{
		{
			name: "should hash with from and to",
			query: streaming.QueryParams{
				From: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				To:   lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			want: "c9d48eb8da92c8f",
		},
		{
			name: "should hash with only from",
			query: streaming.QueryParams{
				From: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			want: "c9d48eb8da92c8f", // same as above
		},
		{
			name: "should hash with subject filter",
			query: streaming.QueryParams{
				FilterSubject: []string{"subject1", "subject2"},
			},
			want: "98e1492cbb349227",
		},
		{
			name: "should hash with subject filter in different order",
			query: streaming.QueryParams{
				FilterSubject: []string{"subject2", "subject1"},
			},
			want: "98e1492cbb349227", // same as above
		},
		{
			name: "should hash with group by filter",
			query: streaming.QueryParams{
				FilterGroupBy: map[string][]string{
					"group1": {"value1.1", "value1.2"},
					"group2": {"value2.1", "value2.2"},
				},
			},
			want: "4c76a15ce8dc6716",
		},
		{
			name: "should hash with group by filter in different order",
			query: streaming.QueryParams{
				FilterGroupBy: map[string][]string{
					"group2": {"value2.2", "value2.1"},
					"group1": {"value1.2", "value1.1"},
				},
			},
			want: "4c76a15ce8dc6716", // same as above
		},
		{
			name: "should hash with group by",
			query: streaming.QueryParams{
				GroupBy: []string{"group1", "group2"},
			},
			want: "ea31d545920a914",
		},
		{
			name: "should hash with group by in different order",
			query: streaming.QueryParams{
				GroupBy: []string{"group2", "group1"},
			},
			want: "ea31d545920a914", // same as above
		},
		{
			name:  "should hash with default window size",
			query: streaming.QueryParams{},
			want:  "c9d48eb8da92c8f",
		},
		{
			name: "should hash with same as default window size",
			query: streaming.QueryParams{
				WindowSize: lo.ToPtr(meterpkg.WindowSizeDay),
			},
			want: "c9d48eb8da92c8f", // same as above
		},
		{
			name: "should hash with different window size",
			query: streaming.QueryParams{
				WindowSize: lo.ToPtr(meterpkg.WindowSizeHour),
			},
			want: "8ebd1ee24821c2ce",
		},
		{
			name:  "should hash with default time zone",
			query: streaming.QueryParams{},
			want:  "c9d48eb8da92c8f",
		},
		{
			name: "should hash with same as default time zone",
			query: streaming.QueryParams{
				WindowTimeZone: time.FixedZone("UTC", 0),
			},
			want: "c9d48eb8da92c8f", // same as above
		},
		{
			name: "should hash with different window time zone",
			query: streaming.QueryParams{
				WindowTimeZone: time.FixedZone("Europe/Budapest", 3600),
			},
			want: "7b273f0f20bda726",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			key := QueryParamsHash(tt.query)

			assert.Equal(t, tt.want, fmt.Sprintf("%x", key))
		})
	}
}
