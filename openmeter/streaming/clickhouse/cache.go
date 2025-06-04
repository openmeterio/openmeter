package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/hasher"
)

var (
	defaultCacheWindowSize     = meterpkg.WindowSizeDay
	defaultCacheWindowTimeZone = time.UTC
)

// Hash returns a deterministic hash for the QueryParams.
// It implements the hasher.Hasher interface.
func QueryParamsHash(p streaming.QueryParams) hasher.Hash {
	h := xxhash.New()

	// Hash FilterSubject (sort for determinism)
	if len(p.FilterSubject) > 0 {
		sorted := make([]string, len(p.FilterSubject))
		copy(sorted, p.FilterSubject)
		sort.Strings(sorted)
		_, _ = h.WriteString(strings.Join(sorted, ","))
	}

	// Hash FilterGroupBy (sort keys and values for determinism)
	if len(p.FilterGroupBy) > 0 {
		keys := make([]string, 0, len(p.FilterGroupBy))
		for k := range p.FilterGroupBy {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			_, _ = h.WriteString(k)
			values := make([]string, len(p.FilterGroupBy[k]))
			copy(values, p.FilterGroupBy[k])
			sort.Strings(values)
			_, _ = h.WriteString(strings.Join(values, ","))
		}
	}

	// Hash GroupBy (sort for determinism)
	if len(p.GroupBy) > 0 {
		sorted := make([]string, len(p.GroupBy))
		copy(sorted, p.GroupBy)
		sort.Strings(sorted)
		_, _ = h.WriteString(strings.Join(sorted, ","))
	}

	// Hash WindowSize
	if p.WindowSize != nil {
		_, _ = h.WriteString(string(*p.WindowSize))
	} else {
		// Default cache granularity is day
		_, _ = h.WriteString(string(defaultCacheWindowSize))
	}

	// Hash WindowTimeZone
	if p.WindowTimeZone != nil {
		_, _ = h.WriteString(p.WindowTimeZone.String())
	} else {
		// Default timezone is UTC
		_, _ = h.WriteString(defaultCacheWindowTimeZone.String())
	}

	return h.Sum64()
}

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
func (c *Connector) executeQueryWithCaching(ctx context.Context, hash string, originalQueryMeter queryMeter) ([]meterpkg.MeterQueryRow, error) {
	var firstCachedWindowStart *time.Time
	var lastCachedWindowEnd *time.Time

	var beforeCacheRows, cachedRows, afterCacheRows []meterpkg.MeterQueryRow

	logger := c.config.Logger.With("hash", hash, "from", originalQueryMeter.From, "to", originalQueryMeter.To)

	// Calculate the period to query from the cache
	cacheableQueryMeter, err := c.prepareCacheableQueryPeriod(originalQueryMeter)
	if err != nil {
		return nil, err
	}

	// Step 1: Look up cached rows
	cachedRows, err = c.fetchCachedMeterRows(ctx, hash, cacheableQueryMeter)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup cached meter rows: %w", err)
	}

	timeWindowGap := isTimeWindowGap(*cacheableQueryMeter.From, *cacheableQueryMeter.To, *cacheableQueryMeter.WindowSize, cachedRows)

	logger = logger.With("cachedRowsCount", len(cachedRows), "timeWindowGap", timeWindowGap)

	// If we have cached values, add them to the results
	// Also, update the query range to query uncached periods
	if len(cachedRows) > 0 && !timeWindowGap {
		// The cached values don't neccesarly cover the entire cached query period so we need to find the latest cached window
		// We find the latest already cached window and use it as the start of the new query period
		for _, cachedValue := range cachedRows {
			if firstCachedWindowStart == nil || cachedValue.WindowStart.Before(*firstCachedWindowStart) {
				firstCachedWindowStart = lo.ToPtr(cachedValue.WindowStart)
			}

			if lastCachedWindowEnd == nil || cachedValue.WindowEnd.After(*lastCachedWindowEnd) {
				lastCachedWindowEnd = lo.ToPtr(cachedValue.WindowEnd)
			}
		}

		logger = logger.With("firstCachedWindowStart", firstCachedWindowStart, "lastCachedWindowEnd", lastCachedWindowEnd)

		logger.Debug("cached rows found")
	} else {
		// If there is no cached data, we query the entire time period
		// We add it to the before cache rows to be returned but it doesn't matter if we add to before or after cache rows variable
		beforeCacheRows, err = c.queryMeter(ctx, originalQueryMeter)
		if err != nil {
			return nil, fmt.Errorf("query rows: %w", err)
		}

		logger.Debug("query the entire time period", "count", len(beforeCacheRows))
	}

	// Step 2: Query new rows for the uncached time period, if there is any
	wg := sync.WaitGroup{}
	errChan := make(chan error)

	// Start a goroutine to collect rows and errors
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Step 2.1: Query rows before the first cached window
	if firstCachedWindowStart != nil && !firstCachedWindowStart.Equal(*originalQueryMeter.From) {
		wg.Add(1)

		go func() {
			defer wg.Done()

			beforeCacheQueryMeter := originalQueryMeter
			beforeCacheQueryMeter.From = originalQueryMeter.From
			beforeCacheQueryMeter.To = firstCachedWindowStart
			beforeCacheQueryMeter.WindowSize = cacheableQueryMeter.WindowSize

			beforeCacheRows, err = c.queryMeter(ctx, beforeCacheQueryMeter)
			if err != nil {
				errChan <- fmt.Errorf("query rows before first cached window: %w", err)
				return
			}

			logger.Debug("querying before first cached window", "queryFrom", beforeCacheQueryMeter.From, "queryTo", beforeCacheQueryMeter.To, "countBeforeRows", len(beforeCacheRows))
		}()
	}

	// Step 2.2: Query new rows for the uncached time period, if there is any
	if lastCachedWindowEnd != nil && !lastCachedWindowEnd.Equal(*originalQueryMeter.To) {
		wg.Add(1)

		go func() {
			defer wg.Done()

			afterCacheQueryMeter := originalQueryMeter
			afterCacheQueryMeter.From = lastCachedWindowEnd
			afterCacheQueryMeter.To = originalQueryMeter.To
			afterCacheQueryMeter.WindowSize = cacheableQueryMeter.WindowSize

			afterCacheRows, err = c.queryMeter(ctx, afterCacheQueryMeter)
			if err != nil {
				errChan <- fmt.Errorf("query rows after last cached window: %w", err)
				return
			}

			logger.Debug("querying after last cached window", "queryFrom", afterCacheQueryMeter.From, "queryTo", afterCacheQueryMeter.To, "countAfterRows", len(afterCacheRows))
		}()
	}

	// Collect rows and errors
	var newRowsNotInCache []meterpkg.MeterQueryRow
	var errs []error

	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	// Step 3: Store the new rows in the cache
	// We filter out rows that are non cacheable
	newRowsNotInCache = filterRowsOutOfPeriod(
		*cacheableQueryMeter.From,
		*cacheableQueryMeter.To,
		*cacheableQueryMeter.WindowSize,
		append(beforeCacheRows, afterCacheRows...),
	)

	// Materialize the cache rows for the cachable query period where there are gaps in the cache
	materializedRows, err := c.materializeCacheRows(
		*cacheableQueryMeter.From,
		*cacheableQueryMeter.To,
		*cacheableQueryMeter.WindowSize,
		concatAppend([][]meterpkg.MeterQueryRow{newRowsNotInCache, cachedRows}),
	)
	if err != nil {
		return nil, fmt.Errorf("materialize cache rows: %w", err)
	}

	// Add the materialized rows to the new rows not in cache
	if len(materializedRows) > 0 {
		newRowsNotInCache = append(newRowsNotInCache, materializedRows...)
	}

	// Store new cachable rows in the cache
	// Due to parallel requests and race condition we can end up with duplicates in the cache.
	// This is handled at cached row retreival with deduplication
	if len(newRowsNotInCache) > 0 {
		logger := logger.With("newRowsNotInCacheCount", len(newRowsNotInCache))

		if err := c.storeCachedMeterRows(ctx, hash, cacheableQueryMeter, newRowsNotInCache); err != nil {
			// Log the error but don't fail the query
			logger.Error("failed to store new rows in cache", "error", err)
		} else {
			logger.Debug("new rows stored in cache")
		}
	}

	// Filter out materialized rows from cached rows
	cachedRowsWithValue := lo.Filter(cachedRows, func(row meterpkg.MeterQueryRow, _ int) bool {
		return row.Value != cacheNoValue
	})

	// Result
	resultRows := concatAppend([][]meterpkg.MeterQueryRow{beforeCacheRows, cachedRowsWithValue, afterCacheRows})

	// Sort results by window start
	sort.Slice(resultRows, func(i, j int) bool {
		return resultRows[i].WindowStart.Before(resultRows[j].WindowStart)
	})

	logger.Debug("returning cached and new rows", "resultRowsCount", len(resultRows))

	return filterOutNaNValues(resultRows), nil
}

