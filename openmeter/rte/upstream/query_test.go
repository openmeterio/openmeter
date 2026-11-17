package upstream

import (
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/rte"
)

func TestIngestQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	query, args := (ingestQuery{
		namespace:  "ns",
		subject:    "subject",
		meterSlugs: []string{"requests"},
		eventTypes: []string{"api.request"},
		timestamps: []time.Time{now.Add(-time.Hour)},
		now:        now,
	}).SQL()

	expectedSQL := `
WITH "customers_by_subject" AS (SELECT "c"."id"
    FROM "customers" AS "c"
    WHERE "c"."namespace" = $1
    AND ("c"."deleted_at" IS NULL
        OR "c"."deleted_at" > $2)
    AND ("c"."key" = $3
        OR EXISTS (SELECT "cs"."id"
            FROM "customer_subjects" AS "cs"
            WHERE "cs"."namespace" = "c"."namespace"
            AND "cs"."customer_id" = "c"."id"
            AND "cs"."subject_key" = $4
            AND "cs"."deleted_at" IS NULL))),
"features_by_meter" AS (SELECT "f"."id", "f"."key"
    FROM "features" AS "f"
    JOIN "meters" AS "m"
    ON "m"."namespace" = "f"."namespace"
    AND "m"."id" = "f"."meter_id"
    WHERE "f"."namespace" = $5
    AND ("m"."key" = ANY($6::text[])
        OR "m"."event_type" = ANY($7::text[])))
SELECT "d"."namespace", "d"."kind", "d"."key"
FROM "rte_demands" AS "d"
WHERE ("d"."namespace" = $8
    AND "d"."deleted_at" IS NULL
    AND ("d"."push_enabled"
        OR "d"."pull_until" > $9))
AND (("d"."kind" = $10
        AND EXISTS (SELECT "e"."id"
            FROM "entitlements" AS "e"
            JOIN "customers_by_subject" AS "cs"
            ON "cs"."id" = "e"."customer_id"
            JOIN "features_by_meter" AS "f"
            ON "f"."id" = "e"."feature_id"
            WHERE "e"."namespace" = "d"."namespace"
            AND "e"."id" = "d"."owner_id"
            AND "e"."entitlement_type" = $11
            AND EXISTS (SELECT "t"."at"
                FROM unnest($12::timestamptz[]) AS "t"("at")
                WHERE "t"."at" >= COALESCE("e"."measure_usage_from", "e"."active_from", "e"."created_at")
                AND ("e"."deleted_at" IS NULL
                    OR "t"."at" < "e"."deleted_at"))))
    OR ("d"."kind" = $13
        AND EXISTS (SELECT "c"."id"
            FROM "charge_usage_based" AS "c"
            JOIN "customers_by_subject" AS "cs"
            ON "cs"."id" = "c"."customer_id"
            JOIN "features_by_meter" AS "f"
            ON "f"."id" = "c"."feature_id"
            OR ("c"."feature_id" IS NULL
                AND "f"."key" = "c"."feature_key")
            LEFT JOIN "charge_usage_based_overrides" AS "o"
            ON "o"."namespace" = "c"."namespace"
            AND "o"."charge_id" = "c"."id"
            WHERE "c"."namespace" = "d"."namespace"
            AND "c"."customer_id" = "d"."owner_id"
            AND "c"."deleted_at" IS NULL
            AND "c"."status" NOT IN ($14, $15)
            AND "c"."settlement_mode" IN ($16, $17)
            AND COALESCE("o"."service_period_from", "c"."service_period_from") <= $18
            AND EXISTS (SELECT "t"."at"
                FROM unnest($19::timestamptz[]) AS "t"("at")
                WHERE "t"."at" >= COALESCE("o"."service_period_from", "c"."service_period_from")
                AND "t"."at" < COALESCE("o"."service_period_to", "c"."service_period_to"))
            AND ((("c"."currency" IS NOT NULL
                        AND "d"."key"->'currency'->>'code' = "c"."currency"
                        AND (NOT ("d"."key"->'currency'->>'custom_currency_id' IS NOT NULL)))
                    OR ("c"."custom_currency_id" IS NOT NULL
                        AND "d"."key"->'currency'->>'custom_currency_id' = "c"."custom_currency_id"))
                AND ((NOT ("d"."key"->'featureFilter' IS NOT NULL))
                    OR ("d"."key"->'featureFilter')::jsonb = 'null'::jsonb
                    OR "d"."key"->'featureFilter' @> jsonb_build_array("c"."feature_key"))))))
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{
		"ns", now, "subject", "subject", "ns",
		pq.Array([]string{"requests"}), pq.Array([]string{"api.request"}),
		"ns", now, rte.CalculationKindMeteredEntitlement, entitlement.EntitlementTypeMetered,
		pq.Array([]string{"2026-01-01T23:00:00Z"}), rte.CalculationKindCreditBalance,
		usagebased.StatusFinal, usagebased.StatusDeleted,
		productcatalog.CreditOnlySettlementMode, productcatalog.CreditThenInvoiceSettlementMode,
		now, pq.Array([]string{"2026-01-01T23:00:00Z"}),
	}, args)
}

func TestLedgerQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	query, args := (ledgerQuery{namespace: "ns", groupID: "group", now: now}).SQL()

	expectedSQL := `
