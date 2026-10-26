package testutils

import (
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	svix "github.com/svix/svix-webhooks/v2/go"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	DefaultSvixAPIURL      = "http://127.0.0.1:8071"
	DefaultSvixJWTSecret   = "DUMMY_JWT_SECRET"
	DefaultSvixWebhookHost = "host.docker.internal"
)

type TestEnv struct {
	Client *svix.Svix
	Logger *slog.Logger
	Tracer trace.Tracer

	// WebhookHost is advertised by test receivers and must be reachable from Svix.
	WebhookHost string
}

// NewTestEnv configures a client for the live Svix server using the same url,
// signing secret, and organization as test/notification. It creates no resources.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	svixAPIURL := lo.CoalesceOrEmpty(os.Getenv("SVIX_API_URL"), DefaultSvixAPIURL)

	signingSecret := lo.CoalesceOrEmpty(os.Getenv("SVIX_JWT_SECRET"), DefaultSvixJWTSecret)

	webhookHost := lo.CoalesceOrEmpty(os.Getenv("SVIX_WEBHOOK_HOST"), DefaultSvixWebhookHost)

	apiKey, err := NewSvixAuthToken(signingSecret)
	require.NoError(t, err, "generating Svix API token must succeed")

	serverURL, err := url.Parse(svixAPIURL)
	require.NoError(t, err, "parsing Svix server URL must succeed")

	client, err := svix.New(apiKey, &svix.SvixOptions{
		ServerUrl: serverURL,
	})
	require.NoError(t, err, "initializing Svix API client must succeed")

	require.Eventuallyf(t, func() bool {
		err := client.Health().Get(t.Context())

		return err == nil
	}, 10*time.Second, 100*time.Millisecond, "failed to connect to Svix server [url=%s]", serverURL.String())

	logger := slog.New(slog.NewTextHandler(t.Output(), nil))

	tracer := noop.NewTracerProvider().Tracer("svix-provider-test")

	return &TestEnv{
		Client:      client,
		Logger:      logger,
		Tracer:      tracer,
		WebhookHost: webhookHost,
	}
}
