package summeterv1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/PaesslerAG/jsonpath"
	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	TableEngineName = "meter-sum.v1" // TODO: rename to numeric meter
)

type EngineState struct {
	StreamDataAfterStoredAt time.Time `json:"streamDataAfterStoredAt"`

	Backfill struct {
		MinStoredAt  time.Time     `json:"minStoredAt"`
		ImportChunks []ImportChunk `json:"importChunks"`
	} `json:"backfill"`

	Ready bool `json:"ready"`
}

type ImportChunk struct {
	timeutil.ClosedPeriod
}

type Engine struct {
	logger *slog.Logger

	database     string
	clickhouse   clickhouse.Conn
	meterService meter.ManageService
}

func (e *Engine) IsOperational(m meter.Meter) bool {
	if m.TableEngine == nil {
		return false
	}

	if m.TableEngine.Engine != TableEngineName {
		return false
	}

	if m.Aggregation != meter.MeterAggregationSum {
		return false
	}

	// Let's parse the state
	var state EngineState
	if err := json.Unmarshal([]byte(m.TableEngine.State), &state); err != nil {
		e.logger.Error("failed to unmarshal table engine state", "error", err, "state", m.TableEngine.State)
		return false
	}

	// TODO: validate state

	return true
}

func (e *Engine) GetRecordForMeter(ctx context.Context, meter meter.Meter, event serializer.CloudEventsKafkaPayload, storedAt time.Time) (*Record, error) {
	// TODO: move this to the cache
	var state EngineState
	if err := json.Unmarshal([]byte(meter.TableEngine.State), &state); err != nil {
		e.logger.Error("failed to unmarshal table engine state", "error", err, "state", meter.TableEngine.State)
		return nil, err
	}

	if storedAt.Truncate(time.Second).Before(state.StreamDataAfterStoredAt.Truncate(time.Second)) {
		return nil, nil
	}

	// Parse CloudEvent data JSON
	var data interface{}
	if event.Data != "" {
		if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
			e.logger.Error("failed to unmarshal cloudevent data", "error", err)
			return nil, err
		}
	}

	// Extract numeric value using JSONPath from meter.ValueProperty
	// The behavior should match clickhouse meter query:
	// - Use the JSONPath as-is (starts with $)
	// - Convert to float64
	// - Treat missing path as nil (skip)
	// - Ignore NaN/Inf (skip)
	if meter.ValueProperty == nil || *meter.ValueProperty == "" {
		return nil, fmt.Errorf("value property is required for numeric meter")
	}

	rawValue, err := jsonpath.Get(*meter.ValueProperty, data)
	if err != nil {
		// Path not found or invalid → skip this record
		return nil, nil
	}

	// TODO: if value is not parsable we need to add a null record or are we fine just skipping it?

	var floatVal float64
	switch v := rawValue.(type) {
	case float64:
		floatVal = v
	case json.Number:
		if fv, err := v.Float64(); err == nil {
			floatVal = fv
		} else {
			return nil, nil
		}
	case string:
		// Attempt to parse numeric values represented as string
		var num json.Number = json.Number(v)
		if fv, err := num.Float64(); err == nil {
			floatVal = fv
		} else {
			// Not a numeric string → skip to mimic toFloat64OrNull(JSON_VALUE(...))
			return nil, nil
		}
	default:
		// Non-numeric types are skipped
		return nil, nil
	}

	// Drop NaN/Inf to mimic ifNotFinite(..., null)
	if math.IsNaN(floatVal) || math.IsInf(floatVal, 0) {
		return nil, nil
	}

	// Build group-by filters: key -> extracted string value from JSONPath
	groupByFilters := make(map[string]string, len(meter.GroupBy))
	for key, path := range meter.GroupBy {
		val, err := jsonpath.Get(path, data)
		if err != nil {
			// Use empty string for missing paths (JSON_VALUE default)
			groupByFilters[key] = ""
			continue
		}
		switch vv := val.(type) {
		case string:
			groupByFilters[key] = vv
		case float64, bool, json.Number:
			// Convert scalar to its string representation
			groupByFilters[key] = fmt.Sprint(vv)
		case nil, map[string]interface{}, []interface{}:
			// Complex types and nil → empty string (JSON_VALUE default)
			groupByFilters[key] = ""
		default:
			// Fallback to string representation
			groupByFilters[key] = fmt.Sprint(vv)
		}
	}

	rec := &Record{
		Namespace:      meter.Namespace,
		MeterID:        meter.ID,
		Subject:        event.Subject,
		Time:           time.Unix(event.Time, 0),
		Value:          alpacadecimal.NewFromFloat(floatVal),
		GroupByFilters: groupByFilters,
		StoredAt:       storedAt,
		StoreRowID:     ulid.Make().String(),
	}

	return rec, nil
}

