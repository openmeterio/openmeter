package common

import (
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteclient "github.com/openmeterio/openmeter/openmeter/rte/client"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// NewRTEClient connects a trusted caller to RTE's stored state and command topic.
func NewRTEClient(conf config.RTEConfiguration, events config.EventsConfiguration, dbClient *db.Client, publisher message.Publisher, meter metric.Meter, logger *slog.Logger) (rte.Client, error) {
	if !conf.Enabled {
		return nil, errors.New("RTE client requires rte.enabled")
	}

	bus, err := newRTEDirectPublisher(events, publisher, logger)
	if err != nil {
		return nil, err
	}

	return rteclient.New(rteclient.Config{DB: dbClient, Publisher: bus, Meter: meter, Logger: logger})
}

// Commands are acknowledged by Kafka. Neither commands nor trial observations use the business outbox.
func newRTEDirectPublisher(events config.EventsConfiguration, publisher message.Publisher, logger *slog.Logger) (eventbus.Publisher, error) {
	return eventbus.New(eventbus.Options{
		Publisher: publisher, TopicMapping: events.EventBusTopicMapping(),
		Logger: logger, MarshalerTransformFunc: watermillkafka.AddPartitionKeyFromSubject,
	})
}
