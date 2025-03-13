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
func mergeMeterQueryRows(meterDef meter.Meter, queryParams streaming.QueryParams, cachedRows []meterpkg.MeterQueryRow, freshRows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
	if len(cachedRows) == 0 {
		return freshRows
	}

	// If window size is set there is no aggregation between cached and fresh rows
	// So we just concatenate the rows
	if queryParams.WindowSize != nil {
		combinedRows := append(freshRows, cachedRows...)

		sort.Slice(combinedRows, func(i, j int) bool {
			return combinedRows[i].WindowStart.Before(combinedRows[j].WindowStart)
		})

		return combinedRows
	}

	// Create a map to store aggregated values by group and window
	groupedRows := make(map[string][]meterpkg.MeterQueryRow)

	// Process all rows and aggregate them together
	for _, row := range append(freshRows, cachedRows...) {
		// Create a key based on groupBy values
		groupKey := createGroupKeyFromRow(row, queryParams)

		// Add the row to the group
		if _, exists := groupedRows[groupKey]; !exists {
			groupedRows[groupKey] = []meterpkg.MeterQueryRow{row}
		} else {
			groupedRows[groupKey] = append(groupedRows[groupKey], row)
		}
	}

	// Aggregate the rows
	var aggregatedResults []meterpkg.MeterQueryRow

	for _, rowGroup := range groupedRows {
		aggregatedRow := aggregateRowsByAggregationType(meterDef.Aggregation, rowGroup)
		aggregatedResults = append(aggregatedResults, aggregatedRow)
	}

	return aggregatedResults
}

// createGroupKeyFromRow creates a unique key for grouping rows based on subject and group by fields
// We don't include window start and end because we assume query window size is not set
func createGroupKeyFromRow(row meterpkg.MeterQueryRow, queryParams streaming.QueryParams) string {
	groupKey := ""

	// Add subject to the key if it exists
	if row.Subject != nil {
		groupKey += fmt.Sprintf("subject=%s;", *row.Subject)
	}

	// Add all groupBy values to the key
	groupByFields := queryParams.GroupBy

	slices.Sort(groupByFields)

	for _, groupByField := range groupByFields {
		valueStr := "nil"
		if groupValue, exists := row.GroupBy[groupByField]; exists && groupValue != nil {
			valueStr = *groupValue
		}

		groupKey += fmt.Sprintf("group=%s=%s;", groupByField, valueStr)
	}

	return groupKey
}

// aggregateRowsByAggregationType combines rows into a single row based on the meter aggregation type
func aggregateRowsByAggregationType(aggregation meterpkg.MeterAggregation, rows []meterpkg.MeterQueryRow) meterpkg.MeterQueryRow {
	// Find earliest window start and latest window end
	earliestStart := lo.MinBy(rows, func(a meterpkg.MeterQueryRow, b meterpkg.MeterQueryRow) bool {
		return a.WindowStart.Before(b.WindowStart)
	}).WindowStart

	latestEnd := lo.MaxBy(rows, func(a meterpkg.MeterQueryRow, b meterpkg.MeterQueryRow) bool {
		return a.WindowEnd.After(b.WindowEnd)
	}).WindowEnd

	// Initialize aggregated row
	aggregatedRow := meterpkg.MeterQueryRow{
		WindowStart: earliestStart,
		WindowEnd:   latestEnd,
		Subject:     rows[0].Subject,
		GroupBy:     make(map[string]*string),
	}

	// Preserve group by values
	for _, row := range rows {
		for key, value := range row.GroupBy {
			aggregatedRow.GroupBy[key] = value
		}
	}

	// Apply appropriate aggregation based on meter type
	if aggregation == meterpkg.MeterAggregationSum || aggregation == meterpkg.MeterAggregationCount {
		var sum float64
		for _, row := range rows {
			sum += row.Value
		}

		aggregatedRow.Value = sum
	} else if aggregation == meterpkg.MeterAggregationMin {
		minValue := rows[0].Value

		for _, row := range rows {
			minValue = math.Min(minValue, row.Value)
		}

		aggregatedRow.Value = minValue
	} else if aggregation == meterpkg.MeterAggregationMax {
		maxValue := rows[0].Value

		for _, row := range rows {
			maxValue = math.Max(maxValue, row.Value)
		}

		aggregatedRow.Value = maxValue
	}

	return aggregatedRow
}
