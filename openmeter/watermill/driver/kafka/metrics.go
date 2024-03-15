// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
