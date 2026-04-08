package invoicesync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// mockAdapter is a test mock for the Adapter interface.
type mockAdapter struct {
	mock.Mock
}

func (m *mockAdapter) CreateSyncPlan(ctx context.Context, plan SyncPlan) (SyncPlan, error) {
	args := m.Called(ctx, plan)
	return args.Get(0).(SyncPlan), args.Error(1)
}

func (m *mockAdapter) GetSyncPlan(ctx context.Context, planID string) (*SyncPlan, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncPlan), args.Error(1)
}

func (m *mockAdapter) GetActiveSyncPlanByInvoice(ctx context.Context, namespace, invoiceID string, phase SyncPlanPhase) (*SyncPlan, error) {
	args := m.Called(ctx, namespace, invoiceID, phase)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncPlan), args.Error(1)
}

func (m *mockAdapter) GetActiveSyncPlansByInvoice(ctx context.Context, namespace, invoiceID string) ([]SyncPlan, error) {
	args := m.Called(ctx, namespace, invoiceID)
	return args.Get(0).([]SyncPlan), args.Error(1)
}

func (m *mockAdapter) GetNextPendingOperation(ctx context.Context, planID string) (*SyncOperation, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncOperation), args.Error(1)
}

func (m *mockAdapter) CompleteOperation(ctx context.Context, opID string, stripeResponse json.RawMessage) error {
	return m.Called(ctx, opID, stripeResponse).Error(0)
}

func (m *mockAdapter) FailOperation(ctx context.Context, opID string, errMsg string) error {
	return m.Called(ctx, opID, errMsg).Error(0)
}

func (m *mockAdapter) UpdatePlanStatus(ctx context.Context, planID string, status PlanStatus, errMsg *string) error {
	return m.Called(ctx, planID, status, errMsg).Error(0)
}

func (m *mockAdapter) CompletePlan(ctx context.Context, planID string) error {
	return m.Called(ctx, planID).Error(0)
}

func (m *mockAdapter) FailPlan(ctx context.Context, planID string, errMsg string) error {
	return m.Called(ctx, planID, errMsg).Error(0)
}

type noopTxDriver struct{}

func (noopTxDriver) Commit() error    { return nil }
func (noopTxDriver) Rollback() error  { return nil }
func (noopTxDriver) SavePoint() error { return nil }

func (m *mockAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, noopTxDriver{}, nil
}

// mockStripeClient is a test mock for the StripeAppClient interface.
type mockStripeClient struct {
	mock.Mock
}

func (m *mockStripeClient) DeleteWebhook(ctx context.Context, input stripeclient.DeleteWebhookInput) error {
	return m.Called(ctx, input).Error(0)
}

func (m *mockStripeClient) GetAccount(ctx context.Context) (stripeclient.StripeAccount, error) {
	args := m.Called(ctx)
	return args.Get(0).(stripeclient.StripeAccount), args.Error(1)
}

func (m *mockStripeClient) GetCustomer(ctx context.Context, id string) (stripeclient.StripeCustomer, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(stripeclient.StripeCustomer), args.Error(1)
}

func (m *mockStripeClient) CreateCustomer(ctx context.Context, input stripeclient.CreateStripeCustomerInput) (stripeclient.StripeCustomer, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(stripeclient.StripeCustomer), args.Error(1)
}

func (m *mockStripeClient) CreateCheckoutSession(ctx context.Context, input stripeclient.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(stripeclient.StripeCheckoutSession), args.Error(1)
}

func (m *mockStripeClient) GetPaymentMethod(ctx context.Context, id string) (stripeclient.StripePaymentMethod, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(stripeclient.StripePaymentMethod), args.Error(1)
}

func (m *mockStripeClient) CreatePortalSession(ctx context.Context, input stripeclient.CreatePortalSessionInput) (stripeclient.PortalSession, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(stripeclient.PortalSession), args.Error(1)
}

