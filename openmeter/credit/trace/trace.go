package credittrace

import (
	"strings"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func WithOwner(owner models.NamespacedID) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("owner_id", owner.ID),
		attribute.String("owner_namespace", owner.Namespace),
	)
}

func WithPeriod(period timeutil.ClosedPeriod) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("period_from", period.From.Format(time.RFC3339)),
		attribute.String("period_to", period.To.Format(time.RFC3339)),
	)
}

func WithEngineParams(params engine.RunParams) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("until", params.Until.Format(time.RFC3339)),
		attribute.String("starting_snapshot_at", params.StartingSnapshot.At.Format(time.RFC3339)),
		attribute.String("resets", strings.Join(lo.Map(params.Resets.GetTimes(), func(t time.Time, _ int) string { return t.Format(time.RFC3339) }), ", ")),
	)
}
