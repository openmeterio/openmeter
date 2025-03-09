package raw_events

import (
	"context"
	"fmt"
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
	// We only cache queries where cachable is set to true
	if !p.Cachable {
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
		newQueryMeter.From = nil
		newQueryMeter.FromExclusive = cachedQueryMeter.To

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

		// We query from the end of the last cached window exclusive
		queryMeterCached.From = nil
		queryMeterCached.FromExclusive = &lastCachedWindow

		// If we've covered the entire range with cached data, return early
		if lastCachedWindow.Equal(*queryMeterCached.To) {
			c.config.Logger.Debug("no new rows to query for cache period, returning cached data", "count", len(values))

			return createReaminingQueryMeter(queryMeterCached), values, nil
		}
	}

	// Step 2: Query new rows for the uncached time period
	newRows, err := c.queryMeter(ctx, queryMeterCached)
	if err != nil {
		return originalQueryMeter, values, fmt.Errorf("query new meter rows: %w", err)
	}

	values = append(values, newRows...)

	// Step 3: Cache the new results
	if len(newRows) > 0 {
		if err := c.insertRowsToCache(ctx, hash, queryMeterCached, newRows); err != nil {
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

// lookupCachedMeterRows queries the meter_query_hash table for cached results
func (c *Connector) lookupCachedMeterRows(ctx context.Context, hash string, hp queryMeter) ([]meterpkg.MeterQueryRow, error) {
	var cachedValues []meterpkg.MeterQueryRow

	hashQuery := getMeterQueryRowsFromCache{
		Database:  hp.Database,
		TableName: meterQueryRowCacheTable,
		Hash:      hash,
		Namespace: hp.Namespace,
		From:      hp.From,
		To:        hp.To,
	}

	sql, args := hashQuery.toSQL()
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query meter query hash: %w", err)
	}

	defer rows.Close()
	cachedValues, err = hashQuery.scanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan meter query hash rows: %w", err)
	}

	return cachedValues, nil
}

// insertRowsToCache stores new meter query results in the meter_query_hash table
func (c *Connector) insertRowsToCache(ctx context.Context, hash string, hp queryMeter, newValues []meterpkg.MeterQueryRow) error {
	insertQuery := insertMeterQueryRowsToCache{
		Database:  hp.Database,
		TableName: meterQueryRowCacheTable,
		Hash:      hash,
		Namespace: hp.Namespace,
		QueryRows: newValues,
	}

	sql, args := insertQuery.toSQL()
	if err := c.config.ClickHouse.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert meter query hash: %w", err)
	}

	return nil
}

// createMeterQueryCacheTable creates the meter_query_hash table if it doesn't exist
func (c *Connector) createMeterQueryCacheTable(ctx context.Context) error {
	table := createMeterQueryRowsCacheTable{
		Database:  c.config.Database,
		TableName: meterQueryRowCacheTable,
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create meter_query_hash table: %w", err)
	}

	return nil
}
