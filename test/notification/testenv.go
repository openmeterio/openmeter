package notification

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/notification/eventhandler"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	webhooksvix "github.com/openmeterio/openmeter/openmeter/notification/webhook/svix"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/defaultx"
)

const (
	TestMeterSlug   = "api-call"
	TestFeatureName = "API Requests"
	TestFeatureKey  = "api-call"
	TestFeatureID   = "api-call-id"
	TestSubjectKey  = "john-doe"
	TestSubjectID   = "john-doe-id"
	TestCustomerID  = "john-doe-customer-id"
	// TestWebhookURL is the target URL where the notifications are sent to.
	// Use the following URL to verify notifications events sent over webhook channel:
	// https://play.svix.com/view/e_eyihAQHBB5d6T9ck1iYevP825pg
	TestWebhookURL = "https://play.svix.com/in/e_eyihAQHBB5d6T9ck1iYevP825pg/"
	// TestSigningSecret used for verifying events sent to webhook.
	TestSigningSecret = "whsec_Fk5kgr5qTdPdQIDniFv+6K0WN2bUpdGjjGtaNeAx8N8="

	PostgresURLTemplate   = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
	SvixServerURLTemplate = "http://%s:8071"
)

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID

type TestEnv interface {
	NotificationRepo() notification.Repository
	Notification() notification.Service
	NotificationWebhook() notificationwebhook.Handler

	Feature() feature.FeatureConnector
	Meter() *meteradapter.TestAdapter

	Namespace() string

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	notificationRepo notification.Repository
	notification     notification.Service
	webhook          notificationwebhook.Handler

	feature feature.FeatureConnector
	meter   *meteradapter.TestAdapter

	namespace string

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

func (n testEnv) Namespace() string {
	return n.namespace
}

const (
	DefaultSvixHost             = "127.0.0.1"
	DefaultSvixJWTSigningSecret = "DUMMY_JWT_SECRET"
)

func NewTestEnv(t *testing.T, ctx context.Context, namespace string) (TestEnv, error) {
	t.Helper()
	logger := slog.Default().WithGroup("notification")

	tracer := noop.NewTracerProvider().Tracer("test")

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
	featureConnector := feature.NewFeatureConnector(featureAdapter, meterService, eventbus.NewMock(t), nil)

	adapter, err := notificationadapter.New(notificationadapter.Config{
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

	svixServerURL, err := url.Parse(fmt.Sprintf(SvixServerURLTemplate, svixHost))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Svix server URL: %w", err)
	}

	svixAPIClient, err := svix.New(svixAPIKey, &svix.SvixOptions{
		ServerUrl: svixServerURL,
		Debug:     false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Svix API client: %w", err)
	}

	webhook, err := webhooksvix.New(webhooksvix.Config{
		SvixAPIClient: svixAPIClient,
		Logger:        logger,
		Tracer:        tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook handler: %w", err)
	}

	eventHandler, err := eventhandler.New(eventhandler.Config{
		Repository: adapter,
		Webhook:    webhook,
		Logger:     logger,
		Tracer:     tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	if err = eventHandler.Start(); err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	service, err := notificationservice.New(notificationservice.Config{
		Adapter:          adapter,
		FeatureConnector: featureConnector,
		Webhook:          webhook,
		EventHandler:     eventHandler,
		Logger:           logger.With(slog.String("subsystem", "notification")),
	})
	if err != nil {
		return nil, err
	}

	closerFunc := func() error {
		var errs error

		if err = eventHandler.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close notification event handler: %w", err))
		}

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
		notificationRepo: adapter,
		notification:     service,
		webhook:          webhook,
		feature:          featureConnector,
		meter:            meterService,
		namespace:        namespace,
		closerFunc:       closerFunc,
	}, nil
}
