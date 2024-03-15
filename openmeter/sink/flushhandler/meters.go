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

package flushhandler

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type metrics struct {
	eventsReceived      metric.Int64Counter
	eventsProcessed     metric.Int64Counter
	eventsFailed        metric.Int64Counter
	eventProcessingTime metric.Int64Histogram
	eventChannelFull    metric.Int64Counter
}

func newMetrics(handlerName string, meter metric.Meter) (*metrics, error) {
	var err error

	r := &metrics{}

	if r.eventsReceived, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_received", handlerName)); err != nil {
		return nil, err
	}

	if r.eventsProcessed, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_processed", handlerName)); err != nil {
		return nil, err
	}

	if r.eventsFailed, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_failed", handlerName)); err != nil {
		return nil, err
	}

	if r.eventChannelFull, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.event_channel_full", handlerName)); err != nil {
		return nil, err
	}

	if r.eventProcessingTime, err = meter.Int64Histogram(fmt.Sprintf("sink.flush_handler.%s.event_processing_time_ms", handlerName)); err != nil {
		return nil, err
	}

	return r, nil
}
