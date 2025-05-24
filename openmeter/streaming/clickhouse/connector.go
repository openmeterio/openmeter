package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"golang.org/x/exp/constraints"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	progressmanagerentity "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ streaming.Connector = (*Connector)(nil)

// Connector implements `ingest.Connector" and `namespace.Handler interfaces.
type Connector struct {
	config            Config
	namespaceTemplate *regexp.Regexp
}

type Config struct {
	Logger              *slog.Logger
	ClickHouse          clickhouse.Conn
	Database            string
	EventsTableName     string
	AsyncInsert         bool
	AsyncInsertWait     bool
	InsertQuerySettings map[string]string
	ProgressManager     progressmanager.Service
	SkipCreateTables    bool
	QueryCacheEnabled   bool
	// Minimum query period that can be cached
	QueryCacheMinimumCacheableQueryPeriod time.Duration
	// Minimum age after usage data is cachable
	QueryCacheMinimumCacheableUsageAge time.Duration
	// Regexp to match namespaces that should be cached
	QueryCacheNamespaceTemplate string
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if c.ClickHouse == nil {
		return fmt.Errorf("clickhouse connection is required")
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	if c.EventsTableName == "" {
		return fmt.Errorf("events table is required")
	}

	if c.ProgressManager == nil {
		return fmt.Errorf("progress manager is required")
	}

	if c.QueryCacheEnabled {
		if c.QueryCacheMinimumCacheableQueryPeriod <= 0 {
			return fmt.Errorf("minimum cacheable query period is required")
		}

		if c.QueryCacheMinimumCacheableUsageAge <= 0 {
			return fmt.Errorf("minimum cacheable usage age is required")
		}
	}

	return nil
}

func New(ctx context.Context, config Config) (*Connector, error) {
	// Validate the config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	// Compile the namespace template
	var namespaceTemplate *regexp.Regexp
	var err error

	if config.QueryCacheNamespaceTemplate != "" {
		namespaceTemplate, err = regexp.Compile(config.QueryCacheNamespaceTemplate)
		if err != nil {
			return nil, fmt.Errorf("namespace template invalid regex: %w", err)
		}
	}

	// Create the connector
	connector := &Connector{
		config:            config,
		namespaceTemplate: namespaceTemplate,
	}

	if !config.SkipCreateTables {
		if err := connector.createTable(ctx); err != nil {
			return nil, fmt.Errorf("create tables: %w", err)
		}
	}

	return connector, nil
}

// createTable creates the tables in ClickHouse
func (c *Connector) createTable(ctx context.Context) error {
	// Create the events table
	err := c.createEventsTable(ctx)
	if err != nil {
		return fmt.Errorf("create events table in clickhouse: %w", err)
	}

	// Create the meter query cache table
	if err := c.createMeterQueryCacheTable(ctx); err != nil {
		return fmt.Errorf("create meter query cache in clickhouse: %w", err)
	}

	return nil
}

func (c *Connector) ListEvents(ctx context.Context, namespace string, params meterevent.ListEventsParams) ([]streaming.RawEvent, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	events, err := c.queryEventsTable(ctx, namespace, params)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("query events: %w", err)
	}

	return events, nil
}

func (c *Connector) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) ([]streaming.RawEvent, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	events, err := c.queryEventsTableV2(ctx, params)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("query events: %w", err)
	}

	return events, nil
}

func (c *Connector) CreateMeter(ctx context.Context, namespace string, meter meterpkg.Meter) error {
	// Do nothing
	return nil
}

func (c *Connector) UpdateMeter(ctx context.Context, namespace string, meter meterpkg.Meter) error {
	// Do nothing
	return nil
}

func (c *Connector) DeleteMeter(ctx context.Context, namespace string, meter meterpkg.Meter) error {
	// Do nothing
	return nil
}

