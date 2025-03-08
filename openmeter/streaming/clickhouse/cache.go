package raw_events

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

const (
	minCachableDuration = 3 * 24 * time.Hour
	minCacheableToAge   = 24 * time.Hour
)

// isQueryCachable returns true if the query params are cachable
func isQueryCachable(m meter.Meter, p streaming.QueryParams) bool {
	// We only cache client queries for now
	if p.ClientID == nil {
		return false
	}

	// We can only cache queries that have a from time
	if p.From == nil {
		return false
	}

	from := *p.From
	to := lo.FromPtrOr(p.To, time.Now().UTC())
	duration := to.Sub(from)

	// It must be at least 3 days of usage to be cachable
	if duration < minCachableDuration {
		return false
	}

	return m.Aggregation == meter.MeterAggregationSum || m.Aggregation == meter.MeterAggregationCount || m.Aggregation == meter.MeterAggregationMin || m.Aggregation == meter.MeterAggregationMax
}

// remainingQueryMeterFactory creates a new query meter that starts from the end of the cached query meter
// It is used to query the fresh rows for the remaining time period from the events table directly
func (c *Connector) remainingQueryMeterFactory(originalQueryMeter queryMeter) func(cachedQueryMeter queryMeter) queryMeter {
	newQueryMeter := originalQueryMeter

	return func(cachedQueryMeter queryMeter) queryMeter {
		newQueryMeter.From = cachedQueryMeter.To

		c.config.Logger.Debug("query fresh rows from events table", "from", newQueryMeter.From, "to", newQueryMeter.To)

		return newQueryMeter
	}
}

// queryMeterCached queries the meter view and manages the cache, as:
// 1. Look up cached rows
// 2. Query new rows for the uncached time period
// 3. Cache the new results
// It returns the cached rows and the updated query meter.
func (c *Connector) queryMeterCached(ctx context.Context, hash string, originalQueryMeter queryMeter) (queryMeter, []meterpkg.MeterQueryRow, error) {
	var values []meterpkg.MeterQueryRow

	createReaminingQueryMeter := c.remainingQueryMeterFactory(originalQueryMeter)

	// Calculate the period to query from the cache
	queryMeterCached, err := c.getQueryMeterForCachedPeriod(originalQueryMeter)
	if err != nil {
		return originalQueryMeter, values, err
	}

	// Step 1: Look up cached rows
	cachedValues, err := c.lookupCachedMeterRows(ctx, hash, queryMeterCached)
	if err != nil {
		return originalQueryMeter, values, fmt.Errorf("failed to lookup cached meter rows: %w", err)
	}

	// If we have cached values, add them to the results
	// Also, update the query range to query uncached periods
	if len(cachedValues) > 0 {
		c.config.Logger.Debug("cached rows found", "from", queryMeterCached.From, "to", queryMeterCached.To, "count", len(cachedValues))

		values = append(values, cachedValues...)

		// We use the last cached window as the start of the new query period
		lastCachedWindow := cachedValues[len(cachedValues)-1].WindowEnd
		queryMeterCached.From = &lastCachedWindow
	}

	// If we've covered the entire range with cached data, return early
	if queryMeterCached.From.Equal(*queryMeterCached.To) {
		c.config.Logger.Debug("no new rows to query for cache period, returning cached data", "count", len(values))

		return createReaminingQueryMeter(queryMeterCached), values, nil
	}

	// Step 2: Query new rows for the uncached time period
	newRows, err := c.queryNewMeterRows(ctx, queryMeterCached)
	if err != nil {
		return originalQueryMeter, values, fmt.Errorf("query new meter rows: %w", err)
	}

	values = append(values, newRows...)

	// Step 3: Cache the new results
	if len(newRows) > 0 {
		if err := c.cacheMeterRows(ctx, hash, queryMeterCached, newRows); err != nil {
			// Log the error but don't fail the query
			c.config.Logger.Error("failed to store new rows in cache", "error", err, "from", queryMeterCached.From, "to", queryMeterCached.To, "count", len(newRows))
		} else {
			c.config.Logger.Debug("new rows stored in cache", "from", queryMeterCached.From, "to", queryMeterCached.To, "count", len(newRows))
		}
	}

	// Result
	c.config.Logger.Debug("returning cached and new rows", "from", queryMeterCached.From, "to", queryMeterCached.To, "count", len(values))

	return createReaminingQueryMeter(queryMeterCached), values, nil
}

// adjustQueryTimeRange prepares the time range for a query meter operation
func (c *Connector) getQueryMeterForCachedPeriod(originalQueryMeter queryMeter) (queryMeter, error) {
	cachedQueryMeter := originalQueryMeter
	now := time.Now().UTC()

	if originalQueryMeter.From == nil {
		return cachedQueryMeter, fmt.Errorf("from is required for cached queries")
	}

	// Set the end time to now if not provided
	if cachedQueryMeter.To == nil {
		cachedQueryMeter.To = &now
	}

	// We do not cache data that is less than 24 hours old
	toFresness := now.Sub(*cachedQueryMeter.To)

	if toFresness < minCacheableToAge {
		delta := minCacheableToAge - toFresness

		cachedQueryMeter.To = lo.ToPtr(cachedQueryMeter.To.Add(-delta))
	}

	// We truncate to complete days to avoid partial days in the cache
	cachedQueryMeter.To = lo.ToPtr(cachedQueryMeter.To.Truncate(time.Hour * 24))

	// This is the window size that the cache will use if no window size is provided
	if cachedQueryMeter.WindowSize == nil {
		cachedQueryMeter.WindowSize = lo.ToPtr(meter.WindowSizeDay)
	}

	return cachedQueryMeter, nil
}

// queryNewMeterRows executes a query to retrieve new meter data that is not cached
func (c *Connector) queryNewMeterRows(ctx context.Context, hp queryMeter) ([]meterpkg.MeterQueryRow, error) {
	// Build the SQL query for uncached data
	sql, args, err := hp.toSQL()
	if err != nil {
		return nil, fmt.Errorf("build meter view query: %w", err)
	}

	// Query the meter view
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, meterpkg.NewMeterNotFoundError(hp.Meter.Key)
		}
		return nil, fmt.Errorf("execute meter view query: %w", err)
	}

	defer rows.Close()

	// Scan the rows
	newValues, err := hp.scanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan meter query row: %w", err)
	}

	return newValues, nil
}

// mergeCachedRows merges cached rows with fresh rows
func mergeCachedRows(meter meter.Meter, params streaming.QueryParams, cachedRows []meterpkg.MeterQueryRow, freshRows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
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
		key := getRowGroupKey(row, params)

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
		aggregated := aggregateRows(meter, rows)

		results = append(results, aggregated)
	}

	return results
}

// getRowGroupKey creates a unique key for grouping rows based on subject and group by fields
// We don't include window start and end because we assume query window size is not set
func getRowGroupKey(row meterpkg.MeterQueryRow, params streaming.QueryParams) string {
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

// aggregateRows combines rows into a single row
func aggregateRows(meter meter.Meter, rows []meterpkg.MeterQueryRow) meterpkg.MeterQueryRow {
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
		var min float64
		for _, row := range rows {
			min = math.Min(min, row.Value)
		}

		aggregated.Value = min
	} else if meter.Aggregation == meterpkg.MeterAggregationMax {
		var max float64
		for _, row := range rows {
			max = math.Max(max, row.Value)
		}

		aggregated.Value = max
	}

	return aggregated
}
