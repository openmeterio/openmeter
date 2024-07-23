package notification

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/internal/notification"
	notificationrepository "github.com/openmeterio/openmeter/internal/notification/repository"
	notificationwebhook "github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalogadapter "github.com/openmeterio/openmeter/internal/productcatalog/adapter"
)

const (
	TestNamespace = "default"
)

type TestEnv interface {
	NotificationRepo() notification.Repository
	NotificationConn() notification.Connector
	NotificationWebhook() notificationwebhook.Handler

	FeatureConn() productcatalog.FeatureConnector

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	notificationRepo notification.Repository
	notification     notification.Connector
	webhook          notificationwebhook.Handler

	feature productcatalog.FeatureConnector

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) NotificationRepo() notification.Repository {
	return n.notificationRepo
}

func (n testEnv) NotificationConn() notification.Connector {
	return n.notification
}

func (n testEnv) NotificationWebhook() notificationwebhook.Handler {
	return n.webhook
}

func (n testEnv) FeatureConn() productcatalog.FeatureConnector {
	return n.feature
}

func NewNotificationTestEnv(postgresURL, clickhouseAddr, svixServerURL, svixJWTSigningSecret string) (TestEnv, error) {
	logger := slog.Default().WithGroup("notification")

	chClient, err := NewClickhouseClient(clickhouseAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if err := chClient.Close(); err != nil {
				logger.Error("failed to close clickhouse client", slog.String("error", err.Error()))
			}
		}
	}()

	meterRepository := NewMeterRepository()

	pgClient, err := NewPGClient(postgresURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if err := pgClient.Close(); err != nil {
				logger.Error("failed to close postgres client", slog.String("error", err.Error()))
			}
		}
	}()

	repo, err := notificationrepository.New(notificationrepository.Config{
		Postgres: notificationrepository.PostgresAdapterConfig{
			Client: pgClient,
			Logger: logger.WithGroup("postgres"),
		},
		Clickhouse: notificationrepository.ClickhouseAdapterConfig{
			Connection:              chClient,
			Logger:                  logger.WithGroup("clickhouse"),
			Database:                "openmeter",
			EventsTableName:         "om_notification_events",
			DeliveryStatusTableName: "om_notification_delivery_status",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create notification repo: %w", err)
	}

	// Setup webhook provider

	authToken, err := NewSvixAuthToken(svixJWTSigningSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Svix auth token: %w", err)
	}

	logger.Info("Svix Auth Token", slog.String("token", authToken))

	webhook, err := notificationwebhook.NewHandler(notificationwebhook.Config{
		ServerURL: svixServerURL,
		AuthToken: authToken,
		Debug:     false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook handler: %w", err)
	}

	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(pgClient, logger.WithGroup("feature.postgres"))
	featureConnector := productcatalog.NewFeatureConnector(featureAdapter, meterRepository)

	connector, err := notification.NewConnector(notification.ConnectorConfig{
		Repository:       repo,
		FeatureConnector: featureConnector,
		Webhook:          webhook,
		Logger:           logger.WithGroup("notification"),
	})
	if err != nil {
		return nil, err
	}

	closerFunc := func() error {
		var errs error

		if err := pgClient.Close(); err != nil {
			errs = errors.Join(errs, err)
		}

		if err := chClient.Close(); err != nil {
			errs = errors.Join(errs, err)
		}

		return errs
	}

	return &testEnv{
		notificationRepo: repo,
		notification:     connector,
		webhook:          webhook,
		feature:          featureConnector,
		closerFunc:       closerFunc,
	}, nil
}
