package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationrepository "github.com/openmeterio/openmeter/openmeter/notification/repository"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/defaultx"
)

const (
	TestNamespace   = "default"
	TestMeterSlug   = "api-call"
	TestFeatureName = "API Requests"
	TestFeatureKey  = "api-call"
	TestFeatureID   = "api-call-id"
	TestSubjectKey  = "john-doe"
	TestSubjectID   = "john-doe-id"
	// TestWebhookURL is the target URL where the notifications are sent to.
	// Use the following URL to verify notifications events sent over webhook channel:
	// https://play.svix.com/view/e_eyihAQHBB5d6T9ck1iYevP825pg
	TestWebhookURL = "https://play.svix.com/in/e_eyihAQHBB5d6T9ck1iYevP825pg/"
	// TestSigningSecret used for verifying events sent to webhook.
	TestSigningSecret = "whsec_Fk5kgr5qTdPdQIDniFv+6K0WN2bUpdGjjGtaNeAx8N8="

	PostgresURLTemplate   = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
	SvixServerURLTemplate = "http://%s:8071"
)

type TestEnv interface {
	NotificationRepo() notification.Repository
	Notification() notification.Service
	NotificationWebhook() notificationwebhook.Handler

	Feature() feature.FeatureConnector
	Meter() *meteradapter.TestAdapter

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	notificationRepo notification.Repository
	notification     notification.Service
	webhook          notificationwebhook.Handler

	feature feature.FeatureConnector
	meter   *meteradapter.TestAdapter

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

func (n testEnv) Feature() feature.FeatureConnector {
	return n.feature
}

func (n testEnv) Meter() *meteradapter.TestAdapter {
	return n.meter
}

const (
	DefaultSvixHost             = "127.0.0.1"
	DefaultSvixJWTSigningSecret = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3MjI5NzYyNzMsImV4cCI6MjAzODMzNjI3MywibmJmIjoxNzIyOTc2MjczLCJpc3MiOiJzdml4LXNlcnZlciIsInN1YiI6Im9yZ18yM3JiOFlkR3FNVDBxSXpwZ0d3ZFhmSGlyTXUifQ.PomP6JWRI62W5N4GtNdJm2h635Q5F54eij0J3BU-_Ds"
)

func NewTestEnv(t *testing.T, ctx context.Context, namespace string) (TestEnv, error) {
	t.Helper()
	logger := slog.Default().WithGroup("notification")

	driver := testutils.InitPostgresDB(t)

	entClient := driver.EntDriver.Client()

	if err := entClient.Schema.Create(ctx); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	meterService, err := meteradapter.New([]meter.Meter{})
	if err != nil {
		return nil, fmt.Errorf("failed to create meter service: %w", err)
	}

	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(entClient, logger.WithGroup("feature.postgres"))
	featureConnector := feature.NewFeatureConnector(featureAdapter, meterService)

	repo, err := notificationrepository.New(notificationrepository.Config{
		Client: entClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create notification repo: %w", err)
	}

	// Setup webhook provider

	svixHost := defaultx.IfZero(os.Getenv("SVIX_HOST"), DefaultSvixHost)
	svixJWTSigningSecret := defaultx.IfZero(os.Getenv("SVIX_JWT_SECRET"), DefaultSvixJWTSigningSecret)

	svixAPIKey, err := NewSvixAuthToken(svixJWTSigningSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Svix API token: %w", err)
	}

	logger.Info("Svix API key", slog.String("apiKey", svixAPIKey))

	webhook, err := notificationwebhook.New(notificationwebhook.Config{
		SvixConfig: notificationwebhook.SvixConfig{
			APIKey:    svixAPIKey,
			ServerURL: fmt.Sprintf(SvixServerURLTemplate, svixHost),
			Debug:     false,
		},
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook handler: %w", err)
	}

	service, err := notificationservice.New(notificationservice.Config{
		Repository:       repo,
		FeatureConnector: featureConnector,
		Webhook:          webhook,
		Logger:           logger.With(slog.String("subsystem", "notification")),
	})
	if err != nil {
		return nil, err
	}

	closerFunc := func() error {
		var errs error

		if err = entClient.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = driver.EntDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = driver.PGDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close postgres driver: %w", err))
		}

		return errs
	}

	return &testEnv{
		notificationRepo: repo,
		notification:     service,
		webhook:          webhook,
		feature:          featureConnector,
		meter:            meterService,
		closerFunc:       closerFunc,
	}, nil
}
