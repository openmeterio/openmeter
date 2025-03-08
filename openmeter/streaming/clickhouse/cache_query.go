package raw_events

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/huandu/go-sqlbuilder"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/samber/lo"
)

// Cache-related constants
const (
	// MeterQueryCacheTable is the name of the table that stores cached meter query results
	MeterQueryCacheTable = "meter_query_cache"
)

// createMeterQueryHashTable is a query to create the meter_query_hash table
type createMeterQueryHashTable struct {
	Database  string
	TableName string
}

func (d createMeterQueryHashTable) toSQL() string {
	tableName := getTableName(d.Database, d.TableName)

	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			hash String,
			namespace String,
			window_start DateTime,
			window_end DateTime,
			value Float64,
			subject String,
			group_by Map(String, String),
			created_at DateTime DEFAULT now()
		)
		ENGINE = MergeTree()
		ORDER BY (namespace, hash, window_start, window_end)
		PARTITION BY toYYYYMM(window_start)
		TTL created_at + INTERVAL 30 DAY;`, tableName)
}

// insertMeterQueryCachedRows is a query to insert rows into the meter_query_hash table
type insertMeterQueryCachedRows struct {
	Database  string
	TableName string
	Hash      string
	Namespace string
	QueryRows []meterpkg.MeterQueryRow
}

func (d insertMeterQueryCachedRows) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.TableName)
	sb := sqlbuilder.ClickHouse.NewInsertBuilder()
	sb.InsertInto(tableName)
	sb.Cols("hash", "namespace", "window_start", "window_end", "value", "subject", "group_by")

	var args []interface{}
	for _, row := range d.QueryRows {
		groupBy := make(map[string]string)
		for k, v := range row.GroupBy {
			groupBy[k] = lo.FromPtrOr(v, "")
		}

		args = append(args,
			d.Hash,
			d.Namespace,
			row.WindowStart,
			row.WindowEnd,
			row.Value,
			lo.FromPtrOr(row.Subject, ""),
			groupBy,
		)

		sb.Values("?", "?", "?", "?", "?", "?", "?")
	}

	sql, _ := sb.Build()
	return sql, args
}

// getMeterQueryCachedRows is a query to retrieve rows from the meter_query_hash table
type getMeterQueryCachedRows struct {
	Database  string
	TableName string
	Hash      string
	Namespace string
	From      *time.Time
	To        *time.Time
}

func (d getMeterQueryCachedRows) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.TableName)
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("window_start", "window_end", "value", "subject", "group_by")
	sb.From(tableName)
	sb.Where(sb.Equal("hash", d.Hash))
	sb.Where(sb.Equal("namespace", d.Namespace))

	if d.From != nil {
		sb.Where(sb.GreaterEqualThan("window_start", d.From.Unix()))
	}

	if d.To != nil {
		sb.Where(sb.LessEqualThan("window_end", d.To.Unix()))
	}

	sb.OrderBy("window_start")

	sql, args := sb.Build()
	return sql, args
}

func (d getMeterQueryCachedRows) scanRows(rows driver.Rows) ([]meterpkg.MeterQueryRow, error) {
	values := []meterpkg.MeterQueryRow{}

	for rows.Next() {
		row := meterpkg.MeterQueryRow{
			GroupBy: map[string]*string{},
		}

		var rowSubject string
		var rowGroupBy map[string]string

		if err := rows.Scan(&row.WindowStart, &row.WindowEnd, &row.Value, &rowSubject, &rowGroupBy); err != nil {
			return values, fmt.Errorf("scan meter query hash row: %w", err)
		}

		if rowSubject != "" {
			row.Subject = &rowSubject
		}

		for k, v := range rowGroupBy {
			if v != "" {
				row.GroupBy[k] = &v
			} else {
				row.GroupBy[k] = nil
			}
		}

		values = append(values, row)
	}

	err := rows.Err()
	if err != nil {
		return values, fmt.Errorf("rows error: %w", err)
	}

	return values, nil
}

// Connector methods for cache operations

// lookupCachedMeterRows queries the meter_query_hash table for cached results
func (c *Connector) lookupCachedMeterRows(ctx context.Context, hash string, hp queryMeter) ([]meterpkg.MeterQueryRow, error) {
	var cachedValues []meterpkg.MeterQueryRow

	hashQuery := getMeterQueryCachedRows{
		Database:  hp.Database,
		TableName: MeterQueryCacheTable,
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

// cacheMeterRows stores new meter query results in the meter_query_hash table
func (c *Connector) cacheMeterRows(ctx context.Context, hash string, hp queryMeter, newValues []meterpkg.MeterQueryRow) error {
	insertQuery := insertMeterQueryCachedRows{
		Database:  hp.Database,
		TableName: MeterQueryCacheTable,
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
	table := createMeterQueryHashTable{
		Database:  c.config.Database,
		TableName: MeterQueryCacheTable,
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create meter_query_hash table: %w", err)
	}

	return nil
}
