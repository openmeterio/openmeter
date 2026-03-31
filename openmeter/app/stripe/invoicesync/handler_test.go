package invoicesync

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestNewHandler_AllFieldsRequired(t *testing.T) {
	// Build a fully valid config, then nil out one field at a time
	// to verify each required field produces an appropriate error.
	validConfig := HandlerConfig{
		Adapter:          &mockAdapter{},
		AppService:       &noopAppService{},
		BillingService:   billing.NoopService{},
		StripeAppService: &noopStripeAppService{},
		SecretService:    &noopSecretService{},
		StripeAppClientFactory: func(config stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return nil, nil
		},
		Publisher: noopPublisher{},
		LockFunc: func(ctx context.Context, namespace, invoiceID string) error {
			return nil
		},
		Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	tests := []struct {
		name    string
		mutate  func(c *HandlerConfig)
		wantErr string
	}{
		{"adapter", func(c *HandlerConfig) { c.Adapter = nil }, "adapter is required"},
		{"app service", func(c *HandlerConfig) { c.AppService = nil }, "app service is required"},
		{"billing service", func(c *HandlerConfig) { c.BillingService = nil }, "billing service is required"},
		{"stripe app service", func(c *HandlerConfig) { c.StripeAppService = nil }, "stripe app service is required"},
		{"secret service", func(c *HandlerConfig) { c.SecretService = nil }, "secret service is required"},
		{"stripe app client factory", func(c *HandlerConfig) { c.StripeAppClientFactory = nil }, "stripe app client factory is required"},
		{"publisher", func(c *HandlerConfig) { c.Publisher = nil }, "publisher is required"},
		{"lock function", func(c *HandlerConfig) { c.LockFunc = nil }, "lock function is required"},
		{"logger", func(c *HandlerConfig) { c.Logger = nil }, "logger is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig
			tt.mutate(&cfg)
			_, err := NewHandler(cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestHandle_NilEvent(t *testing.T) {
	handler := testHandler(t, &mockAdapter{})
	err := handler.Handle(context.Background(), nil)
	assert.NoError(t, err)
}

func TestHandle_PlanNotFound(t *testing.T) {
	adapter := &mockAdapter{}
	adapter.On("GetSyncPlan", mock.Anything, "plan-missing").Return((*SyncPlan)(nil), nil)

	handler := testHandler(t, adapter)
	err := handler.Handle(context.Background(), &ExecuteSyncPlanEvent{
		PlanID: "plan-missing", InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
	})
	assert.NoError(t, err)
	adapter.AssertExpectations(t)
}

func TestHandle_AlreadyCompleted(t *testing.T) {
	adapter := &mockAdapter{}
	adapter.On("GetSyncPlan", mock.Anything, "plan-done").Return(&SyncPlan{
		ID: "plan-done", Status: PlanStatusCompleted,
	}, nil)

	handler := testHandler(t, adapter)
	err := handler.Handle(context.Background(), &ExecuteSyncPlanEvent{
		PlanID: "plan-done", InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
	})
	assert.NoError(t, err)
	adapter.AssertExpectations(t)
}

func TestHandle_AlreadyFailed(t *testing.T) {
	adapter := &mockAdapter{}
	adapter.On("GetSyncPlan", mock.Anything, "plan-fail").Return(&SyncPlan{
		ID: "plan-fail", Status: PlanStatusFailed,
	}, nil)

	handler := testHandler(t, adapter)
	err := handler.Handle(context.Background(), &ExecuteSyncPlanEvent{
		PlanID: "plan-fail", InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
	})
	assert.NoError(t, err)
	adapter.AssertExpectations(t)
}

func TestHandle_GetPlanError(t *testing.T) {
	adapter := &mockAdapter{}
	adapter.On("GetSyncPlan", mock.Anything, "plan-err").Return((*SyncPlan)(nil), assert.AnError)

	handler := testHandler(t, adapter)
	err := handler.Handle(context.Background(), &ExecuteSyncPlanEvent{
		PlanID: "plan-err", InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting sync plan")
}

func TestHandle_PlanWithNoAppID(t *testing.T) {
	adapter := &mockAdapter{}
	adapter.On("GetSyncPlan", mock.Anything, "plan-no-app").Return(&SyncPlan{
		ID:     "plan-no-app",
		Status: PlanStatusPending,
		AppID:  "", // missing
	}, nil)

	handler := testHandler(t, adapter)
	err := handler.Handle(context.Background(), &ExecuteSyncPlanEvent{
		PlanID: "plan-no-app", InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app ID")
}

// testHandler creates a Handler directly, bypassing NewHandler validation.
// This allows testing Handle() paths without needing to implement full service interfaces.
// The handler's service fields (billingService, etc.) are nil — tests must only exercise
// code paths that don't call into them (nil event, plan not found, terminal state).
func testHandler(t *testing.T, adapter *mockAdapter) *Handler {
	t.Helper()
	return &Handler{
		adapter: adapter,
		stripeAppClientFactory: func(config stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return nil, nil
		},
		lockFunc: func(ctx context.Context, namespace, planID string) error {
			return nil // no-op lock for tests
		},
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}
