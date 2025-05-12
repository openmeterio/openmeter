package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

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
)

var Notification = wire.NewSet(
	NewNotificationAdapter,
	NewNotificationService,
	NewNotificationWebhookHandler,
	NewNotificationEventHandler,
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
	logger *slog.Logger,
	adapter notification.Repository,
	webhook notificationwebhook.Handler,
) (notification.EventHandler, func(), error) {
	closeFn := func() {}

	eventHandler, err := eventhandler.New(eventhandler.Config{
		Repository: adapter,
		Webhook:    webhook,
		Logger:     logger,
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

func NewNotificationWebhookHandler(
	logger *slog.Logger,
	webhookConfig config.WebhookConfiguration,
	svixConfig config.SvixConfig,
) (notificationwebhook.Handler, error) {
	if !svixConfig.IsEnabled() {
		return webhooknoop.New(logger), nil
	}

	handler, err := webhooksvix.New(webhooksvix.Config{
		SvixConfig:              svixConfig,
		RegisterEventTypes:      notificationwebhook.NotificationEventTypes,
		RegistrationTimeout:     webhookConfig.EventTypeRegistrationTimeout,
		SkipRegistrationOnError: webhookConfig.SkipEventTypeRegistrationOnError,
		Logger:                  logger.WithGroup("notification.webhook"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification webhook handler: %w", err)
	}

	return handler, nil
}
