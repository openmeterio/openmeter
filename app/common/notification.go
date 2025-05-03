package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	webhooknoop "github.com/openmeterio/openmeter/openmeter/notification/webhook/noop"
	webhooksvix "github.com/openmeterio/openmeter/openmeter/notification/webhook/svix"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

var Notification = wire.NewSet(
	NewNotificationService,
	NewNotificationWebhookHandler,
)

func NewNotificationService(
	logger *slog.Logger,
	db *entdb.Client,
	webhook notificationwebhook.Handler,
	featureConnector feature.FeatureConnector,
) (notification.Service, error) {
	adapter, err := notificationadapter.New(notificationadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification adapter: %w", err)
	}

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
