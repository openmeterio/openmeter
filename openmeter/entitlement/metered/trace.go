package meteredentitlement

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type mtrace struct{}

var mTrace = &mtrace{}

func (m mtrace) WithOwner(owner models.NamespacedID) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("owner_id", owner.ID),
		attribute.String("owner_namespace", owner.Namespace),
	)
}

func (m mtrace) WithPeriod(period timeutil.ClosedPeriod) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("period_from", period.From.String()),
		attribute.String("period_to", period.To.String()),
	)
}
