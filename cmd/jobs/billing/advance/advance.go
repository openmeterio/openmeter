package advance

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/openmeterio/openmeter/app/common"
	appconfig "github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/cmd/jobs/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingworkerautoadvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

var namespace string

var Cmd = &cobra.Command{
	Use:   "advance",
	Short: "Invoice advance operations",
}

func init() {
	Cmd.AddCommand(ListCmd())
	Cmd.AddCommand(InvoiceCmd())
	Cmd.AddCommand(AllCmd())

	Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace the operation should be performed")
}

type autoAdvancer struct {
	*billingworkerautoadvance.AutoAdvancer

	Shutdown func()
}

func NewAutoAdvancer(ctx context.Context, conf appconfig.Configuration, logger *slog.Logger) (*autoAdvancer, error) {
	commonMetadata := common.NewMetadata(conf, "0.0.0", "billing-advancer")

	// We use a noop meter provider as we don't want to monitor cronjobs (for now)
	meterProvider := sdkmetric.NewMeterProvider()
	meter := meterProvider.Meter("billing-advancer")

	// Initialize Postgres driver
	postgresDriver, err := pgdriver.NewPostgresDriver(ctx, conf.Postgres.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize postgres driver: %w", err)
	}

	// Initialize Ent driver
	entPostgresDriver := entdriver.NewEntPostgresDriver(postgresDriver.DB())

	meterRepository := common.NewInMemoryRepository(conf.Meters)

	clickhouseConn, err := common.NewClickHouse(conf.Aggregation.ClickHouse)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clickhouse connection: %w", err)
	}

	streamingConnector, err := common.NewStreamingConnector(ctx, conf.Aggregation, clickhouseConn, meterRepository, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize streaming connection: %w", err)
	}

	brokerOptions := common.NewBrokerConfiguration(conf.Ingest.Kafka.KafkaConfiguration, conf.Telemetry.Log, commonMetadata, logger, meter)

	adminClient, err := common.NewKafkaAdminClient(conf.Ingest.Kafka.KafkaConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka admin client: %w", err)
	}

	kafkaTopicProvisionerConfig := common.NewKafkaTopicProvisionerConfig(adminClient, logger, meter, conf.Ingest.Kafka.TopicProvisionerConfig)

	topicProvisioner, err := common.NewKafkaTopicProvisioner(kafkaTopicProvisionerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka topic provisioner: %w", err)
	}

	publisher, serverPublisherShutdown, err := common.NewServerPublisher(ctx, kafka.PublisherOptions{
		Broker:           brokerOptions,
		ProvisionTopics:  common.ServerProvisionTopics(conf.Events),
		TopicProvisioner: topicProvisioner,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server publisher: %w", err)
	}

	ebPublisher, err := common.NewEventBusPublisher(publisher, conf.Events, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event bus publisher: %w", err)
	}

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     entPostgresDriver.Client(),
		StreamingConnector: streamingConnector,
		Logger:             logger,
		MeterRepository:    meterRepository,
		Publisher:          ebPublisher,
	})

	customerService, err := common.NewCustomerService(logger, entPostgresDriver.Client(), entitlementRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize customer service: %w", err)
	}

	secretService, err := common.NewUnsafeSecretService(logger, entPostgresDriver.Client())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secret service: %w", err)
	}

	appService, err := common.NewAppService(logger, entPostgresDriver.Client(), conf.Apps)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app service: %w", err)
	}

	_, err = common.NewAppStripeService(logger, entPostgresDriver.Client(), conf.Apps, appService, customerService, secretService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stripe app service: %w", err)
	}

	namespaceManager, err := common.NewNamespaceManager(nil, conf.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize namespace manager: %w", err)
	}

	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: entPostgresDriver.Client(),
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize billing adapter: %w", err)
	}

	billingService, err := billingservice.New(billingservice.Config{
		Adapter:            billingAdapter,
		CustomerService:    customerService,
		AppService:         appService,
		Logger:             logger,
		FeatureService:     entitlementRegistry.Feature,
		MeterRepo:          meterRepository,
		StreamingConnector: streamingConnector,
		Publisher:          ebPublisher,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize billing service: %w", err)
	}

	_, err = common.NewAppSandboxProvisioner(ctx, logger, conf.Apps, appService, namespaceManager, billingService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sandbox app provisioner: %w", err)
	}

	a, err := billingworkerautoadvance.NewAdvancer(billingworkerautoadvance.Config{
		BillingService: billingService,
		Logger:         logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize billing auto-advancer: %w", err)
	}

	return &autoAdvancer{
		AutoAdvancer: a,
		Shutdown:     serverPublisherShutdown,
	}, nil
}

var ListCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List invoices which can be advanced",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			conf, err := config.GetConfig()
			if err != nil {
				return err
			}

			a, err := NewAutoAdvancer(ctx, conf, slog.Default())
			if err != nil {
				return err
			}

			defer a.Shutdown()

			var ns []string
			if namespace != "" {
				ns = append(ns, namespace)
			}

			invoices, err := a.ListInvoicesToAdvance(ctx, ns, nil)
			if err != nil {
				return err
			}

			for _, invoice := range invoices {
				fmt.Printf("Namespace: %s ID: %s DraftUntil: %s\n", invoice.Namespace, invoice.ID, invoice.DraftUntil)
			}

			return nil
		},
	}

	return cmd
}

var InvoiceCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoice [INVOICE_ID]",
		Short: "Advance invoice(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			conf, err := config.GetConfig()
			if err != nil {
				return err
			}

			a, err := NewAutoAdvancer(ctx, conf, slog.Default())
			if err != nil {
				return err
			}

			defer a.Shutdown()

			if namespace == "" {
				return fmt.Errorf("invoice namespace is required")
			}

			for _, invoiceID := range args {
				_, err := a.AdvanceInvoice(ctx, billing.InvoiceID{
					Namespace: namespace,
					ID:        invoiceID,
				})
				if err != nil {
					return fmt.Errorf("failed to advance invoice %s: %w", invoiceID, err)
				}
			}

			return nil
		},
	}

	return cmd
}

var batchSize int

var AllCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Advance all eligible invoices",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			conf, err := config.GetConfig()
			if err != nil {
				return err
			}

			a, err := NewAutoAdvancer(ctx, conf, slog.Default())
			if err != nil {
				return err
			}

			defer a.Shutdown()

			var ns []string
			if namespace != "" {
				ns = append(ns, namespace)
			}

			return a.All(ctx, ns, batchSize)
		},
	}

	cmd.PersistentFlags().IntVar(&batchSize, "batch", 0, "operation batch size")

	return cmd
}
