package upstream

import (
	"time"

	"github.com/lib/pq"
)

const activeDemand = `d.deleted_at IS NULL AND (d.push_enabled OR d.pull_until > $2)`

type ledgerQuery struct {
	namespace, groupID string
	now                time.Time
}

func (q ledgerQuery) SQL() (string, []any) {
	return `SELECT d.namespace, d.kind, d.key FROM rte_demands d
WHERE d.namespace = $1 AND ` + activeDemand + ` AND d.kind = 'credit_balance' AND EXISTS (
 SELECT 1 FROM ledger_transactions t
 JOIN ledger_entries e ON e.namespace = t.namespace AND e.transaction_id = t.id
 JOIN ledger_sub_accounts s ON s.namespace = e.namespace AND s.id = e.sub_account_id
 JOIN ledger_customer_accounts a ON a.namespace = s.namespace AND a.account_id = s.account_id
 JOIN ledger_sub_account_routes r ON r.namespace = s.namespace AND r.id = s.route_id
 WHERE t.namespace = d.namespace AND t.group_id = $3 AND a.customer_id = d.owner_id
 AND ((d.key->'currency'->>'custom_currency_id' IS NULL AND r.currency = d.key->'currency'->>'code')
 OR (d.key->'currency'->>'custom_currency_id' IS NOT NULL
     AND split_part(r.currency, '|', 4) = d.key->'currency'->>'custom_currency_id'))
)`, []any{q.namespace, q.now, q.groupID}
}

type customerQuery struct {
	namespace, customerID string
	now                   time.Time
}

func (q customerQuery) SQL() (string, []any) {
	return `SELECT d.namespace, d.kind, d.key FROM rte_demands d
WHERE d.namespace = $1 AND ` + activeDemand + ` AND (
 (d.kind = 'credit_balance' AND d.owner_id = $3) OR
 (d.kind = 'metered_entitlement' AND EXISTS (
   SELECT 1 FROM entitlements e WHERE e.namespace = d.namespace
   AND e.id = d.owner_id AND e.customer_id = $3)))`, []any{q.namespace, q.now, q.customerID}
}

type featureQuery struct {
	namespace, featureID string
	now                  time.Time
}

type meterQuery struct {
	namespace, meterID string
	now                time.Time
}

func (q meterQuery) SQL() (string, []any) {
	return `WITH affected_features AS (
 SELECT f.id, f.key FROM features f WHERE f.namespace = $1 AND f.meter_id = $3
)
SELECT d.namespace, d.kind, d.key FROM rte_demands d
WHERE d.namespace = $1 AND ` + activeDemand + ` AND (
 (d.kind = 'metered_entitlement' AND EXISTS (
   SELECT 1 FROM entitlements e JOIN affected_features f ON f.id = e.feature_id
   WHERE e.namespace = d.namespace AND e.id = d.owner_id)) OR
 (d.kind = 'credit_balance' AND EXISTS (
   SELECT 1 FROM charge_usage_based c JOIN affected_features f
   ON f.id = c.feature_id OR (c.feature_id IS NULL AND f.key = c.feature_key)
   WHERE c.namespace = d.namespace AND c.customer_id = d.owner_id
   AND ` + creditChargeScope + `)))`, []any{q.namespace, q.now, q.meterID}
}

func (q featureQuery) SQL() (string, []any) {
	return `SELECT d.namespace, d.kind, d.key FROM rte_demands d
WHERE d.namespace = $1 AND ` + activeDemand + ` AND (
 (d.kind = 'metered_entitlement' AND EXISTS (
   SELECT 1 FROM entitlements e WHERE e.namespace = d.namespace
   AND e.id = d.owner_id AND e.feature_id = $3)) OR
 (d.kind = 'credit_balance' AND EXISTS (
   SELECT 1 FROM charge_usage_based c JOIN features f
   ON f.namespace = c.namespace AND (f.id = c.feature_id OR (c.feature_id IS NULL AND f.key = c.feature_key))
   WHERE c.namespace = d.namespace AND c.customer_id = d.owner_id
   AND f.id = $3 AND ` + creditChargeScope + `)))`, []any{q.namespace, q.now, q.featureID}
}

// Match current customerbalance live calculation identity. A feature-bound
// charge contributes to the portfolio and that feature, not unrestricted views.
const creditChargeScope = `(
 (c.currency IS NOT NULL AND d.key->'currency'->>'code' = c.currency
  AND d.key->'currency'->>'custom_currency_id' IS NULL) OR
 (c.custom_currency_id IS NOT NULL AND d.key->'currency'->>'custom_currency_id' = c.custom_currency_id)
) AND (
 d.key->'featureFilter' IS NULL OR d.key->'featureFilter' = 'null'::jsonb
 OR d.key->'featureFilter' @> jsonb_build_array(c.feature_key)
)`

type ingestQuery struct {
	namespace, subject string
	meters             []string
	timestamps         []time.Time
	now                time.Time
}

func (q ingestQuery) SQL() (string, []any) {
	timestamps := make([]string, 0, len(q.timestamps))
	for _, timestamp := range q.timestamps {
		timestamps = append(timestamps, timestamp.UTC().Format(time.RFC3339Nano))
	}

	// Do not compare StoredAt to previous source timestamps. A late event can
	// change an earlier charge or the entitlement's resumable balance history.
	return `WITH customers_by_subject AS (
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
WHERE d.namespace = $1 AND ` + activeDemand + ` AND (
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
   AND ` + creditChargeScope + `)))`, []any{q.namespace, q.now, q.subject, pq.Array(q.meters), pq.Array(timestamps)}
}
