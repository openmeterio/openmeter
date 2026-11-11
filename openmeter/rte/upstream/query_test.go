package upstream

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIngestQuery(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	query, args := (ingestQuery{namespace: "ns", subject: "subject", meters: []string{"requests"}, timestamps: []time.Time{now.Add(-time.Hour)}, now: now}).SQL()
	require.Equal(t, `WITH customers_by_subject AS (
 SELECT c.id FROM customers c WHERE c.namespace = $1
 AND (c.deleted_at IS NULL OR c.deleted_at > $2)
 AND (c.key = $3 OR EXISTS (
   SELECT 1 FROM customer_subjects cs WHERE cs.namespace = c.namespace
   AND cs.customer_id = c.id AND cs.subject_key = $3 AND cs.deleted_at IS NULL))
), features_by_meter AS (
 SELECT f.id, f.key FROM features f JOIN meters m
 ON m.namespace = f.namespace AND m.id = f.meter_id
 WHERE f.namespace = $1 AND m.key = ANY($4::text[])
)
SELECT d.namespace, d.kind, d.key FROM rte_demands d
WHERE d.namespace = $1 AND d.deleted_at IS NULL AND (d.push_enabled OR d.pull_until > $2) AND (
 (d.kind = 'metered_entitlement' AND EXISTS (
   SELECT 1 FROM entitlements e JOIN customers_by_subject cs ON cs.id = e.customer_id
   JOIN features_by_meter f ON f.id = e.feature_id
   WHERE e.namespace = d.namespace AND e.id = d.owner_id AND e.entitlement_type = 'metered'
   AND EXISTS (SELECT 1 FROM unnest($5::timestamptz[]) AS t(at)
     WHERE t.at >= COALESCE(e.measure_usage_from, e.active_from, e.created_at)
     AND (e.active_to IS NULL OR t.at < e.active_to)
     AND (e.deleted_at IS NULL OR t.at < e.deleted_at)))) OR
 (d.kind = 'credit_balance' AND EXISTS (
   SELECT 1 FROM charge_usage_based c
   JOIN customers_by_subject cs ON cs.id = c.customer_id
   JOIN features_by_meter f ON f.id = c.feature_id OR (c.feature_id IS NULL AND f.key = c.feature_key)
   LEFT JOIN charge_usage_based_overrides o ON o.namespace = c.namespace AND o.charge_id = c.id
   WHERE c.namespace = d.namespace AND c.customer_id = d.owner_id
   AND c.deleted_at IS NULL AND c.status NOT IN ('final', 'deleted')
   AND c.settlement_mode IN ('credit_only', 'credit_then_invoice')
   AND COALESCE(o.service_period_from, c.service_period_from) <= $2
   AND EXISTS (SELECT 1 FROM unnest($5::timestamptz[]) AS t(at)
     WHERE t.at >= COALESCE(o.service_period_from, c.service_period_from)
     AND t.at < COALESCE(o.service_period_to, c.service_period_to))
   AND (
 (c.currency IS NOT NULL AND d.key->'currency'->>'code' = c.currency
  AND d.key->'currency'->>'custom_currency_id' IS NULL) OR
 (c.custom_currency_id IS NOT NULL AND d.key->'currency'->>'custom_currency_id' = c.custom_currency_id)
) AND (
 d.key->'featureFilter' IS NULL OR d.key->'featureFilter' = 'null'::jsonb
 OR d.key->'featureFilter' @> jsonb_build_array(c.feature_key)
))))`, query)
	require.Equal(t, []any{"ns", now, "subject"}, args[:3])
	meters, err := args[3].(driver.Valuer).Value()
	require.NoError(t, err)
	require.Equal(t, `{"requests"}`, meters)
	timestamps, err := args[4].(driver.Valuer).Value()
	require.NoError(t, err)
	require.Equal(t, `{"2026-01-01T23:00:00Z"}`, timestamps)
}

func TestSourceFactories(t *testing.T) {
	for name := range eventFactories {
		first, ok := NewEvent(name)
		require.True(t, ok)
		second, ok := NewEvent(name)
		require.True(t, ok)
		require.NotSame(t, first, second)
		require.Equal(t, name, first.EventName())
	}

	event, ok := NewEvent("unrelated")
	require.False(t, ok)
	require.Nil(t, event)
}