SELECT "d"."namespace", "d"."kind", "d"."key"
FROM "rte_demands" AS "d"
WHERE ("d"."namespace" = $1
    AND "d"."deleted_at" IS NULL
    AND ("d"."push_enabled"
        OR "d"."pull_until" > $2))
AND ("d"."kind" = $3
    AND EXISTS (SELECT "t"."id"
        FROM "ledger_transactions" AS "t"
        JOIN "ledger_entries" AS "e"
        ON "e"."namespace" = "t"."namespace"
        AND "e"."transaction_id" = "t"."id"
        JOIN "ledger_sub_accounts" AS "s"
        ON "s"."namespace" = "e"."namespace"
        AND "s"."id" = "e"."sub_account_id"
        JOIN "ledger_customer_accounts" AS "a"
        ON "a"."namespace" = "s"."namespace"
        AND "a"."account_id" = "s"."account_id"
        JOIN "ledger_sub_account_routes" AS "r"
        ON "r"."namespace" = "s"."namespace"
        AND "r"."id" = "s"."route_id"
        WHERE "t"."namespace" = "d"."namespace"
        AND "t"."group_id" = $4
        AND "a"."customer_id" = "d"."owner_id"
        AND (((NOT ("d"."key"->'currency'->>'custom_currency_id' IS NOT NULL))
                AND "d"."key"->'currency'->>'code' = "r"."currency")
            OR ("d"."key"->'currency'->>'custom_currency_id' IS NOT NULL
                AND "d"."key"->'currency'->>'custom_currency_id' = split_part("r"."currency", $5, $6)))))
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{
		"ns", now, rte.CalculationKindCreditBalance, "group", "|", 4,
	}, args)
}

func TestCustomerQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	query, args := (customerQuery{namespace: "ns", customerID: "customer", now: now}).SQL()

	expectedSQL := `
SELECT "d"."namespace", "d"."kind", "d"."key"
FROM "rte_demands" AS "d"
WHERE ("d"."namespace" = $1
    AND "d"."deleted_at" IS NULL
    AND ("d"."push_enabled"
        OR "d"."pull_until" > $2))
AND (("d"."kind" = $3
        AND "d"."owner_id" = $4)
    OR ("d"."kind" = $5
        AND EXISTS (SELECT "e"."id"
            FROM "entitlements" AS "e"
            WHERE "e"."namespace" = "d"."namespace"
            AND "e"."id" = "d"."owner_id"
            AND "e"."customer_id" = $6)))
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{
		"ns", now, rte.CalculationKindCreditBalance, "customer", rte.CalculationKindMeteredEntitlement, "customer",
	}, args)
}

func TestFeatureQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	query, args := (featureQuery{namespace: "ns", featureID: "feature", now: now}).SQL()

	expectedSQL := `
