package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationrepository "github.com/openmeterio/openmeter/openmeter/notification/repository"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

var Notification = wire.NewSet(
	NewNotificationService,
)

func NewNotificationService(
	logger *slog.Logger,
	db *entdb.Client,
	notificationConfig config.NotificationConfiguration,
	svixConfig config.SvixConfig,
	featureConnector feature.FeatureConnector,
) (notification.Service, error) {
	// TODO: remove this check after enabled by default
	if db == nil {
		return nil, nil
	}

	var notificationRepo notification.Repository
	notificationRepo, err := notificationrepository.New(notificationrepository.Config{
		Client: db,
		Logger: logger.WithGroup("notification.postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification repository: %w", err)
	}

	var notificationWebhook notificationwebhook.Handler
	notificationWebhook, err = notificationwebhook.New(notificationwebhook.Config{
		SvixConfig:              svixConfig,
		RegistrationTimeout:     notificationConfig.Webhook.EventTypeRegistrationTimeout,
		SkipRegistrationOnError: notificationConfig.Webhook.SkipEventTypeRegistrationOnError,
		Logger:                  logger.WithGroup("notification.webhook"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification webhook handler: %w", err)
	}

	notificationService, err := notificationservice.New(notificationservice.Config{
		Repository:       notificationRepo,
		Webhook:          notificationWebhook,
		FeatureConnector: featureConnector,
		Logger:           logger.With(slog.String("subsystem", "notification")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification service: %w", err)
	}

	return notificationService, nil
}
