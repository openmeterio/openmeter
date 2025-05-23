package clickhouse

import (
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// mergeMeterQueryRows merges cached rows with fresh rows
func mergeMeterQueryRows(meterDef meterpkg.Meter, queryParams streaming.QueryParams, cachedRows []meterpkg.MeterQueryRow, freshRows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
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

	return aggregatedResults
}

// createGroupKeyFromRowWithQueryParams creates a unique key for grouping rows based on subject and group by fields
// We don't include window start and end because we assume query window size is not set (when it is set, we don't merge cached and fresh rows)
func createGroupKeyFromRowWithQueryParams(row meterpkg.MeterQueryRow, queryParams streaming.QueryParams) string {
	return createGroupKeyFromRow(row, queryParams.GroupBy)
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
	return dedupeQueryRowsWithLogger(rows, groupByFields, nil)
}

// dedupeQueryRowsWithLogger deduplicates rows based on group key with optional logging
func dedupeQueryRowsWithLogger(rows []meterpkg.MeterQueryRow, groupByFields []string, logger *slog.Logger) ([]meterpkg.MeterQueryRow, error) {
	deduplicatedValues := []meterpkg.MeterQueryRow{}
	seen := map[string]meterpkg.MeterQueryRow{}

	for _, row := range rows {
		key := createRowKey(row, groupByFields)

		if existingRow, exists := seen[key]; !exists {
			// First time seeing this key
			deduplicatedValues = append(deduplicatedValues, row)
			seen[key] = row
		} else {
			// Duplicate found - need to decide which value to keep
			if row.Value != existingRow.Value {
				// Choose the best value using smart deduplication logic
				bestRow := chooseBestDuplicateRow(existingRow, row, key, logger)

				// Update the stored row if we chose the new one
				if bestRow.Value == row.Value {
					// Replace the existing row in the deduplicatedValues slice
					for i := range deduplicatedValues {
						if createRowKey(deduplicatedValues[i], groupByFields) == key {
							deduplicatedValues[i] = bestRow
							break
						}
					}
					seen[key] = bestRow
				}
				// If we kept the existing row, no changes needed
			}
			// If values are identical, nothing to do
		}
	}

	return deduplicatedValues, nil
}

// chooseBestDuplicateRow selects the best row when duplicates are found
// Priority: non-zero > zero, higher precision > lower precision
func chooseBestDuplicateRow(existing, new meterpkg.MeterQueryRow, key string, logger *slog.Logger) meterpkg.MeterQueryRow {
	// Log the duplicate for debugging
	if logger != nil {
		logger.Warn("duplicate cache row found, choosing best value",
			slog.String("key", key),
			slog.Float64("existing_value", existing.Value),
			slog.Float64("new_value", new.Value))
	}

	// Rule 1: Non-zero values are preferred over zero values
	// This handles the case where some queries return no data (0) while others find actual data
	if existing.Value == 0 && new.Value != 0 {
		if logger != nil {
			logger.Debug("choosing non-zero value over zero",
				slog.Float64("chosen", new.Value),
				slog.Float64("rejected", existing.Value))
		}
		return new // Choose non-zero value
	}
	if new.Value == 0 && existing.Value != 0 {
		if logger != nil {
			logger.Debug("keeping non-zero value over zero",
				slog.Float64("kept", existing.Value),
				slog.Float64("rejected", new.Value))
		}
		return existing // Keep non-zero value
	}

	// Rule 2: If both are non-zero, choose the one with higher absolute value
	// This assumes that floating point precision issues usually result in slightly smaller values
	if existing.Value != 0 && new.Value != 0 {
		if math.Abs(new.Value) > math.Abs(existing.Value) {
			if logger != nil {
				logger.Debug("choosing higher absolute value",
					slog.Float64("chosen", new.Value),
					slog.Float64("rejected", existing.Value))
			}
			return new
		}
		if logger != nil {
			logger.Debug("keeping higher absolute value",
				slog.Float64("kept", existing.Value),
				slog.Float64("rejected", new.Value))
		}
		return existing
	}

	// Rule 3: If both are zero, keep the existing one
	if logger != nil {
		logger.Debug("both values are zero, keeping existing",
			slog.Float64("value", existing.Value))
	}
	return existing
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
