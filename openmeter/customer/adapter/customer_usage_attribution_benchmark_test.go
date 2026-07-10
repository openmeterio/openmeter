package adapter

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

const (
	defaultUsageAttributionBenchmarkCustomers = 100_000
	usageAttributionBenchmarkSizeEnv          = "CUSTOMER_USAGE_ATTRIBUTION_BENCH_CUSTOMERS"
)

// BenchmarkCustomerUsageAttributionLookup compares the cross-table OR with the
// UNION ALL candidate-ID lookup. Seeding is outside the timed sections. Set
// CUSTOMER_USAGE_ATTRIBUTION_BENCH_CUSTOMERS to change the namespace size from
// the default 100,000 customers.
func BenchmarkCustomerUsageAttributionLookup(b *testing.B) {
	b.StopTimer()

	db := testutils.InitPostgresDB(b)
	b.Cleanup(func() { db.Close(b) })

	client := db.EntDriver.Client()
	if err := client.Schema.Create(b.Context()); err != nil {
		b.Fatalf("create database schema: %v", err)
	}

	customerCount := usageAttributionBenchmarkCustomerCount(b)
	namespace := "customer-usage-attribution-benchmark"
	seedUsageAttributionBenchmark(b, client, namespace, customerCount)

	targetCustomerID := fmt.Sprintf("%026d", customerCount)
	now := time.Now().UTC()

	lookups := []struct {
		name string
		key  string
	}{
		{
			name: "customer_key",
			key:  fmt.Sprintf("customer-%d", customerCount),
		},
		{
			name: "subject_key",
			key:  fmt.Sprintf("subject-%d", customerCount),
		},
	}

	for _, lookup := range lookups {
		b.Run(lookup.name, func(b *testing.B) {
			variants := []struct {
				name  string
				query func(context.Context) (string, error)
			}{
				{
					name: "cross_table_or",
					query: func(ctx context.Context) (string, error) {
						return client.Customer.Query().
							Where(
								customerdb.Namespace(namespace),
								customerdb.Or(
									customerdb.HasSubjectsWith(
										customersubjectsdb.SubjectKey(lookup.key),
										customersubjectsdb.Or(
											customersubjectsdb.DeletedAtIsNil(),
											customersubjectsdb.DeletedAtGT(now),
										),
									),
									customerdb.Key(lookup.key),
								),
								customerdb.DeletedAtIsNil(),
							).
							FirstID(ctx)
					},
				},
				{
					name: "union_all",
					query: func(ctx context.Context) (string, error) {
						return client.Customer.Query().
							Where(
								customerdb.Namespace(namespace),
								customerdb.DeletedAtIsNil(),
								customerMatchesUsageAttributionKey(namespace, lookup.key, now),
							).
							FirstID(ctx)
					},
				},
			}

			for _, variant := range variants {
				b.Run(variant.name, func(b *testing.B) {
					customerID, err := variant.query(b.Context())
					if err != nil {
						b.Fatalf("warm up lookup: %v", err)
					}
					if customerID != targetCustomerID {
						b.Fatalf("warm up lookup returned customer %q, expected %q", customerID, targetCustomerID)
					}

					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						customerID, err = variant.query(b.Context())
						if err != nil {
							b.Fatalf("lookup customer: %v", err)
						}
					}
					b.StopTimer()
					b.ReportMetric(float64(customerCount), "customers")

					if customerID != targetCustomerID {
						b.Fatalf("lookup returned customer %q, expected %q", customerID, targetCustomerID)
					}
				})
			}
		})
	}
}

func usageAttributionBenchmarkCustomerCount(b *testing.B) int {
	b.Helper()

	value := os.Getenv(usageAttributionBenchmarkSizeEnv)
	if value == "" {
		return defaultUsageAttributionBenchmarkCustomers
	}

	count, err := strconv.Atoi(value)
	if err != nil || count < 1 {
		b.Fatalf("%s must be a positive integer, got %q", usageAttributionBenchmarkSizeEnv, value)
	}

	return count
}

func seedUsageAttributionBenchmark(b *testing.B, client *entdb.Client, namespace string, customerCount int) {
	b.Helper()

	_, err := client.ExecContext(b.Context(), `
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
		b.Fatalf("seed customers: %v", err)
	}

	_, err = client.ExecContext(b.Context(), `
		INSERT INTO customer_subjects (namespace, subject_key, created_at, customer_id)
		SELECT
			$1,
			'subject-' || customer_number,
			now(),
			lpad(customer_number::text, 26, '0')
		FROM generate_series(1, $2) AS customer_number
	`, namespace, customerCount)
	if err != nil {
		b.Fatalf("seed customer subjects: %v", err)
	}

	if _, err = client.ExecContext(b.Context(), "ANALYZE customers"); err != nil {
		b.Fatalf("analyze customers: %v", err)
	}
	if _, err = client.ExecContext(b.Context(), "ANALYZE customer_subjects"); err != nil {
		b.Fatalf("analyze customer subjects: %v", err)
	}
}
