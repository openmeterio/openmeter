package raw_events

import (
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// meterQueryRowCacheTable is the name of the table that stores cached meter query rows
const meterQueryRowCacheTable = "meterqueryrow_cache"

// createMeterQueryRowsCacheTable is a query to create the cache table
type createMeterQueryRowsCacheTable struct {
	Database  string
	TableName string
}

// toSQL converts the createMeterQueryRowsCacheTable struct to a SQL query
func (d createMeterQueryRowsCacheTable) toSQL() string {
	tableName := getTableName(d.Database, d.TableName)

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("hash", "String")
	sb.Define("window_start", "DateTime")
	sb.Define("window_end", "DateTime")
	sb.Define("value", "Float64")
	sb.Define("subject", "String")
	sb.Define("group_by", "Map(String, String)")
	sb.Define("created_at", "DateTime DEFAULT now()")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(window_start)")
	sb.SQL("ORDER BY (namespace, hash, window_start, window_end)")
	sb.SQL("TTL created_at + INTERVAL 30 DAY")

	sql, _ := sb.Build()
	return sql
}

// insertMeterQueryRowsToCache is a query to insert rows into the cache table
type insertMeterQueryRowsToCache struct {
	Database  string
	TableName string
	Hash      string
	Namespace string
	QueryRows []meterpkg.MeterQueryRow
}

// toSQL converts the insertMeterQueryRowsToCache struct to a SQL query
func (d insertMeterQueryRowsToCache) toSQL() (string, []interface{}) {
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

// getMeterQueryRowsFromCache is a query to retrieve rows from the cache table
type getMeterQueryRowsFromCache struct {
	Database  string
	TableName string
	Hash      string
	Namespace string
	From      *time.Time
	To        *time.Time
}

// toSQL converts the getMeterQueryRowsFromCache struct to a SQL query
func (d getMeterQueryRowsFromCache) toSQL() (string, []interface{}) {
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

// scanMeterQueryRowsFromCache scans the rows from the cache table
func (d getMeterQueryRowsFromCache) scanRows(rows driver.Rows) ([]meterpkg.MeterQueryRow, error) {
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
