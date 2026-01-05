package meterexportservice

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterexport"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MockMeterService is a mock implementation of meter.Service for testing
type MockMeterService struct {
	meters map[string]meter.Meter
}

func NewMockMeterService() *MockMeterService {
	return &MockMeterService{
		meters: make(map[string]meter.Meter),
	}
}

func (m *MockMeterService) AddMeter(met meter.Meter) {
	key := met.Namespace + "/" + met.ID
	m.meters[key] = met
}

func (m *MockMeterService) GetMeterByIDOrSlug(ctx context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	key := input.Namespace + "/" + input.IDOrSlug
	if met, ok := m.meters[key]; ok {
		return met, nil
	}
	return meter.Meter{}, meter.NewMeterNotFoundError(input.IDOrSlug)
}

func (m *MockMeterService) ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return pagination.Result[meter.Meter]{}, nil
}

func TestExportSyntheticMeterData(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)

	tests := []struct {
		name           string
		meter          meter.Meter
		config         meterexport.DataExportConfig
		events         []testutils.SimpleEvent
		wantErr        bool
		wantErrMsg     string
		expectedEvents int
	}{
		{
			name: "should export SUM meter data",
			meter: meter.Meter{
				ManagedResource: models.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
					ManagedModel: models.ManagedModel{
						CreatedAt: now.Add(-time.Hour),
						UpdatedAt: now.Add(-time.Hour),
					},
					ID:   "meter-1",
					Name: "Test Meter",
				},
				Key:           "test-meter",
				Aggregation:   meter.MeterAggregationSum,
				EventType:     "test-event",
				ValueProperty: lo.ToPtr("$.value"),
			},
			config: meterexport.DataExportConfig{
				ExportWindowSize: meter.WindowSizeMinute,
				MeterID: models.NamespacedID{
					Namespace: "test-ns",
					ID:        "meter-1",
				},
				Period: timeutil.StartBoundedPeriod{
					From: now.Add(-10 * time.Minute),
					To:   lo.ToPtr(now),
				},
			},
			events: []testutils.SimpleEvent{
				{MeterSlug: "test-meter", Value: 10.0, Time: now.Add(-9 * time.Minute)},
				{MeterSlug: "test-meter", Value: 20.0, Time: now.Add(-5 * time.Minute)},
				{MeterSlug: "test-meter", Value: 30.0, Time: now.Add(-2 * time.Minute)},
			},
			expectedEvents: 3,
		},
		{
			name: "should export COUNT meter data",
			meter: meter.Meter{
				ManagedResource: models.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
					ManagedModel: models.ManagedModel{
						CreatedAt: now.Add(-time.Hour),
						UpdatedAt: now.Add(-time.Hour),
					},
					ID:   "meter-count",
					Name: "Count Meter",
				},
				Key:         "count-meter",
				Aggregation: meter.MeterAggregationCount,
				EventType:   "count-event",
			},
			config: meterexport.DataExportConfig{
				ExportWindowSize: meter.WindowSizeMinute,
				MeterID: models.NamespacedID{
					Namespace: "test-ns",
					ID:        "meter-count",
				},
				Period: timeutil.StartBoundedPeriod{
					From: now.Add(-5 * time.Minute),
					To:   lo.ToPtr(now),
				},
			},
			events: []testutils.SimpleEvent{
				{MeterSlug: "count-meter", Value: 1.0, Time: now.Add(-4 * time.Minute)},
				{MeterSlug: "count-meter", Value: 1.0, Time: now.Add(-3 * time.Minute)},
			},
			expectedEvents: 2,
		},
		{
			name: "should reject unsupported aggregation type",
			meter: meter.Meter{
				ManagedResource: models.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
					ManagedModel: models.ManagedModel{
						CreatedAt: now.Add(-time.Hour),
						UpdatedAt: now.Add(-time.Hour),
					},
					ID:   "meter-avg",
					Name: "Avg Meter",
				},
				Key:           "avg-meter",
				Aggregation:   meter.MeterAggregationAvg,
				EventType:     "avg-event",
				ValueProperty: lo.ToPtr("$.value"),
			},
			config: meterexport.DataExportConfig{
				ExportWindowSize: meter.WindowSizeMinute,
				MeterID: models.NamespacedID{
					Namespace: "test-ns",
					ID:        "meter-avg",
				},
				Period: timeutil.StartBoundedPeriod{
					From: now.Add(-5 * time.Minute),
					To:   lo.ToPtr(now),
				},
			},
			wantErr:    true,
			wantErrMsg: "unsupported meter aggregation",
		},
		{
			name:  "should fail validation with missing meter ID",
			meter: meter.Meter{},
			config: meterexport.DataExportConfig{
				ExportWindowSize: meter.WindowSizeMinute,
				MeterID: models.NamespacedID{
					Namespace: "test-ns",
					ID:        "",
				},
				Period: timeutil.StartBoundedPeriod{
					From: now.Add(-5 * time.Minute),
					To:   lo.ToPtr(now),
				},
			},
			wantErr:    true,
			wantErrMsg: "meter id is required",
		},
		{
			name:  "should fail when meter not found",
			meter: meter.Meter{},
			config: meterexport.DataExportConfig{
				ExportWindowSize: meter.WindowSizeMinute,
				MeterID: models.NamespacedID{
					Namespace: "test-ns",
					ID:        "non-existent",
				},
				Period: timeutil.StartBoundedPeriod{
					From: now.Add(-5 * time.Minute),
					To:   lo.ToPtr(now),
				},
			},
			wantErr:    true,
			wantErrMsg: "get meter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockMeterService := NewMockMeterService()
			if tt.meter.ID != "" {
				mockMeterService.AddMeter(tt.meter)
			}

			mockStreaming := testutils.NewMockStreamingConnector(t)
			for _, e := range tt.events {
				mockStreaming.AddSimpleEvent(e.MeterSlug, e.Value, e.Time)
			}

			// Create service
			svc, err := New(Config{
				EventSourceGroup:   "test-source",
				StreamingConnector: mockStreaming,
				MeterService:       mockMeterService,
			})
			require.NoError(t, err)

			// Setup channels
			resultCh := make(chan streaming.RawEvent, 100)
			errCh := make(chan error, 10)

			// Execute
			ctx := context.Background()
			err = svc.ExportSyntheticMeterData(ctx, tt.config, resultCh, errCh)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, err)

			// Get descriptor separately
			descriptor, err := svc.GetTargetMeterDescriptor(ctx, tt.config)
			require.NoError(t, err)

			// Collect results
			var events []streaming.RawEvent
			for e := range resultCh {
				events = append(events, e)
			}

			// Drain error channel
			var errs []error
			for e := range errCh {
				errs = append(errs, e)
			}
			require.Empty(t, errs, "unexpected errors: %v", errs)

			// Verify events
			assert.Len(t, events, tt.expectedEvents)

			// Verify descriptor
			assert.Equal(t, meter.MeterAggregationSum, descriptor.Aggregation)
			assert.Equal(t, tt.meter.EventType, descriptor.EventType)
			assert.NotNil(t, descriptor.ValueProperty)
			assert.Equal(t, SUM_VALUE_PROPERTY_KEY, *descriptor.ValueProperty)

			// Verify event structure
			for _, e := range events {
				assert.Equal(t, tt.meter.Namespace, e.Namespace)
				assert.Equal(t, tt.meter.EventType, e.Type)
				assert.NotEmpty(t, e.ID)
				assert.Contains(t, e.Source, "test-source")

				// Verify data contains value property
				var data map[string]interface{}
				err := json.Unmarshal([]byte(e.Data), &data)
				require.NoError(t, err)
				assert.Contains(t, data, SUM_VALUE_PROPERTY_KEY)
			}
		})
	}
}

