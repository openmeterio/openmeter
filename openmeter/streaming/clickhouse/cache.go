package raw_events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/samber/lo"
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
		c.config.Logger.Debug("cached rows found", "from", queryMeterCached.From, "to", queryMeterCached.To, "windowSize", queryMeterCached.WindowSize, "count", len(cachedValues))

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
			c.config.Logger.Error("failed to store new rows in cache", "error", err, "from", queryMeterCached.From, "to", queryMeterCached.To, "windowSize", queryMeterCached.WindowSize, "count", len(newRows))
		} else {
			c.config.Logger.Debug("new rows stored in cache", "from", queryMeterCached.From, "to", queryMeterCached.To, "windowSize", queryMeterCached.WindowSize, "count", len(newRows))
		}
	}

	// Result
	c.config.Logger.Debug("returning cached and new rows", "from", queryMeterCached.From, "to", queryMeterCached.To, "windowSize", queryMeterCached.WindowSize, "count", len(values))

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

	// Set window size based on time range if not provided
	// This is the window size that the cache will use.
	if cachedQueryMeter.WindowSize == nil {
		duration := cachedQueryMeter.To.Sub(*cachedQueryMeter.From)

		// For long time ranges, we use the day window size
		if duration > time.Hour*24 {
			cachedQueryMeter.WindowSize = lo.ToPtr(meter.WindowSizeDay)
		}
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
func mergeCachedRows(params streaming.QueryParams, cachedRows []meterpkg.MeterQueryRow, freshRows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
	if len(cachedRows) == 0 {
		return freshRows
	}

	values := freshRows

	if params.WindowSize == nil {
		for _, row := range cachedRows {
			values[0].Value += row.Value
		}
	} else {
		values = append(values, cachedRows...)
	}

	return values
}