type Counter int

func (c *Counter) Decrement() {
	*c--
}

func (c *Counter) HasMoreSteps() bool {
	return *c > 0
}

func (e *Engine) Maintain(ctx context.Context, meter meter.Meter) error {
	// Extract state
	var state EngineState
	if err := json.Unmarshal([]byte(meter.TableEngine.State), &state); err != nil {
		e.logger.Error("failed to unmarshal table engine state", "error", err, "state", meter.TableEngine.State)
		return err
	}

	// Check if the meter is ready
	if state.Ready {
		return nil
	}

	// Initialize stream start if missing
	if state.StreamDataAfterStoredAt.IsZero() {
		state.StreamDataAfterStoredAt = clock.Now().Add(defaultStreamStartOffset).UTC().Truncate(time.Second)
	}

	// Check if the min stored
	if state.Backfill.MinStoredAt.IsZero() {
		// determine the min stored at by querying om_events table with namespace and event type
		const eventsTableName = "om_events"
		minStoredAt, err := e.MinEventsStoredAt(ctx, eventsTableName, meter.Namespace, meter.EventType)
		if err != nil {
			e.logger.Error("failed to query min stored_at from events", "error", err, "namespace", meter.Namespace, "eventType", meter.EventType)
			return err
		}

		var min time.Time
		if minStoredAt == nil {
			// If no data found, use min(stream starts - 24 hours, now - 24 hours)
			// Note: This ensures any incoming events are considered.
			streamStartsMinusDay := state.StreamDataAfterStoredAt.UTC().Add(-24 * time.Hour)
			nowMinusDay := time.Now().UTC().Add(-24 * time.Hour)
			if streamStartsMinusDay.Before(nowMinusDay) {
				min = streamStartsMinusDay
			} else {
				min = nowMinusDay
			}
			min = min.Truncate(time.Second)
		} else {
			min = minStoredAt.UTC().Truncate(time.Second)
		}
		state.Backfill.MinStoredAt = min

		// once calculated update the state with the new min stored at and generate the import chunks
		// by having one chunk per day between the min stored at and the stream data after stored at
		startDay := time.Date(min.Year(), min.Month(), min.Day(), 0, 0, 0, 0, time.UTC)
		endExclusive := state.StreamDataAfterStoredAt.UTC().Truncate(time.Second)
		state.Backfill.ImportChunks = generateDailyChunks(startDay, endExclusive)
	}

	// Check if the current time is after the stream data after stored at + 5min
	if clock.Now().After(state.StreamDataAfterStoredAt.Add(5 * time.Minute)) {
		// Start the legacy backfill process:
		// - take a chunk and insert the records into the meter table
		// - delete all records from the meter storage table for the chunk to make sure we don't have any duplicates
		// - use a select into query to query the value + the group by values from the om_events table
		// - see map function here: https://clickhouse.com/docs/sql-reference/functions/tuple-map-functions#map
		// If there are no more chunks, mark ready.

		// Process all chunks in this pass
		for len(state.Backfill.ImportChunks) > 0 {
			chunk := state.Backfill.ImportChunks[0]
			if err := e.InsertFromEvents(ctx, "om_events", meter, chunk.ClosedPeriod); err != nil {
				e.logger.Error("failed to insert records from events for chunk", "error", err, "from", chunk.From, "to", chunk.To)
				return err
			}
			state.Backfill.ImportChunks = state.Backfill.ImportChunks[1:]
		}
		// Mark ready
		state.Ready = true
	}

	// persist final state
	if err := e.persistState(ctx, &meter, state); err != nil {
		e.logger.Error("failed to persist meter table engine state", "error", err)
		return err
	}
	return nil
}

func generateDailyChunks(startInclusive time.Time, endExclusive time.Time) []ImportChunk {
	chunks := make([]ImportChunk, 0, 32)
	for cur := startInclusive; cur.Before(endExclusive); {
		next := cur.Add(24 * time.Hour)
		if next.After(endExclusive) {
			next = endExclusive
		}
		chunks = append(chunks, ImportChunk{
			ClosedPeriod: timeutil.ClosedPeriod{
				From: cur,
				To:   next,
			},
		})
		cur = next
	}
	return chunks
}

func (e *Engine) persistState(ctx context.Context, m *meter.Meter, st EngineState) error {
	b, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal engine state: %w", err)
	}
	m.TableEngine.State = string(b)
	if err := e.meterService.UpdateTableEngine(ctx, *m); err != nil {
		return fmt.Errorf("failed to update table engine state: %w", err)
	}
	return nil
}
