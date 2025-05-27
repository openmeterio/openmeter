package clickhouse

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// mergeMeterQueryRows merges cached rows with fresh rows
func mergeMeterQueryRows(meterDef meterpkg.Meter, queryParams streaming.QueryParams, rows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
	if len(rows) == 0 {
		return rows
	}

	// Create a map to store aggregated values by group and window
	groupedRows := make(map[string][]meterpkg.MeterQueryRow)

	// Process all rows and aggregate them together
	for _, row := range rows {
		// Create a key based on groupBy values
		groupKey := createGroupKeyFromRowWithQueryParams(row, queryParams)

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

	// Sort results by window start
	// This is required as go maps are not ordered and we loose the order at groupping
	sort.Slice(aggregatedResults, func(i, j int) bool {
		return aggregatedResults[i].WindowStart.Before(aggregatedResults[j].WindowStart)
	})

	return aggregatedResults
}

// createGroupKeyFromRowWithQueryParams creates a unique key for grouping rows based on subject and group by fields
// We don't include window start and end because we assume query window size is not set (when it is set, we don't merge cached and fresh rows)
func createGroupKeyFromRowWithQueryParams(row meterpkg.MeterQueryRow, queryParams streaming.QueryParams) string {
	groupKey := createGroupKeyFromRow(row, queryParams.GroupBy)

	// Window key is used to group rows by window size
	if queryParams.WindowSize != nil {
		return fmt.Sprintf("%s;%s", row.WindowStart.UTC().Format(time.RFC3339), row.WindowEnd.UTC().Format(time.RFC3339))
	}

	return groupKey
}

// createGroupKeyFromRow creates a unique key for grouping rows based on subject and group by fields
// We don't include window start and end because we assume query window size is not set (when it is set, we don't merge cached and fresh rows)
func createGroupKeyFromRow(row meterpkg.MeterQueryRow, groupByFields []string) string {
	groupByFieldsCopy := append([]string(nil), groupByFields...)
	groupKey := ""

	// Add subject to the key if it exists
	if row.Subject != nil {
		groupKey += fmt.Sprintf("subject=%s;", *row.Subject)
	}

	slices.Sort(groupByFieldsCopy)

	for _, groupByField := range groupByFieldsCopy {
		valueStr := "nil"
		if groupValue, exists := row.GroupBy[groupByField]; exists && groupValue != nil {
			valueStr = *groupValue
		}

		groupKey += fmt.Sprintf("group=%s=%s;", groupByField, valueStr)
	}

	return groupKey
}

// createRowKey creates a unique key for a row based on group key and window start and end
func createRowKey(row meterpkg.MeterQueryRow, groupByFields []string) string {
	groupKey := createGroupKeyFromRow(row, groupByFields)

	// `row.subject` is already included in the group key
	return fmt.Sprintf("%s-%s-%s", groupKey, row.WindowStart.UTC().Format(time.RFC3339), row.WindowEnd.UTC().Format(time.RFC3339))
}

// dedupeQueryRows deduplicates rows based on group key
func dedupeQueryRows(rows []meterpkg.MeterQueryRow, groupByFields []string) ([]meterpkg.MeterQueryRow, error) {
	deduplicatedValues := []meterpkg.MeterQueryRow{}
	seen := map[string]meterpkg.MeterQueryRow{}

	for _, row := range rows {
		key := createRowKey(row, groupByFields)

		if _, ok := seen[key]; !ok {
			deduplicatedValues = append(deduplicatedValues, row)
			seen[key] = row
		} else {
			if row.Value != seen[key].Value {
				return nil, fmt.Errorf("duplicate row found with different value: %s", key)
			}
		}
	}

	return deduplicatedValues, nil
}

// aggregateRowsByAggregationType combines rows into a single row based on the meter aggregation type
func aggregateRowsByAggregationType(aggregation meterpkg.MeterAggregation, rows []meterpkg.MeterQueryRow) meterpkg.MeterQueryRow {
	// Find earliest window start and latest window end
	windowStart := lo.MinBy(rows, func(a meterpkg.MeterQueryRow, b meterpkg.MeterQueryRow) bool {
		return a.WindowStart.Before(b.WindowStart)
	}).WindowStart

	windowEnd := lo.MaxBy(rows, func(a meterpkg.MeterQueryRow, b meterpkg.MeterQueryRow) bool {
		return a.WindowEnd.After(b.WindowEnd)
	}).WindowEnd

	// Initialize aggregated row
	aggregatedRow := meterpkg.MeterQueryRow{
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Subject:     rows[0].Subject,
		GroupBy:     rows[0].GroupBy,
	}

	// Apply appropriate aggregation based on meter type
	switch aggregation {
	case meterpkg.MeterAggregationSum, meterpkg.MeterAggregationCount:
		var sum float64
		for _, row := range rows {
			sum += row.Value
		}

		aggregatedRow.Value = sum
	case meterpkg.MeterAggregationMin:
		minValue := rows[0].Value

		for _, row := range rows {
			minValue = math.Min(minValue, row.Value)
		}

		aggregatedRow.Value = minValue
	case meterpkg.MeterAggregationMax:
		maxValue := rows[0].Value

		for _, row := range rows {
			maxValue = math.Max(maxValue, row.Value)
		}

		aggregatedRow.Value = maxValue
	}

	return aggregatedRow
}