SELECT "d"."namespace", "d"."kind", "d"."key"
FROM "rte_demands" AS "d"
WHERE ("d"."namespace" = $1
    AND "d"."deleted_at" IS NULL
    AND ("d"."push_enabled"
        OR "d"."pull_until" > $2))
AND (("d"."kind" = $3
        AND EXISTS (SELECT "e"."id"
            FROM "entitlements" AS "e"
            WHERE "e"."namespace" = "d"."namespace"
            AND "e"."id" = "d"."owner_id"
            AND "e"."feature_id" = $4))
    OR ("d"."kind" = $5
        AND EXISTS (SELECT "c"."id"
            FROM "charge_usage_based" AS "c"
            JOIN "features" AS "f"
            ON "f"."namespace" = "c"."namespace"
            AND ("f"."id" = "c"."feature_id"
                OR ("c"."feature_id" IS NULL
                    AND "f"."key" = "c"."feature_key"))
            WHERE "c"."namespace" = "d"."namespace"
            AND "c"."customer_id" = "d"."owner_id"
            AND "f"."id" = $6
            AND ((("c"."currency" IS NOT NULL
                        AND "d"."key"->'currency'->>'code' = "c"."currency"
                        AND (NOT ("d"."key"->'currency'->>'custom_currency_id' IS NOT NULL)))
                    OR ("c"."custom_currency_id" IS NOT NULL
                        AND "d"."key"->'currency'->>'custom_currency_id' = "c"."custom_currency_id"))
                AND ((NOT ("d"."key"->'featureFilter' IS NOT NULL))
                    OR ("d"."key"->'featureFilter')::jsonb = 'null'::jsonb
                    OR "d"."key"->'featureFilter' @> jsonb_build_array("c"."feature_key"))))))
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{
		"ns", now, rte.CalculationKindMeteredEntitlement, "feature", rte.CalculationKindCreditBalance, "feature",
	}, args)
}

func TestMeterQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	query, args := (meterQuery{namespace: "ns", meterID: "meter", now: now}).SQL()

	expectedSQL := `
WITH "affected_features" AS (SELECT "features"."id", "features"."key"
    FROM "features"
    WHERE "features"."namespace" = $1
    AND "features"."meter_id" = $2)
SELECT "d"."namespace", "d"."kind", "d"."key"
FROM "rte_demands" AS "d"
WHERE ("d"."namespace" = $3
    AND "d"."deleted_at" IS NULL
    AND ("d"."push_enabled"
        OR "d"."pull_until" > $4))
AND (("d"."kind" = $5
        AND EXISTS (SELECT "e"."id"
            FROM "entitlements" AS "e"
            JOIN "affected_features" AS "f"
            ON "f"."id" = "e"."feature_id"
            WHERE "e"."namespace" = "d"."namespace"
            AND "e"."id" = "d"."owner_id"))
    OR ("d"."kind" = $6
        AND EXISTS (SELECT "c"."id"
            FROM "charge_usage_based" AS "c"
            JOIN "affected_features" AS "f"
            ON "f"."id" = "c"."feature_id"
            OR ("c"."feature_id" IS NULL
                AND "f"."key" = "c"."feature_key")
            WHERE "c"."namespace" = "d"."namespace"
            AND "c"."customer_id" = "d"."owner_id"
            AND ((("c"."currency" IS NOT NULL
                        AND "d"."key"->'currency'->>'code' = "c"."currency"
                        AND (NOT ("d"."key"->'currency'->>'custom_currency_id' IS NOT NULL)))
                    OR ("c"."custom_currency_id" IS NOT NULL
                        AND "d"."key"->'currency'->>'custom_currency_id' = "c"."custom_currency_id"))
                AND ((NOT ("d"."key"->'featureFilter' IS NOT NULL))
                    OR ("d"."key"->'featureFilter')::jsonb = 'null'::jsonb
                    OR "d"."key"->'featureFilter' @> jsonb_build_array("c"."feature_key"))))))
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{
		"ns", "meter", "ns", now, rte.CalculationKindMeteredEntitlement, rte.CalculationKindCreditBalance,
	}, args)
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
