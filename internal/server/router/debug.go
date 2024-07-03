package router

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) GetDebugMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getDebugMetrics")

	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Start from the beginning of the day
	queryParams := streaming.CountEventsParams{
		From: time.Now().Truncate(time.Hour * 24).UTC(),
	}

	// Query events counts
	rows, err := a.config.StreamingConnector.CountEvents(ctx, namespace, queryParams)
	if err != nil {
		err := fmt.Errorf("connector count events: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w)

		return
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
		err := fmt.Errorf("convert metric family to OpenMetrics: %w", err)
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w)

		return
	}

	render.PlainText(w, r, out.String())
}
