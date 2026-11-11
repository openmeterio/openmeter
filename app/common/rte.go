package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/oklog/run"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	creditcalculator "github.com/openmeterio/openmeter/openmeter/rte/creditbalance/calculator"
	"github.com/openmeterio/openmeter/openmeter/rte/demand"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	meteredcalculator "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement/calculator"
	rteservice "github.com/openmeterio/openmeter/openmeter/rte/service"
	"github.com/openmeterio/openmeter/openmeter/rte/upstream"
	rteworker "github.com/openmeterio/openmeter/openmeter/rte/worker"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var RTE = wire.NewSet(
	NewRTEService,
	NewRTEDemandService,
	RTEProvisionTopics,
	RTESubscriber,
	NewRTESubscriptionStatus,
	NewRTEWorker,
	RTEGroup,
)

func NewRTEService(client *db.Client, source customerbalance.Service, entitlements *registry.Entitlement, logger *slog.Logger, credits config.CreditsConfiguration, publisher eventbus.Publisher, meter metric.Meter) (rte.Service, error) {
	if !credits.Enabled {
		return nil, errors.New("RTE requires credits.enabled")
	}

	creditStore, err := rteadapter.NewCreditBalanceStore(rteadapter.Config{Client: client, Logger: logger})
	if err != nil {
		return nil, err
	}

	lookups, err := meter.Int64Counter("rte.rating.lookups", metric.WithDescription("Per-charge rating cache lookups"))
	if err != nil {
		return nil, err
	}

	creditCalculator, err := creditcalculator.New(creditcalculator.Config{
		Source: source, Logger: logger,
		RatingReuseObserver: func(ctx context.Context, hit bool) {
			outcome := "miss"
			if hit {
				outcome = "hit"
			}

			lookups.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
		},
	})
	if err != nil {
		return nil, err
	}

	meteredStore, err := rteadapter.NewMeteredEntitlementBalanceStore(rteadapter.Config{Client: client, Logger: logger})
	if err != nil {
		return nil, err
	}

	meteredCalculator, err := meteredcalculator.New(meteredcalculator.Config{Source: entitlements.MeteredEntitlement, Logger: logger})
	if err != nil {
		return nil, err
	}

	return rteservice.New(rteservice.Config{
		Meter: meter,
		CreditBalances: rteservice.Registration[creditbalance.Key, creditbalance.Balance]{
			Calculator: creditCalculator,
			Store:      creditStore,
		},
		MeteredEntitlements: rteservice.Registration[rtemetered.Key, rtemetered.Balance]{
			Calculator: meteredCalculator,
			Store:      meteredStore,
		},
		Logger:    logger,
		Publisher: publisher,
	})
}

func NewRTEDemandService(client *db.Client, engine rte.Service, logger *slog.Logger, meter metric.Meter, conf config.RTEConfiguration) (rte.DemandService, error) {
	repo, err := rteadapter.NewDemandRepository(rteadapter.Config{Client: client, Logger: logger})
	if err != nil {
		return nil, err
	}

	return demand.New(demand.Config{
		Repository: repo, Service: engine, Logger: logger, MetricMeter: meter,
		CalculationTimeout: conf.CalculationTimeout, RetryMinDelay: time.Second, RetryMaxDelay: time.Minute,
	})
}

func RTEGroup(ctx context.Context, telemetryServer TelemetryServer, worker rte.Worker, subscriptionStatus *watermillkafka.SubscriptionStatus) run.Group {
	var group run.Group
	telemetryServer.Handle("/healthz/subscriptions", subscriptionStatus)

	group.Add(
		func() error { return telemetryServer.ListenAndServe() },
		func(error) { _ = telemetryServer.Shutdown(ctx) },
	)

	group.Add(func() error { return worker.Run(ctx) }, func(error) { _ = worker.Close() })

	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	return group
}

func RTEProvisionTopics(conf config.RTEConfiguration, events config.EventsConfiguration) []pkgkafka.TopicConfig {
	var topics []pkgkafka.TopicConfig
	if conf.DLQ.AutoProvision.Enabled {
		topics = append(topics, pkgkafka.TopicConfig{
			Name:          conf.DLQ.Topic,
			Partitions:    conf.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.DLQ.AutoProvision.Retention),
		})
	}

	for _, source := range []config.EventSubsystemConfiguration{events.SystemEvents, events.IngestEvents} {
		if source.AutoProvision.Enabled {
			topics = append(topics, pkgkafka.TopicConfig{Name: source.Topic, Partitions: source.AutoProvision.Partitions})
		}
	}

	return topics
}

func NewRTESubscriptionStatus(conf config.EventsConfiguration) *watermillkafka.SubscriptionStatus {
	return watermillkafka.NewSubscriptionStatus([]string{conf.SystemEvents.Topic, conf.IngestEvents.Topic})
}

// The worker's router closes the subscriber.
func RTESubscriber(conf config.RTEConfiguration, broker watermillkafka.BrokerOptions, status *watermillkafka.SubscriptionStatus) (message.Subscriber, error) {
	subscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:             broker,
		ConsumerGroupName:  conf.ConsumerGroupName,
		SubscriptionStatus: status,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize RTE subscriber: %w", err)
	}

	return subscriber, nil
}

func NewRTEWorker(service rte.Service, demands rte.DemandService, client *db.Client, conf config.RTEConfiguration, events config.EventsConfiguration, options router.Options, publisher eventbus.Publisher) (rte.Worker, error) {
	sources, err := upstream.New(upstream.Config{Client: client, Logger: options.Logger})
	if err != nil {
		return nil, err
	}

	return rteworker.New(rteworker.Config{
		Demand: demands, Sources: sources, PollInterval: conf.PollInterval, IngestEventsTopic: events.IngestEvents.Topic,
		Service:           service,
		SystemEventsTopic: events.SystemEvents.Topic,
		Router:            options,
		Marshaler:         publisher.Marshaler(),
	})
}
