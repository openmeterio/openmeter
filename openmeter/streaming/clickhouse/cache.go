package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// canQueryBeCached returns true if the query params are cachable
func (c *Connector) canQueryBeCached(meterDef meterpkg.Meter, queryParams streaming.QueryParams) bool {
	// If caching is disabled, we don't cache anything
	if !c.config.QueryCacheEnabled {
		return false
	}

	// We only cache queries where cachable is set to true
	if !queryParams.Cachable {
		return false
	}

	// We can only cache queries that have a from time
	if queryParams.From == nil {
		return false
	}

	from := *queryParams.From
	to := lo.FromPtrOr(queryParams.To, time.Now().UTC())
	duration := to.Sub(from)

	// It must be at least 3 days of usage to be cachable
	if duration < c.config.QueryCacheMinimumCacheableQueryPeriod {
		return false
	}

	return meterDef.Aggregation == meterpkg.MeterAggregationSum || meterDef.Aggregation == meterpkg.MeterAggregationCount || meterDef.Aggregation == meterpkg.MeterAggregationMin || meterDef.Aggregation == meterpkg.MeterAggregationMax
}

// createRemainingQueryFactory returns a function that creates a new query meter starting from the end of the cached query meter
// It is used to query the fresh rows for the remaining time period from the events table directly
func (c *Connector) createRemainingQueryFactory(originalQueryMeter queryMeter) func(cachedQueryMeter queryMeter) queryMeter {
	newQueryMeter := originalQueryMeter

	return func(cachedQueryMeter queryMeter) queryMeter {
		newQueryMeter := newQueryMeter

		newQueryMeter.From = nil
		newQueryMeter.FromExclusive = cachedQueryMeter.To

		c.config.Logger.Debug("query fresh rows from events table", "from", newQueryMeter.From, "to", newQueryMeter.To)

		return newQueryMeter
	}
}

// executeQueryWithCaching queries the meter view and manages the cache, as:
// 1. Look up cached rows
// 2. Query new rows for the uncached time period
// 3. Cache the new results
// It returns the cached rows and the updated query meter.
func (c *Connector) executeQueryWithCaching(ctx context.Context, hash string, originalQueryMeter queryMeter) (queryMeter, []meterpkg.MeterQueryRow, error) {
	var values []meterpkg.MeterQueryRow

	createRemainingQuery := c.createRemainingQueryFactory(originalQueryMeter)

	// Calculate the period to query from the cache
	cacheableQueryMeter, err := c.prepareCacheableQueryPeriod(originalQueryMeter)
	if err != nil {
		return originalQueryMeter, values, err
	}

	// Step 1: Look up cached rows
	cachedValues, err := c.fetchCachedMeterRows(ctx, hash, cacheableQueryMeter)
	if err != nil {
		return originalQueryMeter, values, fmt.Errorf("failed to lookup cached meter rows: %w", err)
	}

	// If we have cached values, add them to the results
	// Also, update the query range to query uncached periods
	if len(cachedValues) > 0 {
		c.config.Logger.Debug("cached rows found", "from", cacheableQueryMeter.From, "to", cacheableQueryMeter.To, "count", len(cachedValues))

		values = append(values, cachedValues...)

		// We use the last cached window as the start of the new query period
		lastCachedWindow := cachedValues[len(cachedValues)-1].WindowEnd

		// We query from the end of the last cached window exclusive
		cacheableQueryMeter.From = nil
		cacheableQueryMeter.FromExclusive = &lastCachedWindow

		// If we've covered the entire range with cached data, return early
		if lastCachedWindow.Equal(*cacheableQueryMeter.To) {
			c.config.Logger.Debug("no new rows to query for cache period, returning cached data", "count", len(values))

			return createRemainingQuery(cacheableQueryMeter), values, nil
		}
	}

	// Step 2: Query new rows for the uncached time period
	newRows, err := c.queryMeter(ctx, cacheableQueryMeter)
	if err != nil {
		return originalQueryMeter, values, fmt.Errorf("query new meter rows: %w", err)
	}

	values = append(values, newRows...)

	// Step 3: Cache the new results
	if len(newRows) > 0 {
		if err := c.storeCachedMeterRows(ctx, hash, cacheableQueryMeter, newRows); err != nil {
			// Log the error but don't fail the query
			c.config.Logger.Error("failed to store new rows in cache", "error", err, "from", cacheableQueryMeter.From, "to", cacheableQueryMeter.To, "count", len(newRows))
		} else {
			c.config.Logger.Debug("new rows stored in cache", "from", cacheableQueryMeter.From, "to", cacheableQueryMeter.To, "count", len(newRows))
		}
	}

	// Result
	c.config.Logger.Debug("returning cached and new rows", "from", cacheableQueryMeter.From, "to", cacheableQueryMeter.To, "count", len(values))

	return createRemainingQuery(cacheableQueryMeter), values, nil
}

