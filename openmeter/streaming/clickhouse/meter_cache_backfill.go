package clickhouse

import (
	"fmt"
	"strings"
	"time"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// meterCacheBackfill generates the one-time INSERT that populates om_meter_cache with a
// meter's full settled history when its MV is created. It renders the exact SELECT shape
// the MV refreshes use (meterCacheSelectSQL), only without the dirty-bucket restriction:
// backfill rows and refresh rows must be indistinguishable so their overlap around the
// first scheduled refresh resolves by newest-wins instead of diverging.
type meterCacheBackfill struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	Grain           CacheGrain
	MinimumUsageAge time.Duration

	// From/To optionally chunk the backfill (see backfillMonthChunks) so a large meter's
	// history is inserted in bounded pieces instead of one giant INSERT SELECT. From is
	// merged with the meter's EventFrom below; To is exclusive and the settled bound
	// still applies on top of it. Both nil = full settled history in one statement.
	From *time.Time
	To   *time.Time
}

func (d meterCacheBackfill) toSQL() (string, error) {
	// Same lower-bound resolution as the live query (queryMeter.from()): the meter's
	// EventFrom wins over an earlier chunk bound so the backfill never caches events the
	// live query would exclude — cached and live results must stay byte-equal.
	from := d.From
	if d.Meter.EventFrom != nil && (from == nil || d.Meter.EventFrom.After(*from)) {
		from = d.Meter.EventFrom
	}

	selectSQL, err := meterCacheSelectSQL(meterCacheSelectParams{
		Database:        d.Database,
		EventsTableName: d.EventsTableName,
		Namespace:       d.Namespace,
		Meter:           d.Meter,
		Grain:           d.Grain,
		MinimumUsageAge: d.MinimumUsageAge,
		From:            from,
		To:              d.To,
	})
	if err != nil {
		return "", err
	}

	getColumn := columnFactory(d.EventsTableName)

	combineColumns, err := valueExprsCombine(d.Meter, getColumn("data"), getColumn("time"))
	if err != nil {
		return "", err
	}

	// The INSERT names its columns because the SELECT only produces the meter's own
	// combine columns; the remaining aggregation columns take their defaults. The list
	// must mirror the SELECT output order (meterCacheSelectSQL), so the combine column
	// names are taken from the very expressions the SELECT emits.
	columns := []string{"namespace", "meter_key", "meter_hash", "windowstart", "subject", "group_by", "created_at"}
	for _, expr := range combineColumns {
		columns = append(columns, expr[strings.LastIndex(expr, " AS ")+len(" AS "):])
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) %s",
		getTableName(d.Database, meterCacheTableName),
		strings.Join(columns, ", "),
		selectSQL,
	), nil
}

// backfillChunk is one [From, To) piece of a chunked backfill.
type backfillChunk struct {
	From time.Time
	To   time.Time
}

// backfillMonthChunks splits [from, to) on UTC calendar month boundaries. Chunks align
// with the events table's toYYYYMM(time) partitioning, so each backfill INSERT scans at
// most two partitions (first chunk may start mid-month) instead of the whole history.
// Returns nil when the range is empty.
func backfillMonthChunks(from, to time.Time) []backfillChunk {
	from = from.UTC()
	to = to.UTC()

	var chunks []backfillChunk

	for cur := from; cur.Before(to); {
		next := time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
		if next.After(to) {
			next = to
		}

		chunks = append(chunks, backfillChunk{From: cur, To: next})
		cur = next
	}

	return chunks
}
