package optimizedengine

import (
	"context"
	"encoding/json"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	credittrace "github.com/openmeterio/openmeter/openmeter/credit/trace"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Wraps the engine in a span to trace the engine's execution
// TODO: we could/should trace internals too to better understand / track the execution
type engineWrapper struct {
	engine.Engine
	Tracer trace.Tracer
	Logger *slog.Logger
}

func (w *engineWrapper) Run(ctx context.Context, params engine.RunParams) (engine.RunResult, error) {
	ctx, span := w.Tracer.Start(ctx, "credit.runEngine", credittrace.WithEngineParams(params))
	defer span.End()

	res, err := w.Engine.Run(ctx, params)

	// Let's annotate the span with the calculated history periods so we understand the engine's execution
	// We can do it even if we got an error, worst case scenario we'll have an empty list of periods.
	periods := res.History.GetPeriods()

	periodsJSON, marshalErr := json.Marshal(periods)
	if marshalErr != nil {
		w.Logger.WarnContext(ctx, "failed to marshal periods for tracing", "error", err)
	} else {
		span.SetAttributes(attribute.String("periods", string(periodsJSON)))
	}

	return res, err
}

type usageQuerierWrapper struct {
	balance.UsageQuerier
	Tracer trace.Tracer
}

func (w *usageQuerierWrapper) QueryUsage(ctx context.Context, ownerID models.NamespacedID, period timeutil.ClosedPeriod) (float64, error) {
	ctx, span := w.Tracer.Start(
		ctx,
		"credit.QueryUsageFn",
		trace.WithAttributes(attribute.String("from", period.From.String())),
		trace.WithAttributes(attribute.String("to", period.To.String())),
	)
	defer span.End()

	return w.UsageQuerier.QueryUsage(ctx, ownerID, period)
}
