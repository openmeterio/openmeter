package rating

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestSnapshotQuantityStaysOnRawMeterQueries pins the charge rating quantity snapshot as
// a never-cached meter consumer: the params it hands to the streaming connector must have
// Cachable=false (the zero value). Rated charge quantities become invoice amounts, and
// this path additionally relies on the stored_at cutoff, which the rollup cache cannot
// evaluate — only the read-only call sites opt in (see streaming.QueryParams.Cachable).
func TestSnapshotQuantityStaysOnRawMeterQueries(t *testing.T) {
	m := meter.Meter{
		Key:         "meter-1",
		EventType:   "api-calls",
		Aggregation: meter.MeterAggregationCount,
	}

	now := time.Now().Truncate(time.Minute)
	period := timeutil.ClosedPeriod{From: now.Add(-2 * time.Hour), To: now.Add(-time.Hour)}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent(m.Key, 10, period.From.Add(time.Minute))

	svc := &service{streamingConnector: streamingConnector}

	_, err := svc.snapshotQuantity(t.Context(), snapshotQuantityInput{
		Customer: billing.InvoiceCustomer{
			CustomerID: "customer-1",
			Name:       "Customer 1",
		},
		FeatureMeter: feature.FeatureMeter{
			Feature: feature.Feature{Namespace: "test-ns", Key: "feature-1"},
			Meter:   &m,
		},
		ServicePeriod: period,
		StoredAtLT:    now,
	})
	require.NoError(t, err)

	captured := streamingConnector.CapturedQueryMeterParams()
	require.Len(t, captured, 1)
	require.False(t, captured[0].Cachable)
}
