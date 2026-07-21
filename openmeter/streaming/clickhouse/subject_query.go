package clickhouse

import (
	"fmt"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type listSubjectsQuery struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           *meterpkg.Meter
	From            *time.Time
	To              *time.Time
	Search          *string
}

func (d listSubjectsQuery) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("DISTINCT subject")
	sb.Where(sb.Equal("namespace", d.Namespace))

	// If we have a meter, we add the type filter
	if d.Meter != nil {
		sb.Where(sb.Equal("type", d.Meter.EventType))
	}

	sb.From(tableName)
	sb.OrderBy("subject")

	if d.From != nil {
		sb.Where(sb.GreaterEqualThan("time", d.From.Unix()))
	}

	if d.To != nil {
		sb.Where(sb.LessThan("time", d.To.Unix()))
	}

	if d.Search != nil && *d.Search != "" {
		sb.Where(fmt.Sprintf("positionCaseInsensitive(subject, %s) > 0", sb.Var(*d.Search)))
	}

	sql, args := sb.Build()
	return sql, args
}

const listSubjectsV2DefaultLimit = 100

// eventsSubjectsProjectionName is the aggregate projection on the events table
// that pre-aggregates the distinct (namespace, subject) pairs.
const eventsSubjectsProjectionName = "prj_namespace_subject"

// createEventsSubjectsProjection adds the events subjects projection.
// Adding the projection only applies to newly inserted data; backfilling
// existing data requires manually running
// `ALTER TABLE <events table> MATERIALIZE PROJECTION prj_namespace_subject`.
type createEventsSubjectsProjection struct {
	Database        string
	EventsTableName string
}

func (d createEventsSubjectsProjection) toSQL() string {
	tableName := getTableName(d.Database, d.EventsTableName)

	return fmt.Sprintf(
		"ALTER TABLE %s ADD PROJECTION IF NOT EXISTS %s (SELECT namespace, subject GROUP BY namespace, subject)",
		tableName,
		eventsSubjectsProjectionName,
	)
}

type listSubjectsV2Query struct {
	Database        string
	EventsTableName string
	Params          streaming.ListSubjectsV2Params
	QuerySettings   map[string]string
}

// toSQL builds the subjects listing query. It uses GROUP BY instead of
// DISTINCT so ClickHouse can serve it from the events subjects projection
// (when present) instead of scanning the events table.
func (d listSubjectsV2Query) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("subject")
	query.From(tableName)
	query.Where(query.Equal("namespace", d.Params.Namespace))
	// Empty subjects cannot be ingested through the API, but rows written by
	// direct producers would break the attributed filter (empty values are
	// rejected by pkg/filter) and produce cursors with an empty ID, so they
	// are excluded at the source.
	query.Where(query.NotEqual("subject", ""))

	if d.Params.Key != nil {
		expr := d.Params.Key.SelectWhereExpr("subject", query)
		if expr != "" {
			query.Where(expr)
		}
	}

	// Keyset pagination: the cursor ID holds the last subject key of the previous page.
	if d.Params.Cursor != nil {
		query.Where(query.GreaterThan("subject", d.Params.Cursor.ID))
	}

	query.GroupBy("namespace", "subject")
	query.OrderBy("namespace", "subject")
	query.Limit(lo.FromPtrOr(d.Params.Limit, listSubjectsV2DefaultLimit))

	// Guard the aggregation cost: without the subjects projection this query
	// scans the namespace's events, so the configured query settings (e.g.
	// max_execution_time) must bound it server-side.
	var settings []string
	for key, value := range d.QuerySettings {
		settings = append(settings, fmt.Sprintf("%s = %s", key, value))
	}

	if len(settings) > 0 {
		query.SQL(fmt.Sprintf("SETTINGS %s", strings.Join(settings, ", ")))
	}

	return query.Build()
}