// prepareCacheableQueryPeriod prepares the time range for cacheable queries
func (c *Connector) prepareCacheableQueryPeriod(originalQueryMeter queryMeter) (queryMeter, error) {
	cacheableQueryMeter := originalQueryMeter
	now := time.Now().UTC()

	if originalQueryMeter.From == nil {
		return cacheableQueryMeter, fmt.Errorf("from is required for cached queries")
	}

	// Set the end time to now if not provided
	if cacheableQueryMeter.To == nil {
		cacheableQueryMeter.To = &now
	}

	// We do not cache data that is less than 24 hours old
	toFreshness := now.Sub(*cacheableQueryMeter.To)

	if toFreshness < c.config.QueryCacheMinimumCacheableUsageAge {
		delta := c.config.QueryCacheMinimumCacheableUsageAge - toFreshness

		cacheableQueryMeter.To = lo.ToPtr(cacheableQueryMeter.To.Add(-delta))
	}

	// We truncate to complete days to avoid partial days in the cache
	cacheableQueryMeter.To = lo.ToPtr(cacheableQueryMeter.To.Truncate(time.Hour * 24))

	// This is the window size that the cache will use if no window size is provided
	if cacheableQueryMeter.WindowSize == nil {
		cacheableQueryMeter.WindowSize = lo.ToPtr(meterpkg.WindowSizeDay)
	}

	return cacheableQueryMeter, nil
}

// fetchCachedMeterRows queries the meter_query_hash table for cached results
func (c *Connector) fetchCachedMeterRows(ctx context.Context, hash string, queryMeter queryMeter) ([]meterpkg.MeterQueryRow, error) {
	var cachedValues []meterpkg.MeterQueryRow

	cacheQuery := getMeterQueryRowsFromCache{
		Database:  queryMeter.Database,
		TableName: meterQueryRowCacheTable,
		Hash:      hash,
		Namespace: queryMeter.Namespace,
		From:      queryMeter.From,
		To:        queryMeter.To,
	}

	sql, args := cacheQuery.toSQL()
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query meter query hash: %w", err)
	}

	defer rows.Close()
	cachedValues, err = cacheQuery.scanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan meter query hash rows: %w", err)
	}

	return cachedValues, nil
}

// storeCachedMeterRows stores new meter query results in the meter_query_hash table
func (c *Connector) storeCachedMeterRows(ctx context.Context, hash string, queryMeter queryMeter, newValues []meterpkg.MeterQueryRow) error {
	insertQuery := insertMeterQueryRowsToCache{
		Database:  queryMeter.Database,
		TableName: meterQueryRowCacheTable,
		Hash:      hash,
		Namespace: queryMeter.Namespace,
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
	tableCreation := createMeterQueryRowsCacheTable{
		Database:  c.config.Database,
		TableName: meterQueryRowCacheTable,
	}

	err := c.config.ClickHouse.Exec(ctx, tableCreation.toSQL())
	if err != nil {
		return fmt.Errorf("create meter_query_hash table: %w", err)
	}

	return nil
}

// findNamespacesToInvalidateCache finds the namespaces that need to be invalidated
func (c *Connector) findNamespacesToInvalidateCache(rawEvents []streaming.RawEvent) []string {
	// Check if any events requires cache invalidation
	now := time.Now().UTC()

	// Remove duplicates from namespaces
	uniqueNamespaces := []string{}
	seen := map[string]struct{}{}

	for _, event := range rawEvents {
		if event.Time.Before(now.Add(-c.config.QueryCacheMinimumCacheableUsageAge)) {
			if _, ok := seen[event.Namespace]; !ok {
				seen[event.Namespace] = struct{}{}
				uniqueNamespaces = append(uniqueNamespaces, event.Namespace)
			}
		}
	}

	return uniqueNamespaces
}

// invalidateCache deletes all cached rows for the specified namespaces
// Potential improvements for finer-grained cache invalidation:
// - invalidate cache for a specific time range
// - invalidate cache for a specific subject (if present in query)
// - invalidate cache for a specific meter (event type)
func (c *Connector) invalidateCache(ctx context.Context, namespaces []string) error {
	if !c.config.QueryCacheEnabled {
		return nil
	}

	if len(namespaces) == 0 {
		return nil
	}

	deleteQuery := deleteCacheForNamespaces{
		Database:   c.config.Database,
		TableName:  meterQueryRowCacheTable,
		Namespaces: namespaces,
	}

	sql, args := deleteQuery.toSQL()

	if err := c.config.ClickHouse.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("delete from cache: %w", err)
	}

	return nil
}
