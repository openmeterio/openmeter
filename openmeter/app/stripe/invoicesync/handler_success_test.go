package invoicesync

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type spyBilling struct {
	billing.NoopService
	syncDraftCalls int
}

func (s *spyBilling) SyncDraftInvoice(ctx context.Context, input billing.SyncDraftStandardInvoiceInput) (billing.StandardInvoice, error) {
	s.syncDraftCalls++
	return s.NoopService.SyncDraftInvoice(ctx, input)
}

type noopStripeAppService struct{}

func (noopStripeAppService) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	aid := input.AppID
	return appstripeentity.AppData{
		StripeAccountID: "acct_test",
		APIKey:          secretentity.NewSecretID(aid, "secret-id", appstripeentity.APIKeySecretKey),
		StripeWebhookID: "wh_test",
		WebhookSecret:   secretentity.NewSecretID(aid, "whsec-id", appstripeentity.WebhookSecretKey),
	}, nil
}

type noopPublisher struct{}

func (noopPublisher) Publish(ctx context.Context, event marshaler.Event) error { return nil }

func (n noopPublisher) WithContext(ctx context.Context) eventbus.ContextPublisher {
	return noopCtxPub{}
}

func (noopPublisher) Marshaler() marshaler.Marshaler { return nil }

type noopCtxPub struct{}

func (noopCtxPub) PublishIfNoError(event marshaler.Event, err error) error { return err }

func TestHandle_SuccessfulSync(t *testing.T) {
	ctx := context.Background()
	planID := "plan-ok"
	pendingPlan := &SyncPlan{
		ID: planID, Status: PlanStatusPending, AppID: "app-1",
		Phase: SyncPlanPhaseDraft,
	}
	completedPlan := &SyncPlan{
		ID: planID, Status: PlanStatusCompleted, AppID: "app-1",
		Phase: SyncPlanPhaseDraft,
	}

	adapter := &mockAdapter{}
	adapter.On("GetSyncPlan", mock.Anything, planID).Return(pendingPlan, nil).Times(2)
	adapter.On("GetSyncPlan", mock.Anything, planID).Return(completedPlan, nil).Once()
	adapter.On("GetActiveSyncPlansByInvoice", mock.Anything, mock.Anything, mock.Anything).Return([]SyncPlan{}, nil)
	adapter.On("UpdatePlanStatus", mock.Anything, planID, PlanStatusExecuting, (*string)(nil)).Return(nil).Once()
	adapter.On("GetNextPendingOperation", mock.Anything, planID).Return((*SyncOperation)(nil), nil).Once()
	adapter.On("CompletePlan", mock.Anything, planID).Return(nil).Once()

	spy := &spyBilling{}

	h, err := NewHandler(HandlerConfig{
		Adapter:          adapter,
		AppService:       noopAppService{},
		BillingService:   spy,
		StripeAppService: noopStripeAppService{},
		SecretService:    noopSecretService{},
		StripeAppClientFactory: func(stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return nil, nil
		},
		Publisher: noopPublisher{},
		LockFunc: func(ctx context.Context, namespace, planID string) error {
			return nil
		},
		Logger: slog.New(slog.DiscardHandler),
	})
	require.NoError(t, err)

	err = h.Handle(ctx, &ExecuteSyncPlanEvent{
		PlanID: planID, InvoiceID: "inv-1", Namespace: "ns", CustomerID: "cust-1",
	})
	require.NoError(t, err)
	require.Equal(t, 1, spy.syncDraftCalls, "draft completion should sync invoice")
	adapter.AssertExpectations(t)
}
