package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"
	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/notification/eventhandler"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	webhooknoop "github.com/openmeterio/openmeter/openmeter/notification/webhook/noop"
	webhooksvix "github.com/openmeterio/openmeter/openmeter/notification/webhook/svix"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

var Notification = wire.NewSet(
	NewNotificationAdapter,
	NewNotificationService,
	NewNotificationWebhookHandler,
	NewNotificationEventHandler,
)

// NotificationService is a wire set for the notification service, it can be used at
// places where only the service is required without svix and event handler.
var NotificationService = wire.NewSet(
	NewNotificationAdapter,
	NewNotificationService,
	NewNoopNotificationWebhookHandler,
)

func NewNotificationAdapter(
	logger *slog.Logger,
	db *entdb.Client,
) (notification.Repository, error) {
	adapter, err := notificationadapter.New(notificationadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification adapter: %w", err)
	}

	return adapter, nil
}

func NewNotificationEventHandler(
	config config.NotificationConfiguration,
	logger *slog.Logger,
	tracer trace.Tracer,
	adapter notification.Repository,
	webhook notificationwebhook.Handler,
	driver *pgdriver.Driver,
) (notification.EventHandler, error) {
	sessionLockr, err := lockr.NewSessionLockr(lockr.SessionLockerConfig{
		Logger:         logger,
		PostgresDriver: driver,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session lockr: %w", err)
	}

	eventHandler, err := eventhandler.New(eventhandler.Config{
		Repository:        adapter,
		Webhook:           webhook,
		Logger:            logger,
		Tracer:            tracer,
		ReconcileInterval: config.ReconcileInterval,
		SendingTimeout:    config.SendingTimeout,
		PendingTimeout:    config.PendingTimeout,
		ReconcilerWorkers: config.ReconcilerWorkers,
		Lockr:             sessionLockr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	return eventHandler, nil
}

func NewNotificationService(
	logger *slog.Logger,
	adapter notification.Repository,
	webhook notificationwebhook.Handler,
	featureConnector feature.FeatureConnector,
) (notification.Service, error) {
	notificationService, err := notificationservice.New(notificationservice.Config{
		Adapter:          adapter,
		Webhook:          webhook,
		FeatureConnector: featureConnector,
		Logger:           logger.With(slog.String("subsystem", "notification")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification service: %w", err)
	}

	return notificationService, nil
}

func NewNoopNotificationWebhookHandler(
	logger *slog.Logger,
) (notificationwebhook.Handler, error) {
	return webhooknoop.New(logger), nil
}

func NewNotificationWebhookHandler(
	logger *slog.Logger,
	tracer trace.Tracer,
	webhookConfig config.WebhookConfiguration,
	svixClient *svix.Svix,
) (notificationwebhook.Handler, error) {
	if svixClient == nil {
		logger.Warn("svix client not configured, using noop handler")

		return webhooknoop.New(logger), nil
	}

	handler, err := webhooksvix.New(webhooksvix.Config{
		SvixAPIClient:           svixClient,
		RegisterEventTypes:      notificationwebhook.NotificationEventTypes,
		RegistrationTimeout:     webhookConfig.EventTypeRegistrationTimeout,
		SkipRegistrationOnError: webhookConfig.SkipEventTypeRegistrationOnError,
		Logger:                  logger.WithGroup("notification.webhook"),
		Tracer:                  tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification webhook handler: %w", err)
	}

	return handler, nil
}
