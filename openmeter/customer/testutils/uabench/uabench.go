// Package uabench holds shared fixtures for the customer usage-attribution benchmarks. It imports
// only the generated ent client (not the customer adapter or service packages), so both the
// adapter-layer benchmark (package adapter) and the service-layer benchmark can seed from one place
// without an import cycle.
package uabench

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

const (
	// CustomersEnv overrides the seeded namespace size.
	CustomersEnv = "CUSTOMER_USAGE_ATTRIBUTION_BENCH_CUSTOMERS"

	defaultCustomers = 100_000

	// BulkKeyCount mirrors the governance QueryAccess OAS cap (@maxItems(100) on customer.keys),
	// the primary bulk consumer's realistic upper bound.
	BulkKeyCount = 100
)

// CustomerCount returns the seeded namespace size, overridable via CustomersEnv (default 100,000).
func CustomerCount(tb testing.TB) int {
	tb.Helper()

	value := os.Getenv(CustomersEnv)
	if value == "" {
		return defaultCustomers
	}

	count, err := strconv.Atoi(value)
	if err != nil || count < 1 {
		tb.Fatalf("%s must be a positive integer, got %q", CustomersEnv, value)
	}

	return count
}

// BulkKeys builds a realistic mixed bulk key set of BulkKeyCount keys: half resolve via a direct
// customer-key match, half via a subject-key match on a disjoint customer range, mirroring how
// governance resolves customer keys and subject keys through the same call. Requires
// customerCount >= 2*BulkKeyCount.
func BulkKeys(customerCount int) []string {
	half := BulkKeyCount / 2

	keys := make([]string, 0, BulkKeyCount)
	for i := 1; i <= half; i++ {
		keys = append(keys, fmt.Sprintf("customer-%d", i))
	}
	for i := half + 1; i <= 2*half; i++ {
		keys = append(keys, fmt.Sprintf("subject-%d", i))
	}

	return keys
}

// Seed bulk-inserts customerCount customers (key "customer-N") each with one subject ("subject-N")
// directly via SQL — orders of magnitude faster than the service create path — then ANALYZEs both
// tables so the planner has stats. Customer ids are the zero-padded ordinal, so callers can compute
// the target id for row N as lpad(N, 26, '0').
func Seed(tb testing.TB, client *entdb.Client, namespace string, customerCount int) {
	tb.Helper()

	ctx := tb.Context()

	_, err := client.ExecContext(ctx, `
		INSERT INTO customers (id, namespace, created_at, updated_at, name, key)
		SELECT
			lpad(customer_number::text, 26, '0'),
			$1,
			now(),
			now(),
			'customer-' || customer_number,
			'customer-' || customer_number
		FROM generate_series(1, $2) AS customer_number
	`, namespace, customerCount)
	if err != nil {
		tb.Fatalf("seed customers: %v", err)
	}

	_, err = client.ExecContext(ctx, `
		INSERT INTO customer_subjects (namespace, subject_key, created_at, customer_id)
		SELECT
			$1,
			'subject-' || customer_number,
			now(),
			lpad(customer_number::text, 26, '0')
		FROM generate_series(1, $2) AS customer_number
	`, namespace, customerCount)
	if err != nil {
		tb.Fatalf("seed customer subjects: %v", err)
	}

	if _, err = client.ExecContext(ctx, "ANALYZE customers"); err != nil {
		tb.Fatalf("analyze customers: %v", err)
	}
	if _, err = client.ExecContext(ctx, "ANALYZE customer_subjects"); err != nil {
		tb.Fatalf("analyze customer subjects: %v", err)
	}
}
