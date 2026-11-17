package common

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/rte/shadow"
)

func NewRTEShadowRecorder(ctx context.Context, conf config.RTEConfiguration, events config.EventsConfiguration, publisher message.Publisher, meter metric.Meter, logger *slog.Logger) (*shadow.Recorder, func(), error) {
	if !conf.Enabled {
		return nil, func() {}, nil
	}

	// Shadow delivery can be dropped. Keep this traffic off the business outbox.
	bus, err := newRTEDirectPublisher(events, publisher, logger)
	if err != nil {
		return nil, nil, err
	}

	recorder, err := shadow.NewRecorder(ctx, shadow.Config{
		Publish: bus.Publish, SamplePercent: conf.ShadowSamplePercent, Meter: meter, Logger: logger,
	})
	if err != nil {
		return nil, nil, err
	}

	return recorder, recorder.Close, nil
}