// prepareCacheableQueryPeriod prepares the time range for cacheable queries
func (c *Connector) prepareCacheableQueryPeriod(originalQueryMeter queryMeter) (queryMeter, error) {
	cacheableQueryMeter := originalQueryMeter
	now := time.Now().UTC()

	if originalQueryMeter.From == nil {
		return cacheableQueryMeter, fmt.Errorf("from is required for cached queries")
	}

	// Set the window size to day if not provided
	// this window size is the granularity of the cache
	// This is the same window size for both cached and remaining query,
	// in the row merge logic we transform the window size to the original query window size.
	if cacheableQueryMeter.WindowSize == nil {
		cacheableQueryMeter.WindowSize = &defaultCacheWindowSize
	}

	// From must be rounded to the window size to ensure we don't have incomplete windows in the cache
	cacheableQueryMeter.From = lo.ToPtr(cacheableQueryMeter.From.Truncate(cacheableQueryMeter.WindowSize.Duration()))

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

	// Align To time to window boundaries
	// This ensures consistent caching periods regardless of query timing
	windowDuration := cacheableQueryMeter.WindowSize.Duration()

	// Align To time to the end of the window
	cacheableQueryMeter.To = lo.ToPtr(cacheableQueryMeter.To.UTC().Truncate(windowDuration))

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
