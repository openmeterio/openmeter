package billingservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestGetFeatureUsageStaysOnRawMeterQueries pins the invoice quantity snapshot path as a
// never-cached meter consumer: every params it hands to the streaming connector must have
// Cachable=false (the zero value), on both the plain and the split-line query shapes.
// Invoiced quantities are financial records; serving them from the rollup cache would let
// a stale bucket become a permanently wrong invoice, which is why only the read-only call
// sites opt in (see streaming.QueryParams.Cachable).
func TestGetFeatureUsageStaysOnRawMeterQueries(t *testing.T) {
	m := meter.Meter{
		Key:         "meter-1",
		EventType:   "api-calls",
		Aggregation: meter.MeterAggregationCount,
	}

	now := time.Now().Truncate(time.Minute)
	period := timeutil.ClosedPeriod{From: now.Add(-2 * time.Hour), To: now.Add(-time.Hour)}

	newLine := func() *billing.StandardLine {
		return &billing.StandardLine{
			StandardLineBase: billing.StandardLineBase{
				ManagedResource: models.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
					ID:              "line-1",
				},
				Period: period,
			},
		}
	}

	invoiceCustomer := billing.InvoiceCustomer{
		CustomerID: "customer-1",
		Name:       "Customer 1",
	}

	t.Run("plain line", func(t *testing.T) {
		streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
		streamingConnector.AddSimpleEvent(m.Key, 10, period.From.Add(time.Minute))

		svc := &Service{streamingConnector: streamingConnector}

		_, err := svc.getFeatureUsage(t.Context(), getFeatureUsageInput{
			Line:     newLine(),
			Meter:    m,
			Feature:  feature.Feature{Namespace: "test-ns", Key: "feature-1"},
			Customer: invoiceCustomer,
		})
		require.NoError(t, err)

		captured := streamingConnector.CapturedQueryMeterParams()
		require.Len(t, captured, 1)
		require.False(t, captured[0].Cachable)
	})

	t.Run("split line pre and up-to-end queries", func(t *testing.T) {
		streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
		streamingConnector.AddSimpleEvent(m.Key, 10, period.From.Add(time.Minute))

		svc := &Service{streamingConnector: streamingConnector}

		line := newLine()
		line.SplitLineHierarchy = &billing.SplitLineHierarchy{
			Group: billing.SplitLineGroup{
				SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
					ServicePeriod: timeutil.ClosedPeriod{
						From: period.From.Add(-time.Hour),
						To:   period.To,
					},
				},
			},
		}

		_, err := svc.getFeatureUsage(t.Context(), getFeatureUsageInput{
			Line:     line,
			Meter:    m,
			Feature:  feature.Feature{Namespace: "test-ns", Key: "feature-1"},
			Customer: invoiceCustomer,
		})
		require.NoError(t, err)

		captured := streamingConnector.CapturedQueryMeterParams()
		require.Len(t, captured, 2)
		require.False(t, captured[0].Cachable)
		require.False(t, captured[1].Cachable)
	})
}
