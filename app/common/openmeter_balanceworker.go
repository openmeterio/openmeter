package common

import (
	"context"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/oklog/run"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/openmeter/registry"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

func BalanceWorkerProvisionTopics(conf config.BalanceWorkerConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.DLQ.Topic,
			Partitions:    conf.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.DLQ.AutoProvision.Retention),
		})
	}

	return provisionTopics
}

// no closer function: the subscriber is closed by the router/worker
func BalanceWorkerSubscriber(conf config.BalanceWorkerConfiguration, brokerOptions watermillkafka.BrokerOptions) (message.Subscriber, error) {
	subscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            brokerOptions,
		ConsumerGroupName: conf.ConsumerGroupName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka subscriber: %w", err)
	}

	return subscriber, nil
}

func NewBalanceWorkerOptions(
	eventConfig config.EventsConfiguration,
	routerOptions router.Options,
	eventBus eventbus.Publisher,
	entitlements *registry.Entitlement,
	repo balanceworker.BalanceWorkerRepository,
	subjectResolver balanceworker.SubjectResolver,
	logger *slog.Logger,
) balanceworker.WorkerOptions {
	return balanceworker.WorkerOptions{
		SystemEventsTopic: eventConfig.SystemEvents.Topic,
		IngestEventsTopic: eventConfig.IngestEvents.Topic,

		Router:          routerOptions,
		EventBus:        eventBus,
		Entitlement:     entitlements,
		Repo:            repo,
		SubjectResolver: subjectResolver,
		Logger:          logger,
	}
}

func NewBalanceWorker(workerOptions balanceworker.WorkerOptions) (*balanceworker.Worker, error) {
	worker, err := balanceworker.New(workerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker: %w", err)
	}

	return worker, nil
}

func BalanceWorkerGroup(
	ctx context.Context,
	worker *balanceworker.Worker,
	telemetryServer TelemetryServer,
) run.Group {
	var group run.Group

	group.Add(
		func() error { return telemetryServer.ListenAndServe() },
		func(err error) { _ = telemetryServer.Shutdown(ctx) },
	)

	group.Add(
		func() error { return worker.Run(ctx) },
		func(err error) { _ = worker.Close() },
	)

	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	return group
}
