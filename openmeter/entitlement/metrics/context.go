// Package metrics provides context for measuring entitlement operations.
package metrics

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

// MetricsContext carries labels for measuring entitlement calls.
type MetricsContext struct {
	Operation entitlement.CurrentValueOperation
}

type metricsContextKey struct{}

// WithMetricsContext attaches labels for observers of the call.
func WithMetricsContext(ctx context.Context, labels MetricsContext) context.Context {
	return context.WithValue(ctx, metricsContextKey{}, labels)
}

// GetMetricsContext returns the attached labels, or the zero value for an unlabeled call.
func GetMetricsContext(ctx context.Context) MetricsContext {
	labels, _ := ctx.Value(metricsContextKey{}).(MetricsContext)
	return labels
}
