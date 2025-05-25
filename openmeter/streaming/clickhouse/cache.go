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
func (c *Connector) canQueryBeCached(namespace string, meterDef meterpkg.Meter, queryParams streaming.QueryParams) bool {
	// If caching is disabled, we don't cache anything
	if !c.config.QueryCacheEnabled {
		return false
	}

	// We only cache queries where cachable is set to true
	if !queryParams.Cachable {
		return false
	}

	// If the namespace template is provided and does not match the namespace, we don't cache anything
	if c.namespaceTemplate != nil && !c.namespaceTemplate.MatchString(namespace) {
		return false
	}

	// We can only cache queries that have a from time
	if queryParams.From == nil {
		return false
	}

	from := *queryParams.From
	to := lo.FromPtrOr(queryParams.To, time.Now().UTC())

	// We respect the minimum cacheable usage age by skipping if from is younger than the minimum cacheable usage age
	// This is a shortcut for cache logic we handle cachable period later when we prepare the query
	minFrom := time.Now().UTC().Add(-c.config.QueryCacheMinimumCacheableUsageAge)

	if from.After(minFrom) {
		return false
	}

	// We respect the minimum cacheable query period
	duration := to.Sub(from)

	if duration < c.config.QueryCacheMinimumCacheableQueryPeriod {
		return false
	}

	return meterDef.Aggregation == meterpkg.MeterAggregationSum || meterDef.Aggregation == meterpkg.MeterAggregationCount || meterDef.Aggregation == meterpkg.MeterAggregationMin || meterDef.Aggregation == meterpkg.MeterAggregationMax
}

// executeQueryWithCaching queries the meter view and manages the cache, as:
// 1. Look up cached rows
// 2. Query rows for the period not in cache
// 3. Store the new cachable rows in the cache
// It returns the cached rows and the new rows.
func (c *Connector) executeQueryWithCaching(ctx context.Context, hash string, originalQueryMeter queryMeter) ([]meterpkg.MeterQueryRow, []meterpkg.MeterQueryRow, error) {
	var firstCachedWindowStart, lastCachedWindowEnd *time.Time

	logger := c.config.Logger.With("hash", hash, "from", originalQueryMeter.From, "to", originalQueryMeter.To)

	// Calculate the period to query from the cache
	cacheableQueryMeter, remainingQueryMeter, err := c.prepareCacheableQueryPeriod(originalQueryMeter)
	if err != nil {
		return nil, nil, err
	}

	logger = logger.With("cacheableFrom", cacheableQueryMeter.From, "cacheableTo", cacheableQueryMeter.To)

	// Step 1: Look up cached rows
	cachedRows, err := c.fetchCachedMeterRows(ctx, hash, cacheableQueryMeter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup cached meter rows: %w", err)
	}

	// If we have cached values, add them to the results
	// Also, update the query range to query uncached periods
	if len(cachedRows) > 0 {
		// The cached values don't neccesarly cover the entire cached query period so we need to find the latest cached window
		firstCachedWindowStart, lastCachedWindowEnd = findMinMaxWindows(cachedRows)

		logger = logger.With("firstCachedWindowStart", firstCachedWindowStart, "lastCachedWindowEnd", lastCachedWindowEnd, "cachedRowsCount", len(cachedRows))

		logger.Debug("cached rows found")

		// We use the latest already cached window and use it as the start of the new query period
		// Cache stores data with "from" inclusive and "to" exclusive.
		// So we query fresh data with inclusive from since last cached.
		remainingQueryMeter.From = lastCachedWindowEnd
	} else {
		// If there is no cached data, we query the entire time period
		remainingQueryMeter.From = cacheableQueryMeter.From
	}

	// Check if the query period is covered by the cache
	periodCoveredByCache := false

	// If there is no cached data or there is no to the query period is not covered by the cache
	// When there is no to time we always have to query new rows
	if originalQueryMeter.To != nil && lastCachedWindowEnd != nil {
		// Let's see if we are right at the end of the last cached window
		if originalQueryMeter.To.Equal(*lastCachedWindowEnd) {
			periodCoveredByCache = true
		}
	}

	// Step 2: Query new rows for the uncached time period, if there is any
	var newRows []meterpkg.MeterQueryRow

	logger = logger.With("remainingFrom", remainingQueryMeter.From, "remainingTo", remainingQueryMeter.To)

	if periodCoveredByCache {
		logger.Debug("no new rows to query, cache covers the entire query period")
	} else {
		logger.Debug("querying new rows for period not covered by cache")

		newRows, err = c.queryMeter(ctx, remainingQueryMeter)
		if err != nil {
			return nil, nil, fmt.Errorf("query new meter rows: %w", err)
		}
	}

	// Step 3: Store the new rows in the cache
	var newRowsNotInCache []meterpkg.MeterQueryRow

	// We filter out rows that are after the cacheable query period
	for _, row := range newRows {
		if row.WindowEnd.After(*cacheableQueryMeter.To) {
			continue
		}

		newRowsNotInCache = append(newRowsNotInCache, row)
	}

	// Results can be double cached in the case of parallel queries to handle this,
	// we deduplicate the results while retrieving them from the cache
	if len(newRowsNotInCache) > 0 {
		logger := logger.With("newRowsNotInCacheCount", len(newRowsNotInCache))

		if err := c.storeCachedMeterRows(ctx, hash, cacheableQueryMeter, newRowsNotInCache); err != nil {
			// Log the error but don't fail the query
			logger.Error("failed to store new rows in cache", "error", err)
		} else {
			logger.Debug("new rows stored in cache")
		}
	}

	// Result
	logger.Debug("returning cached and new rows", "rowsCount", len(cachedRows)+len(newRows))

	return cachedRows, newRows, nil
}

