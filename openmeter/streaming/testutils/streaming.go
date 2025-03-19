package testutils

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

var _ streaming.Connector = &MockStreamingConnector{}

func NewMockStreamingConnector(t testing.TB) *MockStreamingConnector {
	t.Helper()
	out := &MockStreamingConnector{}
	out.Reset()
	return out
}

type SimpleEvent struct {
	MeterSlug string
	Value     float64
	Time      time.Time
}

type MockStreamingConnector struct {
	rows   map[string][]meter.MeterQueryRow
	events map[string][]SimpleEvent
}

func (m *MockStreamingConnector) Reset() {
	m.rows = map[string][]meter.MeterQueryRow{}
	m.events = map[string][]SimpleEvent{}
}

func (m *MockStreamingConnector) AddSimpleEvent(meterSlug string, value float64, at time.Time) {
	m.events[meterSlug] = append(m.events[meterSlug], SimpleEvent{
		MeterSlug: meterSlug,
		Value:     value,
		Time:      at,
	})
}

func (m *MockStreamingConnector) SetSimpleEvents(meterSlug string, fn func(events []SimpleEvent) []SimpleEvent) {
	if _, ok := m.events[meterSlug]; !ok {
		m.events[meterSlug] = []SimpleEvent{}
	}
	m.events[meterSlug] = fn(m.events[meterSlug])
}

func (m *MockStreamingConnector) AddRow(meterSlug string, row meter.MeterQueryRow) {
	m.rows[meterSlug] = append(m.rows[meterSlug], row)
}

func (c *MockStreamingConnector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *MockStreamingConnector) DeleteNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (m *MockStreamingConnector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	return []streaming.CountEventRow{}, nil
}

func (m *MockStreamingConnector) ListEvents(ctx context.Context, namespace string, params meterevent.ListEventsParams) ([]streaming.RawEvent, error) {
	return []streaming.RawEvent{}, nil
}

func (m *MockStreamingConnector) CreateMeter(ctx context.Context, namespace string, meter meter.Meter) error {
	return nil
}

func (m *MockStreamingConnector) UpdateMeter(ctx context.Context, namespace string, meter meter.Meter) error {
	return nil
}

func (m *MockStreamingConnector) DeleteMeter(ctx context.Context, namespace string, meter meter.Meter) error {
	return nil
}

// Returns the result query set for the given params. If the query set is not found,
// it will try to approximate the result by aggregating the simple events
func (m *MockStreamingConnector) QueryMeter(ctx context.Context, namespace string, mm meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	rows := []meter.MeterQueryRow{}
	_, rowOk := m.rows[mm.Key]

	if rowOk {
		for _, row := range m.rows[mm.Key] {
			if row.WindowStart.Equal(*params.From) && row.WindowEnd.Equal(*params.To) {
				rows = append(rows, row)
			}
		}
	} else {
		row, err := m.aggregateEvents(mm.Key, params)
		if err != nil {
			return rows, err
		}
		rows = append(rows, row...)
	}

	return rows, nil
}

func (m *MockStreamingConnector) BatchInsert(ctx context.Context, events []streaming.RawEvent) error {
	return nil
}

func (m *MockStreamingConnector) ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error) {
	return strings.HasPrefix(jsonPath, "$."), nil
}

// We approximate the actual logic by a simple filter + aggregation for most cases
func (m *MockStreamingConnector) aggregateEvents(meterSlug string, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	events, ok := m.events[meterSlug]
	if !ok {
		return []meter.MeterQueryRow{}, meter.NewMeterNotFoundError(meterSlug)
	}

	if params.From == nil || params.To == nil {
		return nil, fmt.Errorf("streaming mock connector does not support filtering without from and to")
	}

	from := *params.From
	to := *params.To

	rows := make([]meter.MeterQueryRow, 0)

	if params.WindowSize != nil && params.WindowTimeZone != nil {
		// TODO: windowtimezone will be ignored

		windowingStart := from.Truncate(params.WindowSize.Duration()) // The first truncated time that from query falls into
		windowingEnd := to.Truncate(params.WindowSize.Duration())     // The last truncated time that to query falls into
		if !to.Equal(windowingEnd) {
			windowingEnd = windowingEnd.Add(params.WindowSize.Duration())
		}

		numOfWindows := int(windowingEnd.Sub(windowingStart).Seconds()) / int(params.WindowSize.Duration().Seconds())

		if numOfWindows == 0 {
			return nil, fmt.Errorf("couldnt calculate windows")
		}

		for i := 0; i < numOfWindows; i++ {
			rows = append(rows, meter.MeterQueryRow{
				Value:       0,
				WindowStart: windowingStart.Add(params.WindowSize.Duration() * time.Duration(i)),
				WindowEnd:   windowingStart.Add(params.WindowSize.Duration() * time.Duration(i+1)),
				GroupBy:     map[string]*string{},
			})
		}
	} else {
		rows = append(rows, meter.MeterQueryRow{
			Value:       0,
			WindowStart: from,
			WindowEnd:   to,
			GroupBy:     map[string]*string{},
		})
	}

	for i := range rows {
		row := &rows[i]
		var value float64
		for _, event := range events {
			eventWindowStart := event.Time.Truncate(time.Minute)
			// windowend is exclusive when doing this rounding
			eventWindowEnd := eventWindowStart.Add(time.Minute)

			if (eventWindowStart.After(row.WindowStart) || eventWindowStart.Equal(row.WindowStart)) &&
				(eventWindowEnd.Before(row.WindowEnd) || eventWindowEnd.Equal(row.WindowEnd)) {
				// TODO: Add support for more aggregation types
				value += event.Value
			}
		}
		rows[i].Value = value
	}

	// Clickhouse doesn't return tumpled result rows if there are no rows (events) in the tumpled period
	// To simulate this for the SUM behavior, we simply filter out rows that have 0 value
	rows = lo.Filter(rows, func(row meter.MeterQueryRow, _ int) bool {
		return row.Value != 0
	})

	return rows, nil
}

func (m *MockStreamingConnector) ListMeterSubjects(ctx context.Context, namespace string, meter meter.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
	return []string{}, nil
}
