package debug

import (
	"bytes"
	"context"
	"fmt"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// DebugConnector is a connector for debug metrics.
type DebugConnector interface {
	GetDebugMetrics(ctx context.Context, namespace string) (string, error)
}

// debugConnector is the internal implementation of the DebugConnector interface.
type debugConnector struct {
	streaming streaming.Connector
}

// NewDebugConnector creates a new DebugConnector.
func NewDebugConnector(streaming streaming.Connector) DebugConnector {
	return &debugConnector{
		streaming: streaming,
	}
}

// GetDebugMetrics returns metrics in an OpenMetrics (Prometheus) format for debugging purposes.
// It is useful to monitor the number of events ingested on the vendor side.
func (c *debugConnector) GetDebugMetrics(ctx context.Context, namespace string) (string, error) {
	// Start from the beginning of the day
	queryParams := streaming.CountEventsParams{
		From: time.Now().Truncate(time.Hour * 24).UTC(),
	}

	// Query events counts
	rows, err := c.streaming.CountEvents(ctx, namespace, queryParams)
	if err != nil {
		return "", fmt.Errorf("connector count events: %w", err)
	}

	// Convert to Prometheus metrics
	var metrics []*dto.Metric
	for _, row := range rows {
		metric := &dto.Metric{
			Label: []*dto.LabelPair{
				{
					Name:  proto.String("subject"),
					Value: proto.String(row.Subject),
				},
			},
			Counter: &dto.Counter{
				// We can lose precision here
				Value:            proto.Float64(float64(row.Count)),
				CreatedTimestamp: timestamppb.New(time.Now()),
			},
		}

		if row.IsError {
			metric.Label = append(metric.Label, &dto.LabelPair{
				Name:  proto.String("error"),
				Value: proto.String("true"),
			})
		}

		metrics = append(metrics, metric)
	}

	family := &dto.MetricFamily{
		Name:   proto.String("openmeter_events_total"),
		Help:   proto.String("Number of ingested events"),
		Type:   dto.MetricType_COUNTER.Enum(),
		Unit:   proto.String("events"),
		Metric: metrics,
	}

	var out bytes.Buffer
	_, err = expfmt.MetricFamilyToOpenMetrics(&out, family)
	if err != nil {
		return "", fmt.Errorf("convert metric family to OpenMetrics: %w", err)
	}

	return out.String(), nil
}