// prepareCacheableQueryPeriod prepares the time range for cacheable queries
func (c *Connector) prepareCacheableQueryPeriod(originalQueryMeter queryMeter) (queryMeter, queryMeter, error) {
	cacheableQueryMeter := originalQueryMeter
	remainingQuery := originalQueryMeter
	now := time.Now().UTC()

	if originalQueryMeter.From == nil {
		return cacheableQueryMeter, remainingQuery, fmt.Errorf("from is required for cached queries")
	}

	// Set the end time to now if not provided
	if cacheableQueryMeter.To == nil {
		cacheableQueryMeter.To = &now
	}

	// We do not cache data that is less than minimum cacheable usage age old
	toFreshness := now.Sub(*cacheableQueryMeter.To)
	if toFreshness < c.config.QueryCacheMinimumCacheableUsageAge {
		delta := c.config.QueryCacheMinimumCacheableUsageAge - toFreshness
		cacheableQueryMeter.To = lo.ToPtr(cacheableQueryMeter.To.Add(-delta))
	}

	// Set the window size to day if not provided
	// this window size is the granularity of the cache
	if cacheableQueryMeter.WindowSize == nil {
		cacheableQueryMeter.WindowSize = lo.ToPtr(meterpkg.WindowSizeDay)
	}

	// Align To time to window boundaries
	// This ensures consistent caching periods regardless of query timing
	windowDuration := cacheableQueryMeter.WindowSize.Duration()

	// Align To time to the end of the window
	cacheableQueryMeter.To = lo.ToPtr(cacheableQueryMeter.To.UTC().Truncate(windowDuration))

	// Remaining query is the time period after the last cached window
	remainingQuery.From = cacheableQueryMeter.To

	return cacheableQueryMeter, remainingQuery, nil
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

	// Deduplicate cached values
	// At insert time we can have duplicates for the same window due to parallel queries
	deduplicatedValues, err := dedupeQueryRows(cachedValues, queryMeter.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("deduplicate cached values: %w", err)
	}

	return deduplicatedValues, nil
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

// findMinMaxWindows finds the min and max window start and end times from a list of rows
func findMinMaxWindows(rows []meterpkg.MeterQueryRow) (*time.Time, *time.Time) {
	var min, max *time.Time

	for _, value := range rows {
		if min == nil || value.WindowStart.Before(*min) {
			min = lo.ToPtr(value.WindowStart)
		}

		if max == nil || value.WindowEnd.After(*max) {
			max = lo.ToPtr(value.WindowEnd)
		}
	}

	return min, max
}