func (m *mockStripeClient) GetInvoice(ctx context.Context, input stripeclient.GetInvoiceInput) (*stripe.Invoice, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (m *mockStripeClient) CreateInvoice(ctx context.Context, input stripeclient.CreateInvoiceInput) (*stripe.Invoice, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (m *mockStripeClient) UpdateInvoice(ctx context.Context, input stripeclient.UpdateInvoiceInput) (*stripe.Invoice, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (m *mockStripeClient) DeleteInvoice(ctx context.Context, input stripeclient.DeleteInvoiceInput) error {
	return m.Called(ctx, input).Error(0)
}

func (m *mockStripeClient) FinalizeInvoice(ctx context.Context, input stripeclient.FinalizeInvoiceInput) (*stripe.Invoice, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (m *mockStripeClient) ListInvoiceLineItems(ctx context.Context, id string) ([]*stripe.InvoiceLineItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*stripe.InvoiceLineItem), args.Error(1)
}

func (m *mockStripeClient) AddInvoiceLines(ctx context.Context, input stripeclient.AddInvoiceLinesInput) ([]stripeclient.StripeInvoiceItemWithLineID, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]stripeclient.StripeInvoiceItemWithLineID), args.Error(1)
}

func (m *mockStripeClient) UpdateInvoiceLines(ctx context.Context, input stripeclient.UpdateInvoiceLinesInput) ([]*stripe.InvoiceItem, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*stripe.InvoiceItem), args.Error(1)
}

func (m *mockStripeClient) RemoveInvoiceLines(ctx context.Context, input stripeclient.RemoveInvoiceLinesInput) error {
	return m.Called(ctx, input).Error(0)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestExecuteNextOperation_AllDone(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusExecuting,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return((*SyncOperation)(nil), nil)
	adapter.On("CompletePlan", ctx, "plan-1").Return(nil)

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.True(t, result.Done)
	assert.False(t, result.Failed)
	adapter.AssertExpectations(t)
}

func TestExecuteNextOperation_InvoiceCreate(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceCreatePayload{
		AppID:               "app-1",
		Namespace:           "ns-test",
		CustomerID:          "cust-1",
		InvoiceID:           "inv-1",
		AutomaticTaxEnabled: true,
		CollectionMethod:    "charge_automatically",
		Currency:            "USD",
		StripeCustomerID:    "cus_stripe",
	})

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusExecuting,
	}

	op := &SyncOperation{
		ID:             "op-1",
		PlanID:         "plan-1",
		Sequence:       0,
		Type:           OpTypeInvoiceCreate,
		Payload:        payload,
		IdempotencyKey: "key-1",
		Status:         OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)

	stripeClient.On("CreateInvoice", ctx, mock.MatchedBy(func(input stripeclient.CreateInvoiceInput) bool {
		return input.StripeCustomerID == "cus_stripe" && input.InvoiceID == "inv-1"
	})).Return(&stripe.Invoice{
		ID:     "in_stripe_new",
		Number: "INV-001",
	}, nil)

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)
	assert.False(t, result.Failed)

	adapter.AssertExpectations(t)
	stripeClient.AssertExpectations(t)

	// Verify the response was stored correctly
	completeCall := adapter.Calls[1]
	responseBytes := completeCall.Arguments[2].(json.RawMessage)
	var resp InvoiceCreateResponse
	require.NoError(t, json.Unmarshal(responseBytes, &resp))
	assert.Equal(t, "in_stripe_new", resp.StripeInvoiceID)
	assert.Equal(t, "INV-001", resp.InvoiceNumber)
}

func TestExecuteNextOperation_InvoiceFinalize(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceFinalizePayload{
		StripeInvoiceID: "in_stripe_123",
		AutoAdvance:     true,
		TaxEnforced:     false,
	})

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusExecuting,
	}

	op := &SyncOperation{
		ID:       "op-1",
		PlanID:   "plan-1",
		Sequence: 0,
		Type:     OpTypeInvoiceFinalize,
		Payload:  payload,
		Status:   OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)

	stripeClient.On("FinalizeInvoice", ctx, mock.MatchedBy(func(input stripeclient.FinalizeInvoiceInput) bool {
		return input.StripeInvoiceID == "in_stripe_123" && input.AutoAdvance
	})).Return(&stripe.Invoice{
		Number: "INV-001",
		PaymentIntent: &stripe.PaymentIntent{
			ID: "pi_123",
		},
	}, nil)

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)

	// Verify finalize response
	completeCall := adapter.Calls[1]
	responseBytes := completeCall.Arguments[2].(json.RawMessage)
	var resp InvoiceFinalizeResponse
	require.NoError(t, json.Unmarshal(responseBytes, &resp))
	assert.Equal(t, "INV-001", resp.InvoiceNumber)
	require.NotNil(t, resp.PaymentExternalID)
	assert.Equal(t, "pi_123", *resp.PaymentExternalID)
}

