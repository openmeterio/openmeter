package common

import (
	"context"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/oklog/run"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subject"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var BalanceWorker = wire.NewSet(
	BalanceWorkerProvisionTopics,
	BalanceWorkerSubscriber,
	NewEntitlementRegistry,
	NewBalanceWorkerOptions,
	NewBalanceWorker,
	BalanceWorkerGroup,
	NewBalanceWorkerFilterStateStorage,
)

var BalanceWorkerAdapter = wire.NewSet(
	NewBalanceWorkerEntitlementRepo,

	wire.Bind(new(balanceworker.BalanceWorkerRepository), new(BalanceWorkerEntitlementRepo)),
)

type BalanceWorkerEntitlementRepo interface {
	entitlement.EntitlementRepo
	balanceworker.BalanceWorkerRepository
}

func NewBalanceWorkerEntitlementRepo(db *db.Client) BalanceWorkerEntitlementRepo {
	return entitlementadapter.NewPostgresEntitlementRepo(db)
}

func BalanceWorkerProvisionTopics(conf config.BalanceWorkerConfiguration) watermillkafka.ProvisionTopics {
	var provisionTopics watermillkafka.ProvisionTopics

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
	notificationService notification.Service,
	subjectService subject.Service,
	customerService customer.Service,
	logger *slog.Logger,
	filterStateStorage balanceworker.FilterStateStorage,
) balanceworker.WorkerOptions {
	return balanceworker.WorkerOptions{
		SystemEventsTopic:   eventConfig.SystemEvents.Topic,
		IngestEventsTopic:   eventConfig.IngestEvents.Topic,
		Router:              routerOptions,
		EventBus:            eventBus,
		Entitlement:         entitlements,
		Repo:                repo,
		Subject:             subjectService,
		Customer:            customerService,
		Logger:              logger,
		MetricMeter:         routerOptions.MetricMeter,
		NotificationService: notificationService,
		FilterStateStorage:  filterStateStorage,
	}
}

func NewBalanceWorker(workerOptions balanceworker.WorkerOptions) (*balanceworker.Worker, error) {
	worker, err := balanceworker.New(workerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker: %w", err)
	}

	return worker, nil
}

func NewBalanceWorkerFilterStateStorage(conf config.BalanceWorkerConfiguration) (balanceworker.FilterStateStorage, error) {
	switch conf.StateStorage.Driver {
	case config.BalanceWorkerStateStorageDriverRedis:
		redis, err := conf.StateStorage.GetRedisBackendConfiguration()
		if err != nil {
			return balanceworker.FilterStateStorage{}, fmt.Errorf("failed to get redis backend configuration: %w", err)
		}

		client, err := redis.NewClient()
		if err != nil {
			return balanceworker.FilterStateStorage{}, fmt.Errorf("failed to create redis client: %w", err)
		}

		return balanceworker.NewFilterStateStorage(balanceworker.FilterStateStorageRedis{
			Client:     client,
			Expiration: redis.Expiration,
		})
	case config.BalanceWorkerStateStorageDriverInMemory:
		return balanceworker.NewFilterStateStorage(balanceworker.FilterStateStorageInMemory{})
	default:
		return balanceworker.FilterStateStorage{}, fmt.Errorf("unsupported state storage driver: %s", conf.StateStorage.Driver)
	}
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
