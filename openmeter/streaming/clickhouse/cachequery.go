package clickhouse

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
func (tableCreation createMeterQueryRowsCacheTable) toSQL() string {
	tableName := getTableName(tableCreation.Database, tableCreation.TableName)

	builder := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	builder.CreateTable(tableName)
	builder.IfNotExists()
	builder.Define("namespace", "String")
	builder.Define("hash", "String")
	builder.Define("window_start", "DateTime")
	builder.Define("window_end", "DateTime")
	builder.Define("value", "Float64")
	builder.Define("subject", "String")
	builder.Define("group_by", "Map(String, String)")
	builder.Define("created_at", "DateTime DEFAULT now()")
	builder.SQL("ENGINE = MergeTree")
	builder.SQL("PARTITION BY toYYYYMM(window_start)")
	builder.SQL("ORDER BY (namespace, hash, window_start, window_end)")
	builder.SQL("TTL created_at + INTERVAL 30 DAY")

	sql, _ := builder.Build()
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

// toSQL converts the insertMeterQueryRowsToCache struct to a SQL query with parameters
func (insertQuery insertMeterQueryRowsToCache) toSQL() (string, []interface{}) {
	tableName := getTableName(insertQuery.Database, insertQuery.TableName)
	builder := sqlbuilder.ClickHouse.NewInsertBuilder()
	builder.InsertInto(tableName)
	builder.Cols("hash", "namespace", "window_start", "window_end", "value", "subject", "group_by")

	var args []interface{}
	for _, row := range insertQuery.QueryRows {
		groupBy := make(map[string]string)
		for key, value := range row.GroupBy {
			groupBy[key] = lo.FromPtrOr(value, "")
		}

		args = append(args,
			insertQuery.Hash,
			insertQuery.Namespace,
			row.WindowStart,
			row.WindowEnd,
			row.Value,
			lo.FromPtrOr(row.Subject, ""),
			groupBy,
		)

		builder.Values("?", "?", "?", "?", "?", "?", "?")
	}

	sql, _ := builder.Build()
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

// toSQL converts the getMeterQueryRowsFromCache struct to a SQL query with parameters
func (queryCache getMeterQueryRowsFromCache) toSQL() (string, []interface{}) {
	tableName := getTableName(queryCache.Database, queryCache.TableName)
	builder := sqlbuilder.ClickHouse.NewSelectBuilder()
	builder.Select("window_start", "window_end", "value", "subject", "group_by")
	builder.From(tableName)
	builder.Where(builder.Equal("hash", queryCache.Hash))
	builder.Where(builder.Equal("namespace", queryCache.Namespace))

	if queryCache.From != nil {
		builder.Where(builder.GreaterEqualThan("window_start", queryCache.From.Unix()))
	}

	if queryCache.To != nil {
		// This is inclusive of the to time even if we usually query up to the end of the previous window
		// This is because we query window_ends not the row data.
		builder.Where(builder.LessEqualThan("window_end", queryCache.To.Unix()))
	}

	builder.OrderBy("window_start")

	sql, args := builder.Build()
	return sql, args
}

// processDatabaseRows processes database rows and returns structured MeterQueryRow objects
func (queryCache getMeterQueryRowsFromCache) scanRows(rows driver.Rows) ([]meterpkg.MeterQueryRow, error) {
	queryRows := []meterpkg.MeterQueryRow{}

	for rows.Next() {
		currentRow := meterpkg.MeterQueryRow{
			GroupBy: map[string]*string{},
		}

		var rowSubject string
		var rowGroupBy map[string]string

		if err := rows.Scan(&currentRow.WindowStart, &currentRow.WindowEnd, &currentRow.Value, &rowSubject, &rowGroupBy); err != nil {
			return queryRows, fmt.Errorf("scan meter query hash row: %w", err)
		}

		if rowSubject != "" {
			currentRow.Subject = &rowSubject
		}

		for groupKey, groupValue := range rowGroupBy {
			if groupValue != "" {
				currentRow.GroupBy[groupKey] = &groupValue
			} else {
				currentRow.GroupBy[groupKey] = nil
			}
		}

		queryRows = append(queryRows, currentRow)
	}

	err := rows.Err()
	if err != nil {
		return queryRows, fmt.Errorf("rows error: %w", err)
	}

	return queryRows, nil
}

// deleteCacheForNamespaces is a query to delete rows from the cache table for specific namespaces
type deleteCacheForNamespaces struct {
	Database   string
	TableName  string
	Namespaces []string
}

// toSQL converts the deleteCacheForNamespaces struct to a SQL query with parameters
func (deleteQuery deleteCacheForNamespaces) toSQL() (string, []interface{}) {
	tableName := getTableName(deleteQuery.Database, deleteQuery.TableName)
	builder := sqlbuilder.ClickHouse.NewDeleteBuilder()
	builder.DeleteFrom(tableName)

	builder.Where(builder.In("namespace", deleteQuery.Namespaces))
	sql, args := builder.Build()
	return sql, args
}
