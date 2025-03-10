package clickhouse

import (
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// mergeMeterQueryRows merges cached rows with fresh rows
func mergeMeterQueryRows(meter meter.Meter, params streaming.QueryParams, cachedRows []meterpkg.MeterQueryRow, freshRows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
	if len(cachedRows) == 0 {
		return freshRows
	}

	// If window size is set there is no aggregation between cached and fresh rows
	// So we just concatenate the rows
	if params.WindowSize != nil {
		values := append(freshRows, cachedRows...)

		sort.Slice(values, func(i, j int) bool {
			return values[i].WindowStart.Before(values[j].WindowStart)
		})

		return values
	}

	// Create a map to store aggregated values by group and window
	grouppedRows := make(map[string][]meterpkg.MeterQueryRow)

	// Process all rows and aggregate them together
	for _, row := range append(freshRows, cachedRows...) {
		// Create a key based on groupBy values
		key := getMeterQueryRowKey(row, params)

		// Add the row to the group
		if _, exists := grouppedRows[key]; !exists {
			grouppedRows[key] = []meterpkg.MeterQueryRow{row}
		} else {
			grouppedRows[key] = append(grouppedRows[key], row)
		}
	}

	// Aggregate the rows
	var results []meterpkg.MeterQueryRow

	for _, rows := range grouppedRows {
		aggregated := aggregateMeterQueryRows(meter, rows)

		results = append(results, aggregated)
	}

	return results
}

// getMeterQueryRowKey creates a unique key for grouping rows based on subject and group by fields
// We don't include window start and end because we assume query window size is not set
func getMeterQueryRowKey(row meterpkg.MeterQueryRow, params streaming.QueryParams) string {
	key := ""

	// Add subject to the key if it exists
	if row.Subject != nil {
		key += fmt.Sprintf("subject=%s;", *row.Subject)
	}

	// Add all groupBy values to the key
	groupByKeys := params.GroupBy

	slices.Sort(groupByKeys)

	for _, groupByKey := range groupByKeys {
		val := "nil"
		if g, exists := row.GroupBy[groupByKey]; exists && g != nil {
			val = *g
		}

		key += fmt.Sprintf("group=%s=%s;", groupByKey, val)
	}

	return key
}

// aggregateMeterQueryRows combines rows into a single row
func aggregateMeterQueryRows(meter meter.Meter, rows []meterpkg.MeterQueryRow) meterpkg.MeterQueryRow {
	aggregated := meterpkg.MeterQueryRow{
		WindowStart: lo.MinBy(rows, func(a meterpkg.MeterQueryRow, b meterpkg.MeterQueryRow) bool {
			return a.WindowStart.Before(b.WindowStart)
		}).WindowStart,
		WindowEnd: lo.MaxBy(rows, func(a meterpkg.MeterQueryRow, b meterpkg.MeterQueryRow) bool {
			return a.WindowEnd.After(b.WindowEnd)
		}).WindowEnd,
		Subject: rows[0].Subject,
		GroupBy: make(map[string]*string),
	}

	for _, row := range rows {
		for k, v := range row.GroupBy {
			aggregated.GroupBy[k] = v
		}
	}

	if meter.Aggregation == meterpkg.MeterAggregationSum || meter.Aggregation == meterpkg.MeterAggregationCount {
		var sum float64
		for _, row := range rows {
			sum += row.Value
		}

		aggregated.Value = sum
	} else if meter.Aggregation == meterpkg.MeterAggregationMin {
		min := rows[0].Value

		for _, row := range rows {
			min = math.Min(min, row.Value)
		}

		aggregated.Value = min
	} else if meter.Aggregation == meterpkg.MeterAggregationMax {
		max := rows[0].Value

		for _, row := range rows {
			max = math.Max(max, row.Value)
		}

		aggregated.Value = max
	}

	return aggregated
}
