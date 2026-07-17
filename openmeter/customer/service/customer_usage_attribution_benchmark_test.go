package customerservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/customer/testutils/uabench"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	noop "github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// newUsageAttributionServiceBenchmark builds a real customer service over a freshly seeded namespace
// (100k customers by default) and returns it with the namespace and seeded size. It constructs the
// service directly (adapter + service, no-op event publisher) rather than via the *testing.T-only
// NewTestEnv, so it is usable from a *testing.B.
func newUsageAttributionServiceBenchmark(b *testing.B) (customer.Service, string, int) {
	b.Helper()

	db := testutils.InitPostgresDB(b, testutils.PostgresDBStateEntMigrated)
	b.Cleanup(func() { db.Close(b) })

	client := db.EntDriver.Client()

	logger := testutils.NewDiscardLogger(b)

	publisher, err := eventbus.New(eventbus.Options{
		Publisher: &noop.Publisher{},
		TopicMapping: eventbus.TopicMapping{
			IngestEventsTopic:        "bench-ingest-events",
			SystemEventsTopic:        "bench-system-events",
			BalanceWorkerEventsTopic: "bench-balance-worker-events",
		},
		Logger: logger,
	})
	if err != nil {
		b.Fatalf("build event publisher: %v", err)
	}

	adapter, err := customeradapter.New(customeradapter.Config{Client: client, Logger: logger})
	if err != nil {
		b.Fatalf("build customer adapter: %v", err)
	}

	svc, err := customerservice.New(customerservice.Config{
		Adapter:   adapter,
		Publisher: publisher,
	})
	if err != nil {
		b.Fatalf("build customer service: %v", err)
	}

	customerCount := uabench.CustomerCount(b)
	namespace := "customer-usage-attribution-service-benchmark"
	uabench.Seed(b, client, namespace, customerCount)

	return svc, namespace, customerCount
}

// BenchmarkCustomerUsageAttributionLookup measures the full single-key service path — candidate
// query, subject hydration, and the real key-over-subject precedence in
// resolveCustomersByKeyWithPrecedence — via the public service method. The adapter-level query cost
// alone is benchmarked by BenchmarkCustomerUsageAttributionQuery in the adapter package.
func BenchmarkCustomerUsageAttributionLookup(b *testing.B) {
	b.StopTimer()

	svc, namespace, customerCount := newUsageAttributionServiceBenchmark(b)
	targetCustomerID := fmt.Sprintf("%026d", customerCount)

	lookups := []struct {
		name string
		key  string
	}{
		{name: "customer_key", key: fmt.Sprintf("customer-%d", customerCount)},
		{name: "subject_key", key: fmt.Sprintf("subject-%d", customerCount)},
	}

	for _, lookup := range lookups {
		b.Run(lookup.name, func(b *testing.B) {
			get := func(ctx context.Context) (string, error) {
				c, err := svc.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
					Namespace: namespace,
					Key:       lookup.key,
				})
				if err != nil {
					return "", err
				}
				return c.ID, nil
			}

			customerID, err := get(b.Context())
			if err != nil {
				b.Fatalf("warm up lookup: %v", err)
			}
			if customerID != targetCustomerID {
				b.Fatalf("warm up lookup returned customer %q, expected %q", customerID, targetCustomerID)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := get(b.Context()); err != nil {
					b.Fatalf("lookup customer: %v", err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(customerCount), "customers")
		})
	}
}

// BenchmarkCustomersUsageAttributionBulkLookup measures the full bulk service path — one candidate
// query for the whole key set plus per-key precedence resolution into the returned map — via the
// public service method.
func BenchmarkCustomersUsageAttributionBulkLookup(b *testing.B) {
	svc, namespace, customerCount := newUsageAttributionServiceBenchmark(b)
	if customerCount < 2*uabench.BulkKeyCount {
		b.Fatalf("%s must be at least %d to build a non-overlapping bulk key set, got %d",
			uabench.CustomersEnv, 2*uabench.BulkKeyCount, customerCount)
	}

	keys := uabench.BulkKeys(customerCount)

	get := func(ctx context.Context) (int, error) {
		resolved, err := svc.GetCustomersByUsageAttribution(ctx, customer.GetCustomersByUsageAttributionInput{
			Namespace: namespace,
			Keys:      keys,
		})
		if err != nil {
			return 0, err
		}
		return len(resolved), nil
	}

	count, err := get(b.Context())
	if err != nil {
		b.Fatalf("warm up lookup: %v", err)
	}
	if count != uabench.BulkKeyCount {
		b.Fatalf("warm up lookup resolved %d customers, expected %d", count, uabench.BulkKeyCount)
	}

	// The timed loop runs inside b.Run so it gets a fresh, running timer: setup above (which seeds
	// 100k rows) is excluded from the measurement without relying on StopTimer/ResetTimer ordering.
	b.Run(fmt.Sprintf("keys_%d", uabench.BulkKeyCount), func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := get(b.Context()); err != nil {
				b.Fatalf("lookup customers: %v", err)
			}
		}
		b.StopTimer()
		b.ReportMetric(float64(customerCount), "customers")
		b.ReportMetric(float64(uabench.BulkKeyCount), "keys")
	})
}
