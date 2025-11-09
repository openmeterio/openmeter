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
	eventhandlernoop "github.com/openmeterio/openmeter/openmeter/notification/eventhandler/noop"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	webhooknoop "github.com/openmeterio/openmeter/openmeter/notification/webhook/noop"
	webhooksvix "github.com/openmeterio/openmeter/openmeter/notification/webhook/svix"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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
	NewNoopNotificationEventHandler,
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

func NewNoopNotificationEventHandler() (notification.EventHandler, func(), error) {
	handler, err := eventhandlernoop.New()
	if err != nil {
		return nil, nil, err
	}

	return handler, func() {}, nil
}

func NewNotificationEventHandler(
	config config.NotificationConfiguration,
	logger *slog.Logger,
	tracer trace.Tracer,
	adapter notification.Repository,
	webhook notificationwebhook.Handler,
) (notification.EventHandler, func(), error) {
	closeFn := func() {}

	eventHandler, err := eventhandler.New(eventhandler.Config{
		Repository:        adapter,
		Webhook:           webhook,
		Logger:            logger,
		Tracer:            tracer,
		ReconcileInterval: config.ReconcileInterval,
		SendingTimeout:    config.SendingTimeout,
		PendingTimeout:    config.PendingTimeout,
	})
	if err != nil {
		return nil, closeFn, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	if err = eventHandler.Start(); err != nil {
		return nil, closeFn, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	closeFn = func() {
		if err = eventHandler.Close(); err != nil {
			logger.Error("failed to close notification event handler", "error", err)
		}
	}

	return eventHandler, closeFn, nil
}

func NewNotificationService(
	logger *slog.Logger,
	adapter notification.Repository,
	webhook notificationwebhook.Handler,
	eventHandler notification.EventHandler,
	featureConnector feature.FeatureConnector,
) (notification.Service, error) {
	notificationService, err := notificationservice.New(notificationservice.Config{
		Adapter:          adapter,
		Webhook:          webhook,
		EventHandler:     eventHandler,
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