func (c *Connector) QueryMeter(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.QueryParams) ([]meterpkg.MeterQueryRow, error) {
	// Validate params
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	// We sort the group by keys to ensure the order of the group by columns is deterministic
	// It helps testing the SQL queries.
	groupBy := append([]string(nil), params.GroupBy...)
	sort.Strings(groupBy)

	query := queryMeter{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		Meter:           meter,
		From:            params.From,
		To:              params.To,
		Subject:         params.FilterSubject,
		FilterGroupBy:   params.FilterGroupBy,
		GroupBy:         groupBy,
		WindowSize:      params.WindowSize,
		WindowTimeZone:  params.WindowTimeZone,
	}

	// Load cached rows if any
	var err error
	var values []meterpkg.MeterQueryRow

	useCache := c.canQueryBeCached(namespace, meter, params)

	// If the query is cached, we load the cached rows
	if useCache {
		hash := fmt.Sprintf("%x", params.Hash())

		cachedRows, newRows, err := c.executeQueryWithCaching(ctx, hash, query)
		if err != nil {
			return values, fmt.Errorf("query cached rows: %w", err)
		}

		values = mergeMeterQueryRows(meter, params, cachedRows, newRows)
	} else {
		// If the client ID is set, we track track the progress of the query
		if params.ClientID != nil {
			values, err = c.queryMeterWithProgress(ctx, namespace, *params.ClientID, query)
			if err != nil {
				return values, fmt.Errorf("query meter with progress: %w", err)
			}
		} else {
			values, err = c.queryMeter(ctx, query)
			if err != nil {
				return values, fmt.Errorf("query meter: %w", err)
			}
		}
	}

	// If the total usage is queried for a single period (no window size),
	// replace the window start and end with the period for each row.
	// We can still have multiple rows for a single period due to group bys.
	if params.WindowSize == nil {
		for i := range values {
			if params.From != nil {
				values[i].WindowStart = *params.From
			}
			if params.To != nil {
				values[i].WindowEnd = *params.To
			}
		}
	}

	return values, nil
}

func (c *Connector) ListMeterSubjects(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meter.Key == "" {
		return nil, fmt.Errorf("meter is required")
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	subjects, err := c.listMeterViewSubjects(ctx, namespace, meter, params.From, params.To)
	if err != nil {
		if meterpkg.IsMeterNotFoundError(err) {
			return nil, err
		}

		return nil, fmt.Errorf("list meter subjects: %w", err)
	}

	return subjects, nil
}

func (c *Connector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *Connector) DeleteNamespace(ctx context.Context, namespace string) error {
	// We don't delete the event tables as it it reused between namespaces
	return nil
}

func (c *Connector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	rows, err := c.queryCountEvents(ctx, namespace, params)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("query count events: %w", err)
	}

	return rows, nil
}

func (c *Connector) BatchInsert(ctx context.Context, rawEvents []streaming.RawEvent) error {
	var err error

	// Insert raw events
	query := InsertEventsQuery{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Events:          rawEvents,
		QuerySettings:   c.config.InsertQuerySettings,
	}
	sql, args := query.ToSQL()

	// By default, ClickHouse is writing data synchronously.
	// See https://clickhouse.com/docs/en/cloud/bestpractices/asynchronous-inserts
	if c.config.AsyncInsert {
		// With the `wait_for_async_insert` setting, you can configure
		// if you want an insert statement to return with an acknowledgment
		// either immediately after the data got inserted into the buffer.
		err = c.config.ClickHouse.AsyncInsert(ctx, sql, c.config.AsyncInsertWait, args...)
	} else {
		err = c.config.ClickHouse.Exec(ctx, sql, args...)
	}

	if err != nil {
		return fmt.Errorf("failed to batch insert raw events: %w", err)
	}

	// If query cache is enabled, we invalidate the cache if any events are inserted
	if c.config.QueryCacheEnabled {
		// Check if any events requires cache invalidation
		namespacesToInvalidateCache := c.findNamespacesToInvalidateCache(rawEvents)

		if len(namespacesToInvalidateCache) > 0 {
			if err := c.invalidateCache(ctx, namespacesToInvalidateCache); err != nil {
				return fmt.Errorf("invalidate query cache: %w", err)
			}

			c.config.Logger.Info("invalidated query cache for namespaces", "namespaces", namespacesToInvalidateCache)
		}
	}

	return nil
}

