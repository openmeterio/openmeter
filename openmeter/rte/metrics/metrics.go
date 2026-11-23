// Package metrics owns RTE instruments, bounded labels and recording methods.
package metrics

import "go.opentelemetry.io/otel/metric"

// durationBuckets covers sub-millisecond state reads through multi-second calculations.
// All instruments using these boundaries record seconds.
func durationBuckets() metric.HistogramOption {
	return metric.WithExplicitBucketBoundaries(0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30)
}