func TestExecuteNextOperation_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceUpdatePayload{
		StripeInvoiceID:     "in_stripe_123",
		AutomaticTaxEnabled: true,
	})

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusExecuting,
	}

	op := &SyncOperation{
		ID:       "op-1",
		PlanID:   "plan-1",
		Sequence: 0,
		Type:     OpTypeInvoiceUpdate,
		Payload:  payload,
		Status:   OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("FailOperation", ctx, "op-1", mock.Anything).Return(nil)
	adapter.On("FailPlan", ctx, "plan-1", mock.Anything).Return(nil)

	// Return a 400 error (non-retryable)
	stripeClient.On("UpdateInvoice", ctx, mock.Anything).Return((*stripe.Invoice)(nil), &stripe.Error{
		HTTPStatusCode: 400,
		Code:           "invalid_request",
		Msg:            "bad request",
	})

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err) // no error returned for non-retryable — it's handled
	assert.True(t, result.Done)
	assert.True(t, result.Failed)
	assert.NotEmpty(t, result.FailError)

	adapter.AssertExpectations(t)
}

func TestExecuteNextOperation_RetryableError(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceUpdatePayload{
		StripeInvoiceID:     "in_stripe_123",
		AutomaticTaxEnabled: true,
	})

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusExecuting,
	}

	op := &SyncOperation{
		ID:       "op-1",
		PlanID:   "plan-1",
		Sequence: 0,
		Type:     OpTypeInvoiceUpdate,
		Payload:  payload,
		Status:   OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)

	// Return a 429 error (retryable)
	stripeClient.On("UpdateInvoice", ctx, mock.Anything).Return((*stripe.Invoice)(nil), &stripe.Error{
		HTTPStatusCode: 429,
		Msg:            "rate limited",
	})

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	_, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.Error(t, err, "retryable errors should be returned for Watermill to retry")
	assert.Contains(t, err.Error(), "retryable error")
}

func TestExecuteNextOperation_StatusTransition(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusPending, // starts as pending
	}

	adapter.On("UpdatePlanStatus", ctx, "plan-1", PlanStatusExecuting, (*string)(nil)).Return(nil)
	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return((*SyncOperation)(nil), nil)
	adapter.On("CompletePlan", ctx, "plan-1").Return(nil)

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.True(t, result.Done)

	// Should have called UpdatePlanStatus to transition to executing
	adapter.AssertCalled(t, "UpdatePlanStatus", ctx, "plan-1", PlanStatusExecuting, (*string)(nil))
}

func TestExecuteNextOperation_ResolveStripeInvoiceIDFromCreate(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	linePayload, _ := json.Marshal(LineItemAddPayload{
		StripeInvoiceID: "", // needs to be resolved from InvoiceCreate response
		Lines: []LineItemParams{
			{
				Description: "Test line",
				Amount:      1000,
				Currency:    "USD",
				CustomerID:  "cus_123",
				PeriodStart: 1704067200,
				PeriodEnd:   1706745600,
				Metadata:    map[string]string{LineMetadataID: "dl-1", LineMetadataType: LineMetadataTypeLine},
			},
		},
	})

	createResponse, _ := json.Marshal(InvoiceCreateResponse{
		StripeInvoiceID: "in_from_create",
		InvoiceNumber:   "INV-001",
	})

	plan := &SyncPlan{
		ID:     "plan-1",
		Status: PlanStatusExecuting,
		Operations: []SyncOperation{
			{
				ID:             "op-create",
				Type:           OpTypeInvoiceCreate,
				Status:         OpStatusCompleted,
				StripeResponse: createResponse,
			},
		},
	}

	op := &SyncOperation{
		ID:       "op-add",
		PlanID:   "plan-1",
		Sequence: 1,
		Type:     OpTypeLineItemAdd,
		Payload:  linePayload,
		Status:   OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-add", mock.Anything).Return(nil)

	stripeClient.On("AddInvoiceLines", ctx, mock.MatchedBy(func(input stripeclient.AddInvoiceLinesInput) bool {
		return input.StripeInvoiceID == "in_from_create" && len(input.Lines) == 1
	})).Return([]stripeclient.StripeInvoiceItemWithLineID{
		{
			InvoiceItem: &stripe.InvoiceItem{
				ID:       "ii_new",
				Metadata: map[string]string{LineMetadataID: "dl-1", LineMetadataType: LineMetadataTypeLine},
			},
			LineID: "il_new",
		},
	}, nil)

	executor := &Executor{
		Adapter: adapter,
		Logger:  testLogger(),
	}

	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)

	// Verify the add lines used the resolved invoice ID
	stripeClient.AssertExpectations(t)
}