func (c *Connector) createEventsTable(ctx context.Context) error {
	table := createEventsTable{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return nil
}

// ValidateJSONPath checks if the given JSON path is valid by executing a simple query with it.
func (c *Connector) ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error) {
	query := validateJsonPathQuery{
		jsonPath: jsonPath,
	}

	sql, args := query.toSQL()

	err := c.config.ClickHouse.Exec(ctx, sql, args...)
	if err != nil {
		// Code 36 means bad arguments
		// See: https://github.com/ClickHouse/ClickHouse/blob/master/src/Common/ErrorCodes.cpp
		if strings.Contains(err.Error(), "code: 36") {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (c *Connector) queryEventsTable(ctx context.Context, namespace string, params meterevent.ListEventsParams) ([]streaming.RawEvent, error) {
	var err error

	table := queryEventsTable{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		From:            params.From,
		To:              params.To,
		IngestedAtFrom:  params.IngestedAtFrom,
		IngestedAtTo:    params.IngestedAtTo,
		ID:              params.ID,
		Subject:         params.Subject,
		Limit:           params.Limit,
	}

	// If the client ID is set, we track track the progress of the query
	if params.ClientID != nil {
		// Build SQL query to count the total number of rows
		countSQL, countArgs := table.toCountRowSQL()

		ctx, err = c.withProgressContext(ctx, namespace, *params.ClientID, countSQL, countArgs)
		// Log error but don't return it
		if err != nil {
			c.config.Logger.Error("failed track progress", "error", err, "clientId", *params.ClientID)
		}
	}

	sql, args := table.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, models.NewNamespaceNotFoundError(namespace)
		}

		return nil, fmt.Errorf("query events table query: %w", err)
	}

	defer rows.Close()

	events := []streaming.RawEvent{}

	for rows.Next() {
		var rawEvent streaming.RawEvent
		if err = rows.ScanStruct(&rawEvent); err != nil {
			return nil, err
		}

		events = append(events, rawEvent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}

// queryEventsTableV2 is similar to queryEventsTable but with advanced filtering options.
func (c *Connector) queryEventsTableV2(ctx context.Context, params meterevent.ListEventsV2Params) ([]streaming.RawEvent, error) {
	var err error

	// Create query struct
	queryBuilder := queryEventsTableV2{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Params:          params,
	}

	// If a client ID is provided, track progress
	if params.ClientID != nil {
		// Build SQL query to count the total number of rows
		countSQL, countArgs := queryBuilder.toCountRowSQL()

		ctx, err = c.withProgressContext(ctx, params.Namespace, *params.ClientID, countSQL, countArgs)
		// Log error but don't return it
		if err != nil {
			c.config.Logger.Error("failed track progress", "error", err, "clientId", *params.ClientID)
		}
	}

	sql, args := queryBuilder.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query events table query: %w", err)
	}

	defer rows.Close()

	events := []streaming.RawEvent{}

	for rows.Next() {
		var event streaming.RawEvent

		err = rows.ScanStruct(&event)
		if err != nil {
			return nil, fmt.Errorf("scan raw event: %w", err)
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}

func (c *Connector) queryCountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	table := queryCountEvents{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		From:            params.From,
	}

	sql, args := table.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, models.NewNamespaceNotFoundError(namespace)
		}

		return nil, fmt.Errorf("query events count query: %w", err)
	}

	defer rows.Close()

	results := []streaming.CountEventRow{}

	for rows.Next() {
		result := streaming.CountEventRow{}

		if err = rows.Scan(&result.Count, &result.Subject); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// queryMeterWithProgress queries the meter and returns the rows
// It also tracks the progress of the query
func (c *Connector) queryMeterWithProgress(ctx context.Context, namespace string, clientID string, query queryMeter) ([]meterpkg.MeterQueryRow, error) {
	var err error

	// Build SQL query to count the total number of rows
	countSQL, countArgs := query.toCountRowSQL()

	ctx, err = c.withProgressContext(ctx, namespace, clientID, countSQL, countArgs)
	// Log error but don't return it
	if err != nil {
		c.config.Logger.Error("failed track progress", "error", err, "clientId", clientID)
	}

	// Query the meter
	values, err := c.queryMeter(ctx, query)
	if err != nil {
		return values, fmt.Errorf("query meter rows: %w", err)
	}

	return values, nil
}

// queryMeter queries the meter and returns the rows
func (c *Connector) queryMeter(ctx context.Context, query queryMeter) ([]meterpkg.MeterQueryRow, error) {
	// Build the SQL query
	sql, args, err := query.toSQL()
	if err != nil {
		return nil, fmt.Errorf("build sql query: %w", err)
	}

	start := time.Now()

	// Query the meter view
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, meterpkg.NewMeterNotFoundError(query.Meter.Key)
		}
		return nil, fmt.Errorf("clickhouse query: %w", err)
	}

	defer rows.Close()

	elapsed := time.Since(start)
	c.config.Logger.Debug("clickhouse query executed", "elapsed", elapsed.String(), "sql", sql, "args", args)

	// Scan the rows
	values, err := query.scanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan query rows: %w", err)
	}

	return values, nil
}

