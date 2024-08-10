package notification

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/notification"
	notificationrepository "github.com/openmeterio/openmeter/internal/notification/repository"
	notificationwebhook "github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalogadapter "github.com/openmeterio/openmeter/internal/productcatalog/adapter"
)

const (
	TestNamespace   = "default"
	TestMeterSlug   = "api-call"
	TestFeatureName = "API Requests"
	TestFeatureKey  = "api-call"
	TestSubjectKey  = "john-doe"
	// TestWebhookURL is the target URL where the notifications are sent to.
	// Use the following URL to verify notifications events sent over webhook channel:
	// https://play.svix.com/view/e_eyihAQHBB5d6T9ck1iYevP825pg
	TestWebhookURL = "https://play.svix.com/in/e_eyihAQHBB5d6T9ck1iYevP825pg/"
	// TestSigningSecret used for verifying events sent to webhook.
	TestSigningSecret = "whsec_Fk5kgr5qTdPdQIDniFv+6K0WN2bUpdGjjGtaNeAx8N8="
)

type TestEnv interface {
	NotificationRepo() notification.Repository
	Notification() notification.Service
	NotificationWebhook() notificationwebhook.Handler

	Feature() productcatalog.FeatureConnector
	Meter() meter.Repository

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	notificationRepo notification.Repository
	notification     notification.Service
	webhook          notificationwebhook.Handler

	feature productcatalog.FeatureConnector
	meter   meter.Repository

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) NotificationRepo() notification.Repository {
	return n.notificationRepo
}

func (n testEnv) Notification() notification.Service {
	return n.notification
}

func (n testEnv) NotificationWebhook() notificationwebhook.Handler {
	return n.webhook
}

func (n testEnv) Feature() productcatalog.FeatureConnector {
	return n.feature
}

func (n testEnv) Meter() meter.Repository {
	return n.meter
}

func NewTestEnv(postgresURL, clickhouseAddr, svixServerURL, svixJWTSigningSecret string) (TestEnv, error) {
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

	meterRepository := NewMeterRepository()

	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(pgClient, logger.WithGroup("feature.postgres"))
	featureConnector := productcatalog.NewFeatureConnector(featureAdapter, meterRepository)

	repo, err := notificationrepository.New(notificationrepository.Config{
		Client: pgClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create notification repo: %w", err)
	}

	// Setup webhook provider

	apiToken, err := NewSvixAuthToken(svixJWTSigningSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Svix API token: %w", err)
	}

	logger.Info("Svix API Token", slog.String("token", apiToken))

	webhook, err := notificationwebhook.New(notificationwebhook.Config{
		SvixConfig: notificationwebhook.SvixConfig{
			APIToken:  apiToken,
			ServerURL: svixServerURL,
			Debug:     false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook handler: %w", err)
	}

	connector, err := notification.New(notification.Config{
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
		meter:            meterRepository,
		closerFunc:       closerFunc,
	}, nil
}
