package kafka

import (
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka/metrics"
)

var ingorePrefixes = []string{
	"protocol-requests-rate", // too low level, we don't need it for now
	"compression-",           // don't care
}

func SaramaMetricRenamer(role string) metrics.TransformMetricsNameToOtel {
	return func(name string) metrics.TransformedMetric {
		res := metrics.TransformedMetric{
			Name: "sarama." + name,
		}

		for _, prefix := range ingorePrefixes {
			if strings.HasPrefix(name, prefix) {
				res.Drop = true
				return res
			}
		}

		attributes := []attribute.KeyValue{
			attribute.String("role", role),
		}

		if strings.Contains(name, "for-broker") || strings.Contains(name, "for-topic") {
			res.Drop = true
			return res
		}

		res.Attributes = attribute.NewSet(attributes...)
		return res
	}
}
