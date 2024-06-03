package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type MockStreamingConnectorParams struct {
	DefaultHighwatermark time.Time
}

func NewMockStreamingConnector(t testing.TB, params MockStreamingConnectorParams) *MockStreamingConnector {
	t.Helper()
	return &MockStreamingConnector{
		rows:   map[string][]models.MeterQueryRow{},
		events: map[string][]SimpleEvent{},
		params: params,
	}
}

type SimpleEvent struct {
	MeterSlug string
	Value     float64
	Time      time.Time
}

type MockStreamingConnector struct {
	rows   map[string][]models.MeterQueryRow
	events map[string][]SimpleEvent
	params MockStreamingConnectorParams
}

func (m *MockStreamingConnector) AddResponse(meterSlug string, value float64, at time.Time) {
	m.events[meterSlug] = append(m.events[meterSlug], SimpleEvent{
		MeterSlug: meterSlug,
		Value:     value,
		Time:      at,
	})
}

func (m *MockStreamingConnector) SetResponses(meterSlug string, fn func(events []SimpleEvent) []SimpleEvent) {
	if _, ok := m.events[meterSlug]; !ok {
		m.events[meterSlug] = []SimpleEvent{}
	}
	m.events[meterSlug] = fn(m.events[meterSlug])
}

func (m *MockStreamingConnector) AddRow(meterSlug string, row models.MeterQueryRow) {
	m.rows[meterSlug] = append(m.rows[meterSlug], row)
}

func (m *MockStreamingConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	return []api.IngestedEvent{}, nil
}

func (m *MockStreamingConnector) CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error {
	return nil
}

func (m *MockStreamingConnector) DeleteMeter(ctx context.Context, namespace string, meterSlug string) error {
	return nil
}

// Returns the result query set for the given params. If the query set is not found,
// it will try to approximate the result by aggregating the simple events
func (m *MockStreamingConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]models.MeterQueryRow, error) {
	rows := []models.MeterQueryRow{}
	_, rowOk := m.rows[meterSlug]

	if rowOk {
		for _, row := range m.rows[meterSlug] {
			if row.WindowStart.Equal(*params.From) && row.WindowEnd.Equal(*params.To) {
				rows = append(rows, row)
			}
		}
	} else {
		row, err := m.aggregateEvents(meterSlug, params)
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// We approximate the actual logic by a simple filter + aggregation for most cases
func (m *MockStreamingConnector) aggregateEvents(meterSlug string, params *streaming.QueryParams) (models.MeterQueryRow, error) {
	events, ok := m.events[meterSlug]
	from := defaultx.WithDefault(params.From, m.params.DefaultHighwatermark)
	to := defaultx.WithDefault(params.To, time.Now())
	if !ok {
		return models.MeterQueryRow{}, &models.MeterNotFoundError{MeterSlug: meterSlug}
	}

	var value float64
	for _, row := range events {
		eventWindowStart := row.Time.Truncate(time.Minute)
		// windowend is exclusive when doing this rounding
		eventWindowEnd := eventWindowStart.Add(time.Minute)

		if (eventWindowStart.After(from) || eventWindowStart.Equal(from)) &&
			(eventWindowEnd.Before(to) || eventWindowEnd.Equal(to)) {
			// Add support for more aggregation types
			value += row.Value
		}
	}

	return models.MeterQueryRow{
		Value:       value,
		WindowStart: *params.From,
		WindowEnd:   *params.To,
		GroupBy:     map[string]*string{},
	}, nil
}

func (m *MockStreamingConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	return []string{}, nil
}
