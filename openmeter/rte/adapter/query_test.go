package adapter

import (
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRenewPullQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	at := pullRenewalTime{now: func() time.Time { return now }}
	until := pullRenewalTime{now: at.now, offset: time.Minute}
	key := `{"entitlementId":{"namespace":"ns","id":"ent"}}`

	query, args := (renewPullQuery{
		identity: rte.TargetIdentity{Namespace: "ns", KeyHash: "hash"},
		kind:     rte.CalculationKindMeteredEntitlement,
		key:      key, at: at, until: until,
	}).SQL()

	expectedSQL := `
UPDATE "rte_demands"
SET "pull_until" = GREATEST("pull_until", $1), "updated_at" = $2
WHERE "id" = (SELECT "id" FROM "rte_demands"
    WHERE "namespace" = $3 AND "kind" = $4 AND "key_hash" = $5 AND "key" = $6
        AND "deleted_at" IS NULL AND NOT "push_enabled" AND "pull_until" > $7
    FOR UPDATE SKIP LOCKED)
RETURNING "completed_version" = "dirty_version" AND ("next_refresh_at" IS NULL OR "next_refresh_at" > $8)`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	for i, arg := range args {
		if delayed, ok := arg.(driver.Valuer); ok {
			var err error
			args[i], err = delayed.Value()
			require.NoError(t, err)
		}
	}

	require.Equal(t, []any{now.Add(time.Minute), now, "ns", rte.CalculationKindMeteredEntitlement, "hash", key, now, now}, args)
}

func TestInvalidateDemandQuery_SQLTemplate(t *testing.T) {
	now := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	target, err := rte.NewCalculationTarget(meteredentitlement.Key{EntitlementID: models.NamespacedID{Namespace: "ns", ID: "ent"}})
	require.NoError(t, err)
	identity, err := target.Identity()
	require.NoError(t, err)

	query, args, err := (invalidateDemandQuery{
		at: now,
		input: rte.InvalidateInput{
			Source: "ledger", EventID: "event",
			Selectors: []rte.DemandSelector{
				{Namespace: "ns", Kind: rte.CalculationKindCreditBalance, OwnerIDs: []string{"customer"}},
				{Namespace: "ns", Kind: rte.CalculationKindMeteredEntitlement, Targets: []rte.CalculationTarget{target}},
			},
		},
	}).SQL()
	require.NoError(t, err)
	expectedSQL := `
UPDATE "rte_demands"
SET "dirty_version" = COALESCE("rte_demands"."dirty_version", 0) + $1, "updated_at" = $2,
    "next_attempt_at" = CASE WHEN "rte_demands"."attempts" = $3 AND "rte_demands"."dirty_version" = "rte_demands"."completed_version" THEN $4 ELSE "rte_demands"."next_attempt_at" END
WHERE (("rte_demands"."deleted_at" IS NULL AND ("rte_demands"."push_enabled" OR "rte_demands"."pull_until" > $5))
    AND (("namespace" = $6 AND "kind" = $7 AND "owner_id" = ANY($8))
        OR ("namespace" = $9 AND "kind" = $10 AND "key_hash" = ANY($11))))
    AND NOT EXISTS (SELECT "id" FROM "rte_invalidations" WHERE "source" = $12 AND "event_id" = $13)`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{
		1, now, 0, now, now,
		"ns", rte.CalculationKindCreditBalance, pq.Array([]string{"customer"}),
		"ns", rte.CalculationKindMeteredEntitlement, pq.Array([]string{identity.KeyHash}),
		"ledger", "event",
	}, args)
}