func TestBuildUpsertResultFromPlan(t *testing.T) {
	createResp, _ := json.Marshal(InvoiceCreateResponse{
		StripeInvoiceID: "in_123",
		InvoiceNumber:   "INV-001",
	})
	addResp, _ := json.Marshal(LineItemAddResponse{
		LineExternalIDs:         map[string]string{"dl-1": "il_1", "dl-2": "il_2"},
		LineDiscountExternalIDs: map[string]string{"disc-1": "il_disc_1"},
	})

	plan := &SyncPlan{
		Operations: []SyncOperation{
			{Type: OpTypeInvoiceCreate, Status: OpStatusCompleted, StripeResponse: createResp},
			{Type: OpTypeLineItemAdd, Status: OpStatusCompleted, StripeResponse: addResp},
		},
	}

	result, err := BuildUpsertResultFromPlan(plan)
	require.NoError(t, err)

	externalID, ok := result.GetExternalID()
	assert.True(t, ok)
	assert.Equal(t, "in_123", externalID)

	number, ok := result.GetInvoiceNumber()
	assert.True(t, ok)
	assert.Equal(t, "INV-001", number)

	lineID, ok := result.GetLineExternalID("dl-1")
	assert.True(t, ok)
	assert.Equal(t, "il_1", lineID)

	discountID, ok := result.GetLineDiscountExternalID("disc-1")
	assert.True(t, ok)
	assert.Equal(t, "il_disc_1", discountID)
}

func TestBuildFinalizeResultFromPlan(t *testing.T) {
	finalizeResp, _ := json.Marshal(InvoiceFinalizeResponse{
		InvoiceNumber:     "INV-001-FINAL",
		PaymentExternalID: strPtr("pi_payment_123"),
	})

	plan := &SyncPlan{
		Operations: []SyncOperation{
			{Type: OpTypeInvoiceFinalize, Status: OpStatusCompleted, StripeResponse: finalizeResp},
		},
	}

	result, err := BuildFinalizeResultFromPlan(plan)
	require.NoError(t, err)

	number, ok := result.GetInvoiceNumber()
	assert.True(t, ok)
	assert.Equal(t, "INV-001-FINAL", number)

	paymentID, ok := result.GetPaymentExternalID()
	assert.True(t, ok)
	assert.Equal(t, "pi_payment_123", paymentID)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil error", nil, false},
		{"non-stripe error", fmt.Errorf("connection refused"), false},
		{"429 rate limit", &stripe.Error{HTTPStatusCode: 429}, true},
		{"500 internal", &stripe.Error{HTTPStatusCode: 500}, true},
		{"502 bad gateway", &stripe.Error{HTTPStatusCode: 502}, true},
		{"503 unavailable", &stripe.Error{HTTPStatusCode: 503}, true},
		{"504 timeout", &stripe.Error{HTTPStatusCode: 504}, true},
		{"400 bad request", &stripe.Error{HTTPStatusCode: 400}, false},
		{"402 payment required", &stripe.Error{HTTPStatusCode: 402}, false},
		{"404 not found", &stripe.Error{HTTPStatusCode: 404}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, isRetryableError(tt.err))
		})
	}
}

