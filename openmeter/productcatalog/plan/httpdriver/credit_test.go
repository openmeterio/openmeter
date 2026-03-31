package httpdriver_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appconfig "github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	httpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// stubPlanService is a minimal plan.Service for handler tests.
// Only CreatePlan and UpdatePlan are implemented; all other methods panic.
type stubPlanService struct {
	createPlan func(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error)
	updatePlan func(ctx context.Context, params plan.UpdatePlanInput) (*plan.Plan, error)
}

func (s *stubPlanService) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
	return s.createPlan(ctx, params)
}

func (s *stubPlanService) UpdatePlan(ctx context.Context, params plan.UpdatePlanInput) (*plan.Plan, error) {
	return s.updatePlan(ctx, params)
}

func (s *stubPlanService) ListPlans(_ context.Context, _ plan.ListPlansInput) (pagination.Result[plan.Plan], error) {
	panic("unexpected call to ListPlans")
}

func (s *stubPlanService) DeletePlan(_ context.Context, _ plan.DeletePlanInput) error {
	panic("unexpected call to DeletePlan")
}

func (s *stubPlanService) GetPlan(_ context.Context, _ plan.GetPlanInput) (*plan.Plan, error) {
	panic("unexpected call to GetPlan")
}

func (s *stubPlanService) PublishPlan(_ context.Context, _ plan.PublishPlanInput) (*plan.Plan, error) {
	panic("unexpected call to PublishPlan")
}

func (s *stubPlanService) ArchivePlan(_ context.Context, _ plan.ArchivePlanInput) (*plan.Plan, error) {
	panic("unexpected call to ArchivePlan")
}

func (s *stubPlanService) NextPlan(_ context.Context, _ plan.NextPlanInput) (*plan.Plan, error) {
	panic("unexpected call to NextPlan")
}

// stubPlan returns a minimal plan with the given settlement mode, suitable as a service response.
func stubPlan(ns string, settlementMode productcatalog.SettlementMode) *plan.Plan {
	return &plan.Plan{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: "test-plan-id"},
		ManagedModel: models.ManagedModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PlanMeta: productcatalog.PlanMeta{
			Key:            "test-plan",
			Name:           "Test Plan",
			SettlementMode: settlementMode,
		},
	}
}

// planCreateCreditOnlyBody is a minimal valid PlanCreate JSON with credit_only settlement mode.
const planCreateCreditOnlyBody = `{
	"key": "test-plan",
	"name": "Test Plan",
	"currency": "USD",
	"billingCadence": "P1M",
	"settlementMode": "credit_only",
	"phases": [{"key": "default", "name": "Default Phase", "rateCards": []}]
}`

// planCreateCreditThenInvoiceBody is a minimal valid PlanCreate JSON with credit_then_invoice settlement mode.
const planCreateCreditThenInvoiceBody = `{
	"key": "test-plan",
	"name": "Test Plan",
	"currency": "USD",
	"billingCadence": "P1M",
	"settlementMode": "credit_then_invoice",
	"phases": [{"key": "default", "name": "Default Phase", "rateCards": []}]
}`

// planUpdateCreditOnlyBody is a minimal valid PlanReplaceUpdate JSON with credit_only settlement mode.
const planUpdateCreditOnlyBody = `{
	"name": "Test Plan Updated",
	"settlementMode": "credit_only",
	"phases": [{"key": "default", "name": "Default Phase", "rateCards": []}]
}`

// planUpdateCreditThenInvoiceBody is a minimal valid PlanReplaceUpdate JSON with credit_then_invoice settlement mode.
const planUpdateCreditThenInvoiceBody = `{
	"name": "Test Plan Updated",
	"settlementMode": "credit_then_invoice",
	"phases": [{"key": "default", "name": "Default Phase", "rateCards": []}]
}`

func TestCreatePlanCreditConfiguration(t *testing.T) {
	const namespace = "test-ns"

	tests := []struct {
		name           string
		body           string
		creditEnabled  bool
		wantStatusCode int
	}{
		{
			name:           "credit disabled: credit_only settlement mode is rejected",
			body:           planCreateCreditOnlyBody,
			creditEnabled:  false,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "credit enabled: credit_only settlement mode is allowed",
			body:           planCreateCreditOnlyBody,
			creditEnabled:  true,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "credit disabled: non-credit settlement mode is allowed",
			body:           planCreateCreditThenInvoiceBody,
			creditEnabled:  false,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "credit enabled: non-credit settlement mode is allowed",
			body:           planCreateCreditThenInvoiceBody,
			creditEnabled:  true,
			wantStatusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createPlanCalled bool
			svc := &stubPlanService{
				createPlan: func(_ context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
					createPlanCalled = true
					return stubPlan(namespace, params.SettlementMode), nil
				},
			}

			h := httpdriver.New(
				namespacedriver.StaticNamespaceDecoder(namespace),
				svc,
				appconfig.CreditsConfiguration{Enabled: tt.creditEnabled},
			)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/plans", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h.CreatePlan().ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatusCode, rec.Code, "response body: %s", rec.Body.String())
			if tt.wantStatusCode == http.StatusBadRequest {
				assert.Contains(t, rec.Body.String(), "credits are not enabled on this deployment of OpenMeter")
				assert.False(t, createPlanCalled, "service must not be called when credit check rejects the request")
			} else {
				assert.True(t, createPlanCalled, "service must be called when credit check passes")
			}
		})
	}
}

func TestUpdatePlanCreditConfiguration(t *testing.T) {
	const namespace = "test-ns"

	tests := []struct {
		name           string
		body           string
		creditEnabled  bool
		wantStatusCode int
	}{
		{
			name:           "credit disabled: credit_only settlement mode is rejected",
			body:           planUpdateCreditOnlyBody,
			creditEnabled:  false,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "credit enabled: credit_only settlement mode is allowed",
			body:           planUpdateCreditOnlyBody,
			creditEnabled:  true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "credit disabled: non-credit settlement mode is allowed",
			body:           planUpdateCreditThenInvoiceBody,
			creditEnabled:  false,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "credit enabled: non-credit settlement mode is allowed",
			body:           planUpdateCreditThenInvoiceBody,
			creditEnabled:  true,
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updatePlanCalled bool
			svc := &stubPlanService{
				updatePlan: func(_ context.Context, params plan.UpdatePlanInput) (*plan.Plan, error) {
					updatePlanCalled = true
					mode := productcatalog.CreditThenInvoiceSettlementMode
					if params.SettlementMode != nil {
						mode = *params.SettlementMode
					}
					return stubPlan(namespace, mode), nil
				},
			}

			h := httpdriver.New(
				namespacedriver.StaticNamespaceDecoder(namespace),
				svc,
				appconfig.CreditsConfiguration{Enabled: tt.creditEnabled},
			)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/plans/test-plan-id", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h.UpdatePlan().With("test-plan-id").ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatusCode, rec.Code, "response body: %s", rec.Body.String())
			if tt.wantStatusCode == http.StatusBadRequest {
				assert.Contains(t, rec.Body.String(), "credits are not enabled on this deployment of OpenMeter")
				assert.False(t, updatePlanCalled, "service must not be called when credit check rejects the request")
			} else {
				assert.True(t, updatePlanCalled, "service must be called when credit check passes")
			}
		})
	}
}
