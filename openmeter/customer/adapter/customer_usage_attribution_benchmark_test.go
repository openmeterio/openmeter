package adapter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer/testutils/uabench"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

// BenchmarkCustomerUsageAttributionQuery compares the cross-table OR with the UNION ALL candidate-ID
// query for a single key. It measures only the adapter-level candidate query (no precedence, no
// hydration); the full service path incl. key-over-subject precedence is benchmarked separately in
// the customer service package. Seeding is outside the timed sections. Set
// CUSTOMER_USAGE_ATTRIBUTION_BENCH_CUSTOMERS to change the namespace size from the default 100,000.
func BenchmarkCustomerUsageAttributionQuery(b *testing.B) {
	b.StopTimer()

	db := testutils.InitPostgresDB(b, testutils.PostgresDBStateEntMigrated)
	b.Cleanup(func() { db.Close(b) })

	client := db.EntDriver.Client()
	customerCount := uabench.CustomerCount(b)
	namespace := "customer-usage-attribution-benchmark"
	uabench.Seed(b, client, namespace, customerCount)

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
								customersMatchUsageAttributionKeys(namespace, []string{lookup.key}, now),
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

// BenchmarkCustomersUsageAttributionBulkQuery compares the cross-table OR with the UNION ALL
// candidate-ID query for the bulk (multi-key) path, mirroring BenchmarkCustomerUsageAttributionQuery
// for the single-key path. Adapter-level query only (no precedence). Seeding is outside the timed
// sections. Set CUSTOMER_USAGE_ATTRIBUTION_BENCH_CUSTOMERS to change the namespace size.
func BenchmarkCustomersUsageAttributionBulkQuery(b *testing.B) {
	b.StopTimer()

	db := testutils.InitPostgresDB(b, testutils.PostgresDBStateEntMigrated)
	b.Cleanup(func() { db.Close(b) })

	client := db.EntDriver.Client()

	customerCount := uabench.CustomerCount(b)
	if customerCount < 2*uabench.BulkKeyCount {
		b.Fatalf("%s must be at least %d to seed a non-overlapping bulk key set, got %d",
			uabench.CustomersEnv, 2*uabench.BulkKeyCount, customerCount)
	}

	namespace := "customer-usage-attribution-bulk-benchmark"
	uabench.Seed(b, client, namespace, customerCount)

	now := time.Now().UTC()
	keys := uabench.BulkKeys(customerCount)

	variants := []struct {
		name  string
		query func(context.Context) (int, error)
	}{
		{
			name: "cross_table_or",
			query: func(ctx context.Context) (int, error) {
				return client.Customer.Query().
					Where(
						customerdb.Namespace(namespace),
						customerdb.Or(
							customerdb.HasSubjectsWith(
								customersubjectsdb.SubjectKeyIn(keys...),
								customersubjectsdb.Or(
									customersubjectsdb.DeletedAtIsNil(),
									customersubjectsdb.DeletedAtGT(now),
								),
							),
							customerdb.KeyIn(keys...),
						),
						customerdb.DeletedAtIsNil(),
					).
					Count(ctx)
			},
		},
		{
			name: "union_all",
			query: func(ctx context.Context) (int, error) {
				return client.Customer.Query().
					Where(
						customerdb.Namespace(namespace),
						customerdb.DeletedAtIsNil(),
						customersMatchUsageAttributionKeys(namespace, keys, now),
					).
					Count(ctx)
			},
		},
	}

	for _, variant := range variants {
		b.Run(variant.name, func(b *testing.B) {
			count, err := variant.query(b.Context())
			if err != nil {
				b.Fatalf("warm up lookup: %v", err)
			}
			if count != uabench.BulkKeyCount {
				b.Fatalf("warm up lookup returned %d customers, expected %d", count, uabench.BulkKeyCount)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				count, err = variant.query(b.Context())
				if err != nil {
					b.Fatalf("lookup customers: %v", err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(customerCount), "customers")
			b.ReportMetric(float64(uabench.BulkKeyCount), "keys")

			if count != uabench.BulkKeyCount {
				b.Fatalf("lookup returned %d customers, expected %d", count, uabench.BulkKeyCount)
			}
		})
	}
}