func TestExportSyntheticMeterData_ContextCancellation(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)

	// Setup meter
	testMeter := meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
			ManagedModel: models.ManagedModel{
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now.Add(-time.Hour),
			},
			ID:   "meter-1",
			Name: "Test Meter",
		},
		Key:           "test-meter",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test-event",
		ValueProperty: lo.ToPtr("$.value"),
	}

	t.Run("should stop operation when context is canceled mid-export", func(t *testing.T) {
		mockMeterService := NewMockMeterService()
		mockMeterService.AddMeter(testMeter)

		mockStreaming := testutils.NewMockStreamingConnector(t)
		// Add many events across a very long time period to force multiple query iterations
		// The funnel queries in batches of TARGET_ROWS_PER_QUERY (500) windows
		// Using MINUTE window size with 1000+ minutes ensures multiple iterations
		for i := 0; i < 1000; i++ {
			mockStreaming.AddSimpleEvent("test-meter", float64(i+1), now.Add(-time.Duration(i)*time.Minute))
		}

		svc, err := New(Config{
			EventSourceGroup:   "test-source",
			StreamingConnector: mockStreaming,
			MeterService:       mockMeterService,
		})
		require.NoError(t, err)

		// Use unbuffered channel to create backpressure so we can control timing
		resultCh := make(chan streaming.RawEvent)
		errCh := make(chan error, 10)

		ctx, cancel := context.WithCancel(context.Background())

		config := meterexport.DataExportConfig{
			ExportWindowSize: meter.WindowSizeMinute,
			MeterID: models.NamespacedID{
				Namespace: "test-ns",
				ID:        "meter-1",
			},
			Period: timeutil.StartBoundedPeriod{
				From: now.Add(-1000 * time.Minute),
				To:   lo.ToPtr(now),
			},
		}

		// Start export in background
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = svc.ExportSyntheticMeterData(ctx, config, resultCh, errCh)
		}()

		// Receive first event to confirm operation started
		select {
		case <-resultCh:
			// Good, operation is running
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for first event")
		}

		// Cancel the context - simulating caller deciding to stop
		cancel()

		// Drain result channel concurrently to avoid deadlock
		// (the producer might be blocked trying to send)
		go func() {
			for range resultCh {
			}
		}()

		// Wait for operation to finish
		<-done

		// Check error channel for exactly one context.Canceled error (no duplicates)
		var errs []error
		for e := range errCh {
			errs = append(errs, e)
		}

		contextCanceledCount := 0
		for _, err := range errs {
			if err == context.Canceled {
				contextCanceledCount++
			}
		}
		assert.Equal(t, 1, contextCanceledCount, "expected exactly one context.Canceled error, got %d in: %v", contextCanceledCount, errs)
	})

	t.Run("should handle pre-canceled context", func(t *testing.T) {
		mockMeterService := NewMockMeterService()
		mockMeterService.AddMeter(testMeter)

		mockStreaming := testutils.NewMockStreamingConnector(t)
		mockStreaming.AddSimpleEvent("test-meter", 10.0, now.Add(-5*time.Minute))

		svc, err := New(Config{
			EventSourceGroup:   "test-source",
			StreamingConnector: mockStreaming,
			MeterService:       mockMeterService,
		})
		require.NoError(t, err)

		resultCh := make(chan streaming.RawEvent, 10)
		errCh := make(chan error, 10)

		// Create an already-canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		config := meterexport.DataExportConfig{
			ExportWindowSize: meter.WindowSizeMinute,
			MeterID: models.NamespacedID{
				Namespace: "test-ns",
				ID:        "meter-1",
			},
			Period: timeutil.StartBoundedPeriod{
				From: now.Add(-10 * time.Minute),
				To:   lo.ToPtr(now),
			},
		}

		exportErr := svc.ExportSyntheticMeterData(ctx, config, resultCh, errCh)

		// The function itself doesn't return an error for context cancellation
		// (it's a streaming operation - errors go to channel)
		require.NoError(t, exportErr)

		// Drain channels
		for range resultCh {
		}

		var errs []error
		for e := range errCh {
			errs = append(errs, e)
		}

		// Should have exactly one context.Canceled on error channel (no duplicates)
		contextCanceledCount := 0
		for _, err := range errs {
			if err == context.Canceled {
				contextCanceledCount++
			}
		}
		assert.Equal(t, 1, contextCanceledCount, "expected exactly one context.Canceled error, got %d in: %v", contextCanceledCount, errs)
	})
}

