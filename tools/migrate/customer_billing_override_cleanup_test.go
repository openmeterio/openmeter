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
CREATE TEMP TABLE billing_profiles (id character(26) PRIMARY KEY);
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
    tax_code_id character(26),
    CONSTRAINT "billing_customer_overrides_tax_codes_billing_customer_overrides"
        FOREIGN KEY (tax_code_id) REFERENCES tax_codes(id) ON DELETE SET NULL,
    CONSTRAINT "billing_customer_overrides_billing_profiles_billing_customer_override"
        FOREIGN KEY (billing_profile_id) REFERENCES billing_profiles(id) ON DELETE SET NULL
);
CREATE INDEX billingcustomeroverride_tax_code_id ON billing_customer_overrides(tax_code_id);`)
	require.NoError(t, err)

	createdAt := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	deletedAt := updatedAt.Add(time.Hour)
	for _, namespace := range []string{"cleanup-a", "cleanup-b"} {
		profileID := ulid.Make().String()
		_, err = conn.ExecContext(t.Context(), "INSERT INTO billing_profiles (id) VALUES ($1)", profileID)
		require.NoError(t, err)

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
		"20261026101725_require_customer_billing_profile.up.sql",
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

	// New empty selections are rejected by the migrated database schema.
	_, err = conn.ExecContext(t.Context(), `
INSERT INTO billing_customer_overrides (id, namespace, customer_id, created_at, updated_at)
VALUES ($1, 'cleanup-invalid', $2, $3, $4)`, ulid.Make().String(), ulid.Make().String(), createdAt, updatedAt)
	require.ErrorContains(t, err, `null value in column "billing_profile_id"`)

	t.Run("rollback restores tax code relationships", func(t *testing.T) {
		// given: the cleanup has preserved assigned rows and removed the legacy columns.
		// when: the profile requirement and column cleanup are rolled back.
		for _, migration := range []string{
			"20261026101725_require_customer_billing_profile.down.sql",
			"20261026095422_remove_customer_override_fields.down.sql",
		} {
			_, err := conn.ExecContext(t.Context(), readMigration(t, migration))
			require.NoError(t, err)
		}

		// then: the historical index and validated foreign key are restored.
		var indexDefinition string
		err := conn.QueryRowContext(t.Context(), `
SELECT pg_get_indexdef('pg_temp.billingcustomeroverride_tax_code_id'::regclass)`).Scan(&indexDefinition)
		require.NoError(t, err)
		require.Contains(t, indexDefinition, "USING btree (tax_code_id)")

		var constraintDefinition string
		var constraintValidated bool
		err = conn.QueryRowContext(t.Context(), `
SELECT pg_get_constraintdef(oid), convalidated
FROM pg_constraint
WHERE conrelid = 'pg_temp.billing_customer_overrides'::regclass
  AND conname = 'billing_customer_overrides_tax_codes_billing_customer_overrides'`).Scan(&constraintDefinition, &constraintValidated)
		require.NoError(t, err)
		require.Equal(t, "FOREIGN KEY (tax_code_id) REFERENCES tax_codes(id) ON DELETE SET NULL", constraintDefinition)
		require.True(t, constraintValidated)

		// Deleting a tax code clears historical references without deleting assignments.
		taxCodeID := ulid.Make().String()
		_, err = conn.ExecContext(t.Context(), "INSERT INTO tax_codes (id) VALUES ($1)", taxCodeID)
		require.NoError(t, err)

		_, err = conn.ExecContext(t.Context(), "UPDATE billing_customer_overrides SET tax_code_id = $1", taxCodeID)
		require.NoError(t, err)

		_, err = conn.ExecContext(t.Context(), "DELETE FROM tax_codes WHERE id = $1", taxCodeID)
		require.NoError(t, err)

		var assignmentsWithoutTaxCode int
		err = conn.QueryRowContext(t.Context(), "SELECT count(*) FROM billing_customer_overrides WHERE tax_code_id IS NULL").Scan(&assignmentsWithoutTaxCode)
		require.NoError(t, err)
		require.Equal(t, 4, assignmentsWithoutTaxCode)

		_, err = conn.ExecContext(t.Context(), "UPDATE billing_customer_overrides SET tax_code_id = $1", ulid.Make().String())
		require.ErrorContains(t, err, "billing_customer_overrides_tax_codes_billing_customer_overrides")
	})
}
