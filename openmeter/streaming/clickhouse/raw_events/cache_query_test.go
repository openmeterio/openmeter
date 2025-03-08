package raw_events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
)

// TestCreateMeterQueryRowsCacheTableToSQL tests the SQL generation of the createMeterQueryRowsCacheTable struct
func TestCreateMeterQueryRowsCacheTableToSQL(t *testing.T) {
	query := createMeterQueryRowsCacheTable{
		Database:  "openmeter",
		TableName: "meter_query_cache",
	}

	sql := query.toSQL()

	// Make sure the SQL contains important parts
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS")
	assert.Contains(t, sql, "openmeter.meter_query_cache")
	assert.Contains(t, sql, "hash String")
	assert.Contains(t, sql, "namespace String")
	assert.Contains(t, sql, "window_start DateTime")
	assert.Contains(t, sql, "window_end DateTime")
	assert.Contains(t, sql, "value Float64")
	assert.Contains(t, sql, "subject String")
	assert.Contains(t, sql, "group_by Map(String, String)")
	assert.Contains(t, sql, "ENGINE = MergeTree")
	assert.Contains(t, sql, "PARTITION BY toYYYYMM(window_start)")
	assert.Contains(t, sql, "ORDER BY (namespace, hash, window_start, window_end)")
	assert.Contains(t, sql, "TTL created_at + INTERVAL 30 DAY")
}

// TestInsertMeterQueryRowsToCache_ToSQL tests the SQL generation of the insertMeterQueryRowsToCache struct
func TestInsertMeterQueryRowsToCache_ToSQL(t *testing.T) {
	subject := "test-subject"
	groupValue := "group-value"
	now := time.Now().UTC()

	meterQueryRow := meterpkg.MeterQueryRow{
		WindowStart: now,
		WindowEnd:   now.Add(time.Hour),
		Value:       42.0,
		Subject:     &subject,
		GroupBy: map[string]*string{
			"group1": &groupValue,
		},
	}

	query := insertMeterQueryRowsToCache{
		Database:  "openmeter",
		TableName: "meter_query_cache",
		Hash:      "test-hash",
		Namespace: "test-namespace",
		QueryRows: []meterpkg.MeterQueryRow{meterQueryRow},
	}

	sql, args := query.toSQL()

	// Check SQL
	assert.Contains(t, sql, "INSERT INTO openmeter.meter_query_cache")
	assert.Contains(t, sql, "hash, namespace, window_start, window_end, value, subject, group_by")

	// Check args
	require.Len(t, args, 7)
	assert.Equal(t, "test-hash", args[0])
	assert.Equal(t, "test-namespace", args[1])
	assert.Equal(t, meterQueryRow.WindowStart, args[2])
	assert.Equal(t, meterQueryRow.WindowEnd, args[3])
	assert.Equal(t, meterQueryRow.Value, args[4])
	assert.Equal(t, "test-subject", args[5])

	// Check the group_by map
	groupByMap, ok := args[6].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "group-value", groupByMap["group1"])
}