func TestServiceNew(t *testing.T) {
	t.Run("should fail without streaming connector", func(t *testing.T) {
		_, err := New(Config{
			MeterService: NewMockMeterService(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "streaming connector is required")
	})

	t.Run("should fail without meter service", func(t *testing.T) {
		_, err := New(Config{
			StreamingConnector: testutils.NewMockStreamingConnector(t),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "meter service is required")
	})

	t.Run("should fail without event source group", func(t *testing.T) {
		_, err := New(Config{
			StreamingConnector: testutils.NewMockStreamingConnector(t),
			MeterService:       NewMockMeterService(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event source group is required")
	})

	t.Run("should succeed with valid config", func(t *testing.T) {
		svc, err := New(Config{
			StreamingConnector: testutils.NewMockStreamingConnector(t),
			MeterService:       NewMockMeterService(),
			EventSourceGroup:   "test-source",
		})
		require.NoError(t, err)
		assert.NotNil(t, svc)
	})
}

func TestExportSyntheticMeterDataIter(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)

	testMeter := meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
			ManagedModel: models.ManagedModel{
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now.Add(-time.Hour),
			},
			ID:   "meter-1",
			Name: "Test Meter",
		},
		Key:           "test-meter",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test-event",
		ValueProperty: lo.ToPtr("$.value"),
	}

	t.Run("should iterate over exported events", func(t *testing.T) {
		mockMeterService := NewMockMeterService()
		mockMeterService.AddMeter(testMeter)

		mockStreaming := testutils.NewMockStreamingConnector(t)
		mockStreaming.AddSimpleEvent("test-meter", 10.0, now.Add(-9*time.Minute))
		mockStreaming.AddSimpleEvent("test-meter", 20.0, now.Add(-5*time.Minute))
		mockStreaming.AddSimpleEvent("test-meter", 30.0, now.Add(-2*time.Minute))

		svc, err := New(Config{
			EventSourceGroup:   "test-source",
			StreamingConnector: mockStreaming,
			MeterService:       mockMeterService,
		})
		require.NoError(t, err)

		config := meterexport.DataExportConfig{
			ExportWindowSize: meter.WindowSizeMinute,
			MeterID: models.NamespacedID{
				Namespace: "test-ns",
				ID:        "meter-1",
			},
			Period: timeutil.StartBoundedPeriod{
				From: now.Add(-10 * time.Minute),
				To:   lo.ToPtr(now),
			},
		}

		// Get descriptor first
		descriptor, err := svc.GetTargetMeterDescriptor(context.Background(), config)
		require.NoError(t, err)

		// Verify descriptor
		assert.Equal(t, meter.MeterAggregationSum, descriptor.Aggregation)
		assert.Equal(t, testMeter.EventType, descriptor.EventType)
		assert.NotNil(t, descriptor.ValueProperty)

		seq, err := svc.ExportSyntheticMeterDataIter(context.Background(), config)
		require.NoError(t, err)

		// Collect events from iterator
		var events []streaming.RawEvent
		var errs []error
		for event, err := range seq {
			if err != nil {
				errs = append(errs, err)
				continue
			}
			events = append(events, event)
		}

		assert.Empty(t, errs)
		assert.Len(t, events, 3)

		// Verify event structure
		for _, e := range events {
			assert.Equal(t, testMeter.Namespace, e.Namespace)
			assert.Equal(t, testMeter.EventType, e.Type)
			assert.NotEmpty(t, e.ID)
		}
	})

	t.Run("should stop operation when caller breaks early", func(t *testing.T) {
		mockMeterService := NewMockMeterService()
		mockMeterService.AddMeter(testMeter)

		mockStreaming := testutils.NewMockStreamingConnector(t)
		// Add many events
		for i := 0; i < 100; i++ {
			mockStreaming.AddSimpleEvent("test-meter", float64(i+1), now.Add(-time.Duration(i)*time.Minute))
		}

		svc, err := New(Config{
			EventSourceGroup:   "test-source",
			StreamingConnector: mockStreaming,
			MeterService:       mockMeterService,
		})
		require.NoError(t, err)

		config := meterexport.DataExportConfig{
			ExportWindowSize: meter.WindowSizeMinute,
			MeterID: models.NamespacedID{
				Namespace: "test-ns",
				ID:        "meter-1",
			},
			Period: timeutil.StartBoundedPeriod{
				From: now.Add(-100 * time.Minute),
				To:   lo.ToPtr(now),
			},
		}

		seq, err := svc.ExportSyntheticMeterDataIter(context.Background(), config)
		require.NoError(t, err)

		// Only consume first 3 events then break
		count := 0
		for event, err := range seq {
			if err != nil {
				continue
			}
			count++
			_ = event
			if count >= 3 {
				break // This should trigger context cancellation
			}
		}

		assert.Equal(t, 3, count)
		// The operation should have been canceled - we can't easily verify this
		// but the test completing without hanging proves the cancellation worked
	})

	t.Run("should return error for invalid config", func(t *testing.T) {
		mockMeterService := NewMockMeterService()
		mockStreaming := testutils.NewMockStreamingConnector(t)

		svc, err := New(Config{
			EventSourceGroup:   "test-source",
			StreamingConnector: mockStreaming,
			MeterService:       mockMeterService,
		})
		require.NoError(t, err)

		config := meterexport.DataExportConfig{
			ExportWindowSize: meter.WindowSizeMinute,
			MeterID: models.NamespacedID{
				Namespace: "test-ns",
				ID:        "", // Missing ID
			},
			Period: timeutil.StartBoundedPeriod{
				From: now.Add(-10 * time.Minute),
				To:   lo.ToPtr(now),
			},
		}

		_, err = svc.ExportSyntheticMeterDataIter(context.Background(), config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "meter id is required")
	})

	t.Run("should return error for unsupported aggregation", func(t *testing.T) {
		avgMeter := meter.Meter{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
				ManagedModel: models.ManagedModel{
					CreatedAt: now.Add(-time.Hour),
					UpdatedAt: now.Add(-time.Hour),
				},
				ID:   "meter-avg",
				Name: "Avg Meter",
			},
			Key:           "avg-meter",
			Aggregation:   meter.MeterAggregationAvg,
			EventType:     "avg-event",
			ValueProperty: lo.ToPtr("$.value"),
		}

		mockMeterService := NewMockMeterService()
		mockMeterService.AddMeter(avgMeter)

		mockStreaming := testutils.NewMockStreamingConnector(t)

		svc, err := New(Config{
			EventSourceGroup:   "test-source",
			StreamingConnector: mockStreaming,
			MeterService:       mockMeterService,
		})
		require.NoError(t, err)

		config := meterexport.DataExportConfig{
			ExportWindowSize: meter.WindowSizeMinute,
			MeterID: models.NamespacedID{
				Namespace: "test-ns",
				ID:        "meter-avg",
			},
			Period: timeutil.StartBoundedPeriod{
				From: now.Add(-10 * time.Minute),
				To:   lo.ToPtr(now),
			},
		}

		_, err = svc.ExportSyntheticMeterDataIter(context.Background(), config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported meter aggregation")
	})
}