func (c *Connector) listMeterViewSubjects(ctx context.Context, namespace string, meter meterpkg.Meter, from *time.Time, to *time.Time) ([]string, error) {
	query := listMeterSubjectsQuery{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		Meter:           meter,
		From:            from,
		To:              to,
	}

	sql, args := query.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, meterpkg.NewMeterNotFoundError(meter.Key)
		}

		return nil, fmt.Errorf("list meter view subjects: %w", err)
	}

	defer rows.Close()

	subjects := []string{}
	for rows.Next() {
		var subject string
		if err = rows.Scan(&subject); err != nil {
			return nil, err
		}

		subjects = append(subjects, subject)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return subjects, nil
}

// withProgressContext wraps the context with a progress tracking
func (c *Connector) withProgressContext(ctx context.Context, namespace string, clientID string, countSQL string, countArgs []interface{}) (context.Context, error) {
	totalRows := uint64(0)
	successRows := uint64(0)

	// Count the total number of rows
	countRows, err := c.config.ClickHouse.Query(ctx, countSQL, countArgs...)
	if err != nil {
		return ctx, fmt.Errorf("count query: %w", err)
	}

	defer countRows.Close()

	for countRows.Next() {
		if err := countRows.Scan(&totalRows); err != nil {
			return ctx, fmt.Errorf("count row scan: %w", err)
		}
	}

	if err := countRows.Err(); err != nil {
		return ctx, fmt.Errorf("count rows error: %w", err)
	}

	// Use context to pass a call back for progress and profile info
	ctx = clickhouse.Context(ctx, clickhouse.WithProgress(func(p *clickhouse.Progress) {
		successRows += p.Rows

		progress := progressmanagerentity.Progress{
			ProgressID: progressmanagerentity.ProgressID{
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ID: clientID,
			},
			Total: totalRows,
			// Rows it scans is maybe more than the total rows returned by the count query
			Success:   min(successRows, totalRows),
			UpdatedAt: time.Now(),
		}

		// Update progress
		err = c.config.ProgressManager.UpsertProgress(ctx, progressmanagerentity.UpsertProgressInput{
			Progress: progress,
		})
		// Log error but don't return it
		if err != nil {
			c.config.Logger.Error("failed to upsert progress", "error", err)
		}
	}))

	return ctx, nil
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