// TestGetMeterQueryRowsFromCache_ToSQL tests the SQL generation of the getMeterQueryRowsFromCache struct
func TestGetMeterQueryRowsFromCache_ToSQL(t *testing.T) {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	tests := []struct {
		name        string
		queryParams getMeterQueryRowsFromCache
		wantSQL     string
		wantArgs    []interface{}
	}{
		{
			name: "with from and to",
			queryParams: getMeterQueryRowsFromCache{
				Database:  "openmeter",
				TableName: "meter_query_cache",
				Hash:      "test-hash",
				Namespace: "test-namespace",
				From:      &from,
				To:        &to,
			},
			wantSQL: "SELECT window_start, window_end, value, subject, group_by FROM openmeter.meter_query_cache WHERE hash = ? AND namespace = ? AND window_start >= ? AND window_end <= ? ORDER BY window_start",
		},
		{
			name: "without from",
			queryParams: getMeterQueryRowsFromCache{
				Database:  "openmeter",
				TableName: "meter_query_cache",
				Hash:      "test-hash",
				Namespace: "test-namespace",
				To:        &to,
			},
			wantSQL: "SELECT window_start, window_end, value, subject, group_by FROM openmeter.meter_query_cache WHERE hash = ? AND namespace = ? AND window_end <= ? ORDER BY window_start",
		},
		{
			name: "without to",
			queryParams: getMeterQueryRowsFromCache{
				Database:  "openmeter",
				TableName: "meter_query_cache",
				Hash:      "test-hash",
				Namespace: "test-namespace",
				From:      &from,
			},
			wantSQL: "SELECT window_start, window_end, value, subject, group_by FROM openmeter.meter_query_cache WHERE hash = ? AND namespace = ? AND window_start >= ? ORDER BY window_start",
		},
		{
			name: "without from and to",
			queryParams: getMeterQueryRowsFromCache{
				Database:  "openmeter",
				TableName: "meter_query_cache",
				Hash:      "test-hash",
				Namespace: "test-namespace",
			},
			wantSQL: "SELECT window_start, window_end, value, subject, group_by FROM openmeter.meter_query_cache WHERE hash = ? AND namespace = ? ORDER BY window_start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.queryParams.toSQL()

			// Check SQL has the right parts
			assert.Contains(t, sql, "SELECT window_start, window_end, value, subject, group_by")
			assert.Contains(t, sql, "FROM openmeter.meter_query_cache")
			assert.Contains(t, sql, "WHERE hash = ? AND namespace = ?")
			assert.Contains(t, sql, "ORDER BY window_start")

			// Check conditionals match
			if tt.queryParams.From != nil {
				assert.Contains(t, sql, "window_start >= ?")
			} else {
				assert.NotContains(t, sql, "window_start >= ?")
			}

			if tt.queryParams.To != nil {
				assert.Contains(t, sql, "window_end <= ?")
			} else {
				assert.NotContains(t, sql, "window_end <= ?")
			}

			// Check args
			assert.Contains(t, args, "test-hash")
			assert.Contains(t, args, "test-namespace")

			if tt.queryParams.From != nil {
				assert.Contains(t, args, tt.queryParams.From.Unix())
			}

			if tt.queryParams.To != nil {
				assert.Contains(t, args, tt.queryParams.To.Unix())
			}
		})
	}
}

// TestGetMeterQueryRowsFromCache_ScanRows tests the scanning of the getMeterQueryRowsFromCache struct
func TestGetMeterQueryRowsFromCache_ScanRows(t *testing.T) {
	query := getMeterQueryRowsFromCache{}

	// Test scanning a single row
	windowStart := time.Now().UTC()
	windowEnd := windowStart.Add(time.Hour)
	value := 42.0
	subject := "test-subject"
	groupBy := map[string]string{"group1": "value1", "group2": ""}

	// Set up mock to return one row
	mockRows := clickhouse.NewMockRows()
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

	rows, err := query.scanRows(mockRows)

	require.NoError(t, err)
	require.Len(t, rows, 1)

	// Verify scanned values
	assert.Equal(t, windowStart, rows[0].WindowStart)
	assert.Equal(t, windowEnd, rows[0].WindowEnd)
	assert.Equal(t, value, rows[0].Value)
	require.NotNil(t, rows[0].Subject)
	assert.Equal(t, subject, *rows[0].Subject)

	// Verify group by values
	require.Contains(t, rows[0].GroupBy, "group1")
	require.NotNil(t, rows[0].GroupBy["group1"])
	assert.Equal(t, "value1", *rows[0].GroupBy["group1"])

	// Empty string should be converted to nil
	require.Contains(t, rows[0].GroupBy, "group2")
	assert.Nil(t, rows[0].GroupBy["group2"])

	mockRows.AssertExpectations(t)
}
