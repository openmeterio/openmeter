package credit

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ctrace struct{}

var cTrace = &ctrace{}

func (c ctrace) WithOwner(owner models.NamespacedID) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("owner_id", owner.ID),
		attribute.String("owner_namespace", owner.Namespace),
	)
}

func (c ctrace) WithPeriod(period timeutil.Period) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("period_from", period.From.String()),
		attribute.String("period_to", period.To.String()),
	)
}
