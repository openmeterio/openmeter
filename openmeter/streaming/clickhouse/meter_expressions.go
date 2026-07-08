package clickhouse

import (
	"fmt"
	"slices"
	"strings"

	"github.com/huandu/go-sqlbuilder"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// This file is the single source of truth for the SQL expressions shared between the live
// meter query (meter_query.go) and the meter cache (per-meter refreshable materialized
// views, their backfill inserts, and the cached read legs). The cache must serve rows that
// are byte-equal to what the live query would return, so window bucketing, value
// extraction, and group-by dimensions have to be built from these builders on both paths;
// duplicating the expressions would let cached and live results drift apart silently.

// windowExprs returns the windowstart/windowend SELECT expressions bucketing timeExpr into
// windowSize windows, evaluated in the tz timezone (an IANA name embedded as a literal).
func windowExprs(windowSize meterpkg.WindowSize, timeExpr string, tz string) ([]string, error) {
	switch windowSize {
	case meterpkg.WindowSizeMinute:
		return []string{
			fmt.Sprintf("tumbleStart(%s, toIntervalMinute(1), '%s') AS windowstart", timeExpr, tz),
			fmt.Sprintf("tumbleEnd(%s, toIntervalMinute(1), '%s') AS windowend", timeExpr, tz),
		}, nil

	case meterpkg.WindowSizeHour:
		return []string{
			fmt.Sprintf("tumbleStart(%s, toIntervalHour(1), '%s') AS windowstart", timeExpr, tz),
			fmt.Sprintf("tumbleEnd(%s, toIntervalHour(1), '%s') AS windowend", timeExpr, tz),
		}, nil

	case meterpkg.WindowSizeDay:
		return []string{
			fmt.Sprintf("tumbleStart(%s, toIntervalDay(1), '%s') AS windowstart", timeExpr, tz),
			"windowstart + toIntervalDay(1) AS windowend",
		}, nil

	case meterpkg.WindowSizeMonth:
		return []string{
			// We need to convert the tumbleStart and tumbleEnd to DateTime, as otherwise we got a Date type. Given
			// we are scanning the result into a time.Time, we will end up with the correct date in UTC. In case the timezone
			// is not UTC, the returned values will be offset by the timezone difference.
			//
			// e.g.:
			//  if timezone is Europe/Budapest, then if we are not casting to DateTime, then:
			// 	 tumbleStart will return 2025-01-01 which will become 2025-01-01 00:00:00 in UTC
			//   this is wrong, as in CET this is 2024-12-31 23:00:00
			//  if we are casting to DateTime, then:
			// 	 tumbleStart will return 2025-01-01 00:00:00 in Europe/Budapest

			// Other queries are not affected by this, as for anything < Month, the result is always a DateTime (most probably due to
			// DST changes).
			fmt.Sprintf("toDateTime(tumbleStart(%s, toIntervalMonth(1), '%s'), '%s') AS windowstart", timeExpr, tz, tz),
			fmt.Sprintf("toDateTime(tumbleEnd(%s, toIntervalMonth(1), '%s'), '%s') AS windowend", timeExpr, tz, tz),
		}, nil

	default:
		return nil, models.NewGenericValidationError(
			fmt.Errorf("invalid window size type: %s", windowSize),
		)
	}
}

// rawStringValueExpr extracts the meter's value property from the event JSON payload as a
// string, mapping an explicit JSON null (which JSON_VALUE yields as the literal 'null') to
// SQL NULL so aggregations skip it. UNIQUE_COUNT aggregates over exactly this expression:
// distinctness is defined on the raw string representation of the property, never on a
// numeric conversion of it.
func rawStringValueExpr(dataColumn, valueProperty string) string {
	return fmt.Sprintf("nullIf(JSON_VALUE(%s, '%s'), 'null')", dataColumn, escapeJSONPathLiteral(valueProperty))
}

// numericValueExpr extracts the meter's value property from the event JSON payload as a
// number. The decimal and float legs intentionally produce different SQL types
// (Nullable(Decimal128(19)) vs Nullable(Float64)) with no UNION supertype; the meter cache
// stores decimals only, which is why the cache read gate requires EnableDecimalPrecision.
func numericValueExpr(dataColumn, valueProperty string, enableDecimalPrecision bool) string {
	if enableDecimalPrecision {
		return fmt.Sprintf("toDecimal128OrNull(%s, 19)", rawStringValueExpr(dataColumn, valueProperty))
	}

	// JSON_VALUE returns an empty string if the JSON Path is not found. With toFloat64OrNull we convert it to NULL so the aggregation function can handle it properly.
	return fmt.Sprintf("ifNotFinite(toFloat64OrNull(JSON_VALUE(%s, '%s')), null)", dataColumn, escapeJSONPathLiteral(valueProperty))
}

// valueExprPlain returns the aggregated value SELECT expression (aliased AS value) exactly
// as the live meter query emits it. The cached read path re-emits this shape on its outer
// query so cached and live results stay byte-equal.
func valueExprPlain(m meterpkg.Meter, dataColumn, timeColumn string, enableDecimalPrecision bool) (string, error) {
	sqlAggregation := ""

	switch m.Aggregation {
	case meterpkg.MeterAggregationSum:
		sqlAggregation = "sum"
	case meterpkg.MeterAggregationAvg:
		sqlAggregation = "avg"
	case meterpkg.MeterAggregationMin:
		sqlAggregation = "min"
	case meterpkg.MeterAggregationMax:
		sqlAggregation = "max"
	case meterpkg.MeterAggregationUniqueCount:
		// Use the uniqExact function if you absolutely need an exact result.
		// See: https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/uniqexact
		sqlAggregation = "uniqExact"
	case meterpkg.MeterAggregationCount:
		sqlAggregation = "count"
	case meterpkg.MeterAggregationLatest:
		sqlAggregation = "argMax"
	default:
		return "", models.NewGenericValidationError(
			fmt.Errorf("invalid aggregation type: %s", m.Aggregation),
		)
	}

	if err := requireValueProperty(m); err != nil {
		return "", err
	}

	switch m.Aggregation {
	case meterpkg.MeterAggregationCount:
		return fmt.Sprintf("%s(*) AS value", sqlAggregation), nil
	case meterpkg.MeterAggregationUniqueCount:
		return fmt.Sprintf("%s(%s) AS value", sqlAggregation, rawStringValueExpr(dataColumn, *m.ValueProperty)), nil
	case meterpkg.MeterAggregationLatest:
		return fmt.Sprintf("%s(%s, %s) AS value", sqlAggregation, numericValueExpr(dataColumn, *m.ValueProperty, enableDecimalPrecision), timeColumn), nil
	default:
		return fmt.Sprintf("%s(%s) AS value", sqlAggregation, numericValueExpr(dataColumn, *m.ValueProperty, enableDecimalPrecision)), nil
	}
}

// valueExprsCombine returns the SELECT expressions that persist a meter's aggregation into
// the om_meter_cache combine columns. Unlike the plain form, these shapes must compose
// under a second aggregation pass: cached buckets are re-windowed against each other and
// combined with an always-live tail leg, so a column holding an already-collapsed result
// would be wrong to combine.
//
//   - SUM/MIN/MAX/COUNT re-aggregate with themselves (sum of sums, min of mins, ...).
//   - AVG is stored as the sum + non-null value count pair because an average of averages
//     is wrong; the reader computes sum(sum_value) / sum(value_count) at the end.
//   - UNIQUE_COUNT stores uniqExactState: distinct counts of two buckets cannot be summed
//     (shared values would double count), the states must be merged for an exact result.
//
// LATEST is not a valid input here: it is excluded from the cache entirely (see
// meterCacheStaticReject) because it only ever needs the single newest value in the
// queried window, so there is no re-aggregation of settled history for a cached combine
// form to save.
//
// The cache stores decimals only (the cache read gate requires EnableDecimalPrecision), so
// numeric expressions always use the Decimal128 leg.
//
// timeColumn is unused now that LATEST (the only combine-form case that needed event time)
// is excluded; the parameter is kept to match valueExprPlain's signature since both
// builders are called side by side from the same sites for a given aggregation.
func valueExprsCombine(m meterpkg.Meter, dataColumn, timeColumn string) ([]string, error) {
	switch m.Aggregation {
	case meterpkg.MeterAggregationSum,
		meterpkg.MeterAggregationAvg,
		meterpkg.MeterAggregationMin,
		meterpkg.MeterAggregationMax,
		meterpkg.MeterAggregationUniqueCount,
		meterpkg.MeterAggregationCount:
	default:
		return nil, models.NewGenericValidationError(
			fmt.Errorf("invalid aggregation type: %s", m.Aggregation),
		)
	}

	if err := requireValueProperty(m); err != nil {
		return nil, err
	}

	switch m.Aggregation {
	case meterpkg.MeterAggregationSum:
		return []string{fmt.Sprintf("sum(%s) AS sum_value", numericValueExpr(dataColumn, *m.ValueProperty, true))}, nil
	case meterpkg.MeterAggregationAvg:
		valueExpr := numericValueExpr(dataColumn, *m.ValueProperty, true)

		return []string{
			fmt.Sprintf("sum(%s) AS sum_value", valueExpr),
			// count(expr) counts non-null values only: events whose value property is
			// missing or a JSON null must not inflate the AVG denominator.
			fmt.Sprintf("count(%s) AS value_count", valueExpr),
		}, nil
	case meterpkg.MeterAggregationMin:
		return []string{fmt.Sprintf("min(%s) AS min_value", numericValueExpr(dataColumn, *m.ValueProperty, true))}, nil
	case meterpkg.MeterAggregationMax:
		return []string{fmt.Sprintf("max(%s) AS max_value", numericValueExpr(dataColumn, *m.ValueProperty, true))}, nil
	case meterpkg.MeterAggregationUniqueCount:
		return []string{fmt.Sprintf("uniqExactState(%s) AS uniq_state", rawStringValueExpr(dataColumn, *m.ValueProperty))}, nil
	case meterpkg.MeterAggregationCount:
		return []string{"count(*) AS count_value"}, nil
	default:
		// Unreachable: the first switch above already rejected every aggregation not
		// handled by one of the cases here.
		return nil, models.NewGenericValidationError(
			fmt.Errorf("invalid aggregation type: %s", m.Aggregation),
		)
	}
}

// requireValueProperty guards the *m.ValueProperty dereferences in the value expression
// builders. Meter validation guarantees a value property for every non-COUNT aggregation,
// but SQL generation must degrade to an error rather than panic when handed a meter that
// bypassed validation (e.g. loaded from a store that predates the rule).
func requireValueProperty(m meterpkg.Meter) error {
	if m.Aggregation != meterpkg.MeterAggregationCount && m.ValueProperty == nil {
		return models.NewGenericValidationError(
			fmt.Errorf("meter value property is required for %s aggregation", m.Aggregation),
		)
	}

	return nil
}

// groupByJSONExpr returns the expression extracting a group-by dimension from the event
// JSON payload at jsonPath (unescaped; escaping happens here). The live query uses it for
// both SELECT dimensions and FilterGroupBy predicates, and the cache MV uses it for the
// group_by array, so cached dimension values match live ones byte-for-byte.
func groupByJSONExpr(dataColumn, jsonPath string) string {
	return fmt.Sprintf("JSON_VALUE(%s, '%s')", dataColumn, escapeJSONPathLiteral(jsonPath))
}

// groupBySelectExprs returns the SELECT expressions and GROUP BY columns for the requested
// group-by keys as the live meter query emits them.
//
// subject and customer_id are top-level event columns, not JSON dimensions: subject is
// selected directly, while customer_id is only added to GROUP BY because its SELECT
// expression (a subject-to-customer map lookup) is attached separately by
// selectCustomerIdColumn. Every other key is extracted from the JSON payload at the
// meter's configured path and aliased to the group-by key.
func groupBySelectExprs(groupBy []string, meterGroupBy map[string]string, subjectColumn, dataColumn string) ([]string, []string) {
	var selectColumns, groupByColumns []string

	for _, groupByKey := range groupBy {
		// Subject is a special case as it's a top level column
		if groupByKey == "subject" {
			selectColumns = append(selectColumns, subjectColumn)
			groupByColumns = append(groupByColumns, "subject")
			continue
		}

		// Customer ID is a special case as it's a top level column
		if groupByKey == "customer_id" {
			groupByColumns = append(groupByColumns, "customer_id")
			continue
		}

		// Group by columns need to be parsed from the JSON data
		groupByColumn := sqlbuilder.Escape(groupByKey)

		selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", groupByJSONExpr(dataColumn, meterGroupBy[groupByKey]), groupByColumn))
		groupByColumns = append(groupByColumns, groupByColumn)
	}

	return selectColumns, groupByColumns
}

// reservedMeterSQLAliases lists the exact identifiers the generated meter SQL claims for
// itself: the om_events source columns plus the om_meter_cache columns and query output
// aliases. Meter group-by keys become SELECT aliases in generated SQL, and ClickHouse
// resolves SELECT aliases inside WHERE — a group-by key shadowing a source column would
// silently neutralize generated filters (an alias named namespace, for example, would
// defeat tenant isolation without any error).
var reservedMeterSQLAliases = map[string]struct{}{
	// om_events columns
	"namespace":    {},
	"id":           {},
	"type":         {},
	"subject":      {},
	"source":       {},
	"time":         {},
	"data":         {},
	"ingested_at":  {},
	"stored_at":    {},
	"store_row_id": {},
	// om_meter_cache columns and meter query output aliases
	"meter_key":   {},
	"meter_hash":  {},
	"windowstart": {},
	"windowend":   {},
	"group_by":    {},
	"created_at":  {},
	"value":       {},
	// value_count is the one om_meter_cache combine column not covered by the suffix
	// families below (it ends in _count)
	"value_count": {},
	// aliases the cached read path's leg subqueries claim: the grain-bucket column and
	// the AVG newest-wins pick not covered by the suffix families; a group-by key with
	// one of these names would shadow the outer re-window / combine expressions (the
	// remaining picked_* aliases end in _value or _state and are covered below)
	"windowstart_bucket": {},
	"picked_value_count": {},
}

// reservedAliasCheck rejects meter group-by keys (the meter's configured JSON dimensions,
// not a query's group-by selection) that would collide with identifiers the generated
// cache SQL depends on. Beyond the exact reserved names, the *_value and *_state suffix
// families are reserved because they are the om_meter_cache combine column families.
//
// Callers must treat a non-nil error as "this meter cannot be cached" — skip the meter and
// read it live — never as a hard failure: the live query path tolerates these keys today
// and existing meters must keep working when the cache is enabled.
func reservedAliasCheck(groupByKeys []string) error {
	var offending []string

	for _, key := range groupByKeys {
		_, reserved := reservedMeterSQLAliases[key]
		if reserved || strings.HasSuffix(key, "_value") || strings.HasSuffix(key, "_state") {
			offending = append(offending, key)
		}
	}

	if len(offending) == 0 {
		return nil
	}

	// Sorted so the error is deterministic when keys come from map iteration
	slices.Sort(offending)

	return fmt.Errorf("meter group by keys collide with reserved SQL aliases: %s", strings.Join(offending, ", "))
}
