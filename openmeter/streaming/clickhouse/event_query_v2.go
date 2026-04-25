package clickhouse

import (
	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const eventQueryV2DefaultLimit = 100

// queryEventsTableV2 struct holds the parameters for v2 event queries
type queryEventsTableV2 struct {
	Database        string
	EventsTableName string
	Params          streaming.ListEventsV2Params
}

// toSQL generates the SQL query and arguments for fetching events with v2 filtering
func (q queryEventsTableV2) toSQL() (string, []interface{}) {
	tableName := getTableName(q.Database, q.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("id", "type", "subject", "source", "time", "data", "ingested_at", "stored_at", "store_row_id")

	// Select customer_id column if customer filter is provided
	if q.Params.Customers != nil {
		query = selectCustomerIdColumn(q.EventsTableName, *q.Params.Customers, query)
	}

	query.From(tableName)

	// Base filter for namespace
	query.Where(query.Equal("namespace", q.Params.Namespace))

	if q.Params.ID != nil {
		expr := q.Params.ID.SelectWhereExpr("id", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.Source != nil {
		expr := q.Params.Source.SelectWhereExpr("source", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.Subject != nil {
		expr := q.Params.Subject.SelectWhereExpr("subject", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.Customers != nil {
		query = customersWhere(tableName, *q.Params.Customers, query)
	}

	if q.Params.Type != nil {
		expr := q.Params.Type.SelectWhereExpr("type", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.Time != nil {
		expr := q.Params.Time.SelectWhereExpr("time", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.IngestedAt != nil {
		expr := q.Params.IngestedAt.SelectWhereExpr("ingested_at", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.StoredAt != nil {
		expr := q.Params.StoredAt.SelectWhereExpr("stored_at", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	sortCol := string(q.Params.SortBy)
	if sortCol == "" {
		sortCol = string(streaming.EventSortFieldTime)
	}

	if q.Params.Cursor != nil {
		if q.Params.SortOrder == sortx.OrderAsc {
			query.Where(
				// First filter by sort column
				query.GreaterEqualThan(sortCol, q.Params.Cursor.Time.Unix()),
				// Tie-break by store_row_id (per-row unique ULID) so same-second
				// events are not skipped across page boundaries.
				query.Or(
					query.GreaterThan(sortCol, q.Params.Cursor.Time.Unix()),
					query.GreaterThan("store_row_id", q.Params.Cursor.ID),
				),
			)
		} else {
			query.Where(
				// First filter by sort column
				query.LessEqualThan(sortCol, q.Params.Cursor.Time.Unix()),
				// Tie-break by store_row_id (per-row unique ULID) so same-second
				// events are not skipped across page boundaries.
				query.Or(
					query.LessThan(sortCol, q.Params.Cursor.Time.Unix()),
					query.LessThan("store_row_id", q.Params.Cursor.ID),
				),
			)
		}
	}

	switch q.Params.SortOrder {
	case sortx.OrderAsc:
		query.OrderByAsc(sortCol).OrderByAsc("store_row_id")
	case sortx.OrderDesc:
		fallthrough
	default:
		query.OrderByDesc(sortCol).OrderByDesc("store_row_id")
	}

	// Apply limit
	query.Limit(lo.FromPtrOr(q.Params.Limit, eventQueryV2DefaultLimit))

	return query.Build()
}

// toCountRowSQL returns the SQL query for the estimated number of rows for tracking progress
func (q queryEventsTableV2) toCountRowSQL() (string, []interface{}) {
	tableName := getTableName(q.Database, q.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() as total")
	query.From(tableName)

	// Base filter for namespace
	query.Where(query.Equal("namespace", q.Params.Namespace))

	// Apply basic filters that might significantly affect the count
	// Note: We don't include all filters here for performance reasons

	if q.Params.Type != nil {
		expr := q.Params.Type.SelectWhereExpr("type", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.Subject != nil {
		expr := q.Params.Subject.SelectWhereExpr("subject", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	if q.Params.Customers != nil {
		query = customersWhere(tableName, *q.Params.Customers, query)
	}

	if q.Params.Time != nil {
		expr := q.Params.Time.SelectWhereExpr("time", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	return query.Build()
}
