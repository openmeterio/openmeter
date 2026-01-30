package clickhouse

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// MockConnectorOption is a function that configures the Connector
type MockConnectorOption func(Config) Config

// GetMockConnector returns a new Connector with a mock ClickHouse
func GetMockConnector(t *testing.T, opts ...MockConnectorOption) (*Connector, *MockClickHouse) {
	mockClickhouse := NewMockClickHouse()

	config := Config{
		Logger:           slog.Default(),
		ClickHouse:       mockClickhouse,
		Database:         "testdb",
		EventsTableName:  "events",
		ProgressManager:  progressmanager.NewMockProgressManager(),
		SkipCreateTables: true,
	}

	// Apply options
	for _, opt := range opts {
		config = opt(config)
	}

	// Create the connector
	connector, err := New(context.Background(), config)
	require.NoError(t, err)

	return connector, mockClickhouse
}

// TestConnector_QueryMeter tests the queryMeter function
func TestConnector_QueryMeter(t *testing.T) {
	mockCH := NewMockClickHouse()

	config := Config{
		Logger:          slog.Default(),
		ClickHouse:      mockCH,
		Database:        "testdb",
		EventsTableName: "events",
		ProgressManager: progressmanager.NewMockProgressManager(),
	}

	connector := &Connector{config: config}

	// Create test data
	subject := "test-subject"
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now
	windowSize := meter.WindowSizeHour

	queryMeter := queryMeter{
		Database:        "testdb",
		EventsTableName: "events",
		Namespace:       "test-namespace",
		Meter: meter.Meter{
			Key:           "test-meter",
			EventType:     "test-event",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		},
		From:       &from,
		To:         &to,
		WindowSize: &windowSize,
	}

	// Mock the SQL query and response
	mockRows1 := NewMockRows()
	mockCH.On("Query", mock.Anything, mock.AnythingOfType("string"), []interface{}{
		"test-namespace",
		"test-event",
		from.Unix(),
		to.Unix(),
	}).Run(func(args mock.Arguments) {
		// Verify the arguments contain expected values
		queryArgs := args[2].([]interface{})
		assert.Contains(t, queryArgs, "test-namespace")
		assert.Contains(t, queryArgs, "test-event")
		assert.Contains(t, queryArgs, from.Unix())
		assert.Contains(t, queryArgs, to.Unix())
	}).Return(mockRows1, nil)

	// Set up mock rows to return test data
	windowStart := from
	windowEnd := from.Add(time.Hour)
	value := 42.0

	mockRows1.On("Columns").Return([]string{"windowstart", "windowend", "value"})
	mockRows1.On("Next").Return(true).Once()
	mockRows1.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(0).([]interface{})
		// Different scanRows implementations might use different indices, adjust accordingly
		*(dest[0].(*time.Time)) = windowStart
		*(dest[1].(*time.Time)) = windowEnd
		*(dest[2].(*float64)) = value
		// If there are more fields used in scanRows, set them appropriately
		if len(dest) > 3 {
			*(dest[3].(*string)) = subject
		}
	}).Return(nil)
	mockRows1.On("Next").Return(false)
	mockRows1.On("Err").Return(nil)
	mockRows1.On("Close").Return(nil)

	// Call the method being tested
	results, err := connector.queryMeter(context.Background(), queryMeter)

	// Verify the results
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, windowStart, results[0].WindowStart)
	assert.Equal(t, windowEnd, results[0].WindowEnd)
	assert.Equal(t, value, results[0].Value)

	// Test handling of errors
	// 1. Query error
	queryErrorMsg := "query error"
	mockCH.ExpectedCalls = nil
	mockRows2 := NewMockRows()
	mockCH.On("Query", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(mockRows2, errors.New(queryErrorMsg))

	_, err = connector.queryMeter(context.Background(), queryMeter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), queryErrorMsg)

	// 2. Meter not found error
	mockCH.ExpectedCalls = nil
	mockRows3 := NewMockRows()
	mockCH.On("Query", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(mockRows3, errors.New("code: 60"))

	_, err = connector.queryMeter(context.Background(), queryMeter)
	assert.Error(t, err)
	meterErr, ok := err.(*meter.MeterNotFoundError)
	assert.True(t, ok)
	assert.Equal(t, "not found error: meter not found: test-meter", meterErr.Error())

	// 3. Scan error
	scanErrorMsg := "scan error"
	mockCH.ExpectedCalls = nil
	mockRows4 := NewMockRows()
	mockCH.On("Query", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(mockRows4, nil)
	mockRows4.On("Columns").Return([]string{"windowstart", "windowend", "value"})
	mockRows4.On("Next").Return(true).Once()
	mockRows4.On("Scan", mock.Anything).Return(errors.New(scanErrorMsg))
	mockRows4.On("Close").Return(nil)

	_, err = connector.queryMeter(context.Background(), queryMeter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), scanErrorMsg)

	mockCH.AssertExpectations(t)
	mockRows1.AssertExpectations(t)
	mockRows2.AssertExpectations(t)
	mockRows3.AssertExpectations(t)
	mockRows4.AssertExpectations(t)
}

func TestBatchInsert(t *testing.T) {
	connector, mockCH := GetMockConnector(t)

	ctx := context.Background()
	now := time.Now().UTC()

	// Set up test events
	events := []streaming.RawEvent{
		{
			Namespace: "test-namespace-1",
			ID:        "1",
			Time:      now.Add(-48 * time.Hour),
		},
	}

	// Mock the batch insert
	mockCH.On("Exec", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Once()

	// Execute the method
	err := connector.BatchInsert(ctx, events)
	require.NoError(t, err)

	// Verify mocks were called
	mockCH.AssertExpectations(t)
}
