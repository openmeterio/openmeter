package migrate_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestCustomerBillingOverrideCleanupPreservesAssignments(t *testing.T) {
	// given: live and deleted empty selections and assignments in multiple namespaces.
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() { testDB.Close(t) })
	conn, err := testDB.PGDriver.DB().Conn(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	_, err = conn.ExecContext(t.Context(), `
CREATE TEMP TABLE tax_codes (id character(26) PRIMARY KEY);
CREATE TEMP TABLE billing_customer_overrides (
    id character(26) PRIMARY KEY,
    namespace character varying NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    customer_id character(26) NOT NULL UNIQUE,
    billing_profile_id character(26),
    collection_alignment character varying,
    anchored_alignment_detail jsonb,
    line_collection_period character varying,
    invoice_auto_advance boolean,
    invoice_draft_period character varying,
    invoice_due_after character varying,
    invoice_collection_method character varying,
    invoice_progressive_billing boolean,
    invoice_default_tax_config jsonb,
    tax_behavior character varying,
    tax_code_id character(26) REFERENCES tax_codes(id)
);
CREATE INDEX billingcustomeroverride_tax_code_id ON billing_customer_overrides(tax_code_id);`)
	require.NoError(t, err)

	createdAt := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	deletedAt := updatedAt.Add(time.Hour)
	for _, namespace := range []string{"cleanup-a", "cleanup-b"} {
		profileID := ulid.Make().String()
		for _, selection := range []struct {
			profileID *string
			deletedAt *time.Time
		}{
			{},
			{deletedAt: &deletedAt},
			{profileID: lo.ToPtr(profileID)},
			{profileID: lo.ToPtr(profileID), deletedAt: &deletedAt},
		} {
			_, err = conn.ExecContext(t.Context(), `
INSERT INTO billing_customer_overrides (
    id, namespace, customer_id, billing_profile_id, created_at, updated_at, deleted_at,
    line_collection_period, invoice_auto_advance
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PT999H', false)`,
				ulid.Make().String(), namespace, ulid.Make().String(), selection.profileID,
				createdAt, updatedAt, selection.deletedAt)
			require.NoError(t, err)
		}
	}

	var expectedAssignments string
	err = conn.QueryRowContext(t.Context(), `
SELECT jsonb_agg(to_jsonb(assignments) ORDER BY assignments.id)::text
FROM (
    SELECT id, namespace, customer_id, billing_profile_id, created_at, updated_at, deleted_at
    FROM billing_customer_overrides WHERE billing_profile_id IS NOT NULL
) AS assignments`).Scan(&expectedAssignments)
	require.NoError(t, err)

	// when: the empty-row cleanup and generated schema migration run in order.
	for _, migration := range []string{
		"20261026095354_cleanup_empty_customer_overrides.up.sql",
		"20261026095422_remove_customer_override_fields.up.sql",
	} {
		_, err = conn.ExecContext(t.Context(), readMigration(t, migration))
		require.NoError(t, err)
	}

	// then: only assigned rows remain, preserving their identity, timestamps and deletion state.
	// Comparing complete rows also verifies that every obsolete workflow/tax column was removed.
	var actualAssignments string
	err = conn.QueryRowContext(t.Context(), `
SELECT jsonb_agg(to_jsonb(assignments) ORDER BY assignments.id)::text
FROM billing_customer_overrides AS assignments`).Scan(&actualAssignments)
	require.NoError(t, err)
	require.JSONEq(t, expectedAssignments, actualAssignments)
}
