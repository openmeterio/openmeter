package raw_events

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
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIsQueryCachable(t *testing.T) {
	now := time.Now().UTC()

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
			result := isQueryCachable(tt.meter, tt.params)
			assert.Equal(t, tt.wantCachable, result)
		})
	}
}

// Integration test for queryMeterCached
func TestConnector_QueryMeterCached(t *testing.T) {
	mockCH := new(MockClickHouse)
	progressMgr := new(MockProgressManager)

	config := ConnectorConfig{
		Logger:          slog.Default(),
		ClickHouse:      mockCH,
		Database:        "testdb",
		EventsTableName: "events",
		ProgressManager: progressMgr,
	}

	connector := &Connector{config: config}

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

	mockRows1 := new(MockRows)
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
	mockRows2 := new(MockRows)
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
	require.NotNil(t, resultQueryMeter.From)
	assert.Equal(t, cachedEnd, *resultQueryMeter.From)

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

func TestRemainingQueryMeterFactory(t *testing.T) {
	mockCH := &MockClickHouse{}
	progressMgr := &MockProgressManager{}

	config := ConnectorConfig{
		Logger:          slog.Default(),
		ClickHouse:      mockCH,
		Database:        "testdb",
		EventsTableName: "events",
		ProgressManager: progressMgr,
	}

	connector := &Connector{config: config}

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
	assert.Equal(t, cachedTo, *resultQuery.From)
	// Original To should be preserved
	assert.Equal(t, to, *resultQuery.To)
}

func TestGetQueryMeterForCachedPeriod(t *testing.T) {
	mockCH := &MockClickHouse{}
	progressMgr := &MockProgressManager{}

	config := ConnectorConfig{
		Logger:          slog.Default(),
		ClickHouse:      mockCH,
		Database:        "testdb",
		EventsTableName: "events",
		ProgressManager: progressMgr,
	}

	connector := &Connector{config: config}

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
				To:   lo.ToPtr(now.Add(-12 * time.Hour)), // Less than minCacheableToAge
			},
			expectedError:  false,
			expectedFrom:   lo.ToPtr(now.Add(-7 * 24 * time.Hour)),
			expectedTo:     lo.ToPtr(now.Add(-minCacheableToAge).Truncate(time.Hour * 24)),
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
			expectedTo:     lo.ToPtr(now.Add(-minCacheableToAge).Truncate(time.Hour * 24)),
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
			expectedTo:     lo.ToPtr(now.Add(-minCacheableToAge).Truncate(time.Hour * 24)),
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
			expectedTo:     lo.ToPtr(now.Add(-minCacheableToAge).Truncate(time.Hour * 24)),
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

func TestMergeCachedRows(t *testing.T) {
	subject1 := "subject1"
	subject2 := "subject2"
	group1Value := "group1_value"
	group2Value := "group2_value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	windowSize := meter.WindowSizeHour

	tests := []struct {
		name       string
		meter      meter.Meter
		params     streaming.QueryParams
		cachedRows []meterpkg.MeterQueryRow
		freshRows  []meterpkg.MeterQueryRow
		wantCount  int
	}{
		{
			name: "empty cached rows",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params:     streaming.QueryParams{},
			cachedRows: []meterpkg.MeterQueryRow{},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			wantCount: 1,
		},
		{
			name: "with window size, rows are concatenated",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				WindowSize: &windowSize,
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
				},
			},
			wantCount: 2,
		},
		{
			name: "without window size, sum aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
				},
			},
			wantCount: 1, // Aggregated to a single row
		},
		{
			name: "without window size, different subjects",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject2,
				},
			},
			wantCount: 2, // One row per subject
		},
		{
			name: "without window size, with group by values",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
						"group2": &group2Value,
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
						"group2": &group2Value,
					},
				},
			},
			wantCount: 1, // Aggregated by groups
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeCachedRows(tt.meter, tt.params, tt.cachedRows, tt.freshRows)
			assert.Equal(t, tt.wantCount, len(result))

			if tt.meter.Aggregation == meter.MeterAggregationSum && len(tt.params.GroupBy) > 0 && tt.params.WindowSize == nil {
				// If we're aggregating, check that values are summed
				if len(result) == 1 && len(tt.cachedRows) > 0 && len(tt.freshRows) > 0 {
					expectedSum := tt.cachedRows[0].Value + tt.freshRows[0].Value
					assert.Equal(t, expectedSum, result[0].Value)
				}
			}
		})
	}
}

func TestGetRowGroupKey(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"
	group2Value := "group2-value"

	row := meterpkg.MeterQueryRow{
		Subject: &subject,
		GroupBy: map[string]*string{
			"group1": &group1Value,
			"group2": &group2Value,
		},
	}

	tests := []struct {
		name   string
		params streaming.QueryParams
		want   string
	}{
		{
			name: "subject only",
			params: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			want: "subject=test-subject;group=subject=nil;",
		},
		{
			name: "with group by fields",
			params: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			want: "subject=test-subject;group=group1=group1-value;group=group2=group2-value;group=subject=nil;",
		},
		{
			name: "with missing group by field",
			params: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group3"},
			},
			want: "subject=test-subject;group=group1=group1-value;group=group3=nil;group=subject=nil;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRowGroupKey(row, tt.params)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAggregateRows(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	// Rows have the same subject and groupBy values
	rows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": &group1Value,
			},
		},
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       20,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": &group1Value,
			},
		},
	}

	tests := []struct {
		name        string
		meter       meter.Meter
		rows        []meterpkg.MeterQueryRow
		wantValue   float64
		wantSubject string
	}{
		{
			name: "sum aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			rows:        rows,
			wantValue:   30, // 10 + 20
			wantSubject: subject,
		},
		{
			name: "count aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationCount,
			},
			rows:        rows,
			wantValue:   30, // count should be the same as sum
			wantSubject: subject,
		},
		{
			name: "min aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationMin,
			},
			rows:        rows,
			wantValue:   10, // min of 10 and 20
			wantSubject: subject,
		},
		{
			name: "max aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationMax,
			},
			rows:        rows,
			wantValue:   20, // max of 10 and 20
			wantSubject: subject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregateRows(tt.meter, tt.rows)

			assert.Equal(t, tt.wantValue, result.Value)
			require.NotNil(t, result.Subject)
			assert.Equal(t, tt.wantSubject, *result.Subject)

			// Window range should span from earliest to latest
			assert.Equal(t, windowStart1, result.WindowStart)
			assert.Equal(t, windowEnd2, result.WindowEnd)

			// GroupBy values should be preserved
			require.Contains(t, result.GroupBy, "group1")
			require.NotNil(t, result.GroupBy["group1"])
			assert.Equal(t, group1Value, *result.GroupBy["group1"])
		})
	}
}