func TestExecuteNextOperation_InvoiceDelete(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceDeletePayload{
		StripeInvoiceID: "in_stripe_123",
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceDelete, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("DeleteInvoice", ctx, mock.MatchedBy(func(input stripeclient.DeleteInvoiceInput) bool {
		return input.StripeInvoiceID == "in_stripe_123"
	})).Return(nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)
	stripeClient.AssertExpectations(t)
}

func TestExecuteNextOperation_LineItemUpdate(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(LineItemUpdatePayload{
		StripeInvoiceID: "in_stripe_123",
		Lines: []LineItemUpdateParams{
			{
				ID: "il_1", Description: "Updated", Amount: 2000, Currency: "USD",
				PeriodStart: 1704067200, PeriodEnd: 1706745600,
				Metadata: map[string]string{LineMetadataID: "dl-1", LineMetadataType: LineMetadataTypeLine},
			},
		},
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeLineItemUpdate, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("UpdateInvoiceLines", ctx, mock.MatchedBy(func(input stripeclient.UpdateInvoiceLinesInput) bool {
		return input.StripeInvoiceID == "in_stripe_123" && len(input.Lines) == 1 && input.Lines[0].ID == "il_1"
	})).Return([]*stripe.InvoiceItem{{ID: "ii_1"}}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)
	stripeClient.AssertExpectations(t)
}

func TestExecuteNextOperation_LineItemRemove(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(LineItemRemovePayload{
		StripeInvoiceID: "in_stripe_123",
		LineIDs:         []string{"il_old_1", "il_old_2"},
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeLineItemRemove, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("RemoveInvoiceLines", ctx, mock.MatchedBy(func(input stripeclient.RemoveInvoiceLinesInput) bool {
		return input.StripeInvoiceID == "in_stripe_123" && len(input.Lines) == 2
	})).Return(nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)
	stripeClient.AssertExpectations(t)
}

func TestExecuteNextOperation_InvoiceFinalize_TaxRetry(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceFinalizePayload{
		StripeInvoiceID: "in_stripe_123",
		AutoAdvance:     true,
		TaxEnforced:     false, // not enforced, so tax retry should work
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceFinalize, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)

	// First finalize fails with tax location error
	taxErr := stripeclient.NewStripeInvoiceCustomerTaxLocationInvalidError("in_stripe_123", "bad location")
	stripeClient.On("FinalizeInvoice", ctx, mock.Anything).Return((*stripe.Invoice)(nil), taxErr).Once()

	// Disable tax
	stripeClient.On("UpdateInvoice", ctx, mock.MatchedBy(func(input stripeclient.UpdateInvoiceInput) bool {
		return !input.AutomaticTaxEnabled && input.StripeInvoiceID == "in_stripe_123"
	})).Return(&stripe.Invoice{ID: "in_stripe_123"}, nil)

	// Second finalize succeeds
	stripeClient.On("FinalizeInvoice", ctx, mock.Anything).Return(&stripe.Invoice{
		Number:        "INV-002",
		PaymentIntent: &stripe.PaymentIntent{ID: "pi_abc"},
	}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)

	// Verify the response includes the finalized invoice data
	completeCall := adapter.Calls[1]
	responseBytes := completeCall.Arguments[2].(json.RawMessage)
	var resp InvoiceFinalizeResponse
	require.NoError(t, json.Unmarshal(responseBytes, &resp))
	assert.Equal(t, "INV-002", resp.InvoiceNumber)
	require.NotNil(t, resp.PaymentExternalID)
	assert.Equal(t, "pi_abc", *resp.PaymentExternalID)

	stripeClient.AssertExpectations(t)
}

func TestExecuteNextOperation_InvoiceFinalize_TaxEnforcedFails(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceFinalizePayload{
		StripeInvoiceID: "in_stripe_123",
		AutoAdvance:     true,
		TaxEnforced:     true, // enforced — should fail, not retry
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceFinalize, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("FailOperation", ctx, "op-1", mock.Anything).Return(nil)
	adapter.On("FailPlan", ctx, "plan-1", mock.Anything).Return(nil)

	taxErr := stripeclient.NewStripeInvoiceCustomerTaxLocationInvalidError("in_stripe_123", "bad location")
	stripeClient.On("FinalizeInvoice", ctx, mock.Anything).Return((*stripe.Invoice)(nil), taxErr)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err) // non-retryable, handled inline
	assert.True(t, result.Done)
	assert.True(t, result.Failed)
	assert.Contains(t, result.FailError, "tax enforced")
}

func TestExecuteNextOperation_InvoiceUpdate(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceUpdatePayload{
		StripeInvoiceID:     "in_stripe_123",
		AutomaticTaxEnabled: true,
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceUpdate, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("UpdateInvoice", ctx, mock.MatchedBy(func(input stripeclient.UpdateInvoiceInput) bool {
		return input.StripeInvoiceID == "in_stripe_123" && input.AutomaticTaxEnabled
	})).Return(&stripe.Invoice{ID: "in_stripe_123", Number: "INV-UPD"}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)

	completeCall := adapter.Calls[1]
	responseBytes := completeCall.Arguments[2].(json.RawMessage)
	var resp InvoiceUpdateResponse
	require.NoError(t, json.Unmarshal(responseBytes, &resp))
	assert.Equal(t, "in_stripe_123", resp.StripeInvoiceID)
	assert.Equal(t, "INV-UPD", resp.InvoiceNumber)
}

func TestExecuteNextOperation_InvoiceCreate_PopulatesExternalIDs(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceCreatePayload{
		AppID: "app-1", Namespace: "ns", CustomerID: "c-1", InvoiceID: "inv-1",
		CollectionMethod: "charge_automatically", Currency: "USD", StripeCustomerID: "cus_1",
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceCreate, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("CreateInvoice", ctx, mock.Anything).Return(&stripe.Invoice{
		ID: "in_new", Number: "INV-001",
	}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)

	// Verify external IDs are populated on the result
	require.NotNil(t, result.InvoicingExternalID)
	assert.Equal(t, "in_new", *result.InvoicingExternalID)
	assert.Nil(t, result.LineExternalIDs)
}

func TestExecuteNextOperation_LineItemAdd_PopulatesExternalIDs(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(LineItemAddPayload{
		StripeInvoiceID: "in_123",
		Lines: []LineItemParams{
			{
				Description: "Line 1", Amount: 1000, Currency: "USD", CustomerID: "cus_1",
				PeriodStart: 1704067200, PeriodEnd: 1706745600,
				Metadata: map[string]string{LineMetadataID: "dl-1", LineMetadataType: LineMetadataTypeLine},
			},
			{
				Description: "Discount", Amount: -100, Currency: "USD", CustomerID: "cus_1",
				PeriodStart: 1704067200, PeriodEnd: 1706745600,
				Metadata: map[string]string{LineMetadataID: "disc-1", LineMetadataType: LineMetadataTypeDiscount},
			},
		},
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 1,
		Type: OpTypeLineItemAdd, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("AddInvoiceLines", ctx, mock.Anything).Return([]stripeclient.StripeInvoiceItemWithLineID{
		{InvoiceItem: &stripe.InvoiceItem{ID: "ii_1", Metadata: map[string]string{LineMetadataID: "dl-1", LineMetadataType: LineMetadataTypeLine}}, LineID: "il_1"},
		{InvoiceItem: &stripe.InvoiceItem{ID: "ii_2", Metadata: map[string]string{LineMetadataID: "disc-1", LineMetadataType: LineMetadataTypeDiscount}}, LineID: "il_2"},
	}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)

	// Verify line external IDs
	assert.Nil(t, result.InvoicingExternalID)
	require.Len(t, result.LineExternalIDs, 1)
	assert.Equal(t, "il_1", result.LineExternalIDs["dl-1"])
	require.Len(t, result.LineDiscountExternalIDs, 1)
	assert.Equal(t, "il_2", result.LineDiscountExternalIDs["disc-1"])
}

func TestExecuteNextOperation_InvoiceUpdate_NoExternalIDs(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceUpdatePayload{
		StripeInvoiceID: "in_123", AutomaticTaxEnabled: true,
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceUpdate, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("UpdateInvoice", ctx, mock.Anything).Return(&stripe.Invoice{ID: "in_123", Number: "INV-1"}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)

	// InvoiceUpdate should NOT populate external IDs on the result
	assert.Nil(t, result.InvoicingExternalID)
	assert.Nil(t, result.LineExternalIDs)
	assert.Nil(t, result.LineDiscountExternalIDs)
}

func TestExecuteNextOperation_InvoiceCreate_WithDefaultPayment(t *testing.T) {
	ctx := context.Background()
	adapter := &mockAdapter{}
	stripeClient := &mockStripeClient{}

	payload, _ := json.Marshal(InvoiceCreatePayload{
		AppID: "app-1", Namespace: "ns", CustomerID: "c-1", InvoiceID: "inv-1",
		CollectionMethod: "charge_automatically", Currency: "USD",
		StripeCustomerID: "cus_1", StripeDefaultPaymentMethodID: "pm_default",
	})

	plan := &SyncPlan{ID: "plan-1", Status: PlanStatusExecuting}
	op := &SyncOperation{
		ID: "op-1", PlanID: "plan-1", Sequence: 0,
		Type: OpTypeInvoiceCreate, Payload: payload, Status: OpStatusPending,
	}

	adapter.On("GetNextPendingOperation", ctx, "plan-1").Return(op, nil)
	adapter.On("CompleteOperation", ctx, "op-1", mock.Anything).Return(nil)
	stripeClient.On("CreateInvoice", ctx, mock.MatchedBy(func(input stripeclient.CreateInvoiceInput) bool {
		return input.StripeDefaultPaymentMethodID != nil && *input.StripeDefaultPaymentMethodID == "pm_default"
	})).Return(&stripe.Invoice{ID: "in_new", Number: "INV-001"}, nil)

	executor := &Executor{Adapter: adapter, Logger: testLogger()}
	result, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
	require.NoError(t, err)
	assert.False(t, result.Done)
	stripeClient.AssertExpectations(t)
}
