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
// Only CreatePlan is implemented; all other methods panic.
type stubPlanService struct {
	createPlan func(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error)
}

func (s *stubPlanService) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
	return s.createPlan(ctx, params)
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

func (s *stubPlanService) UpdatePlan(_ context.Context, _ plan.UpdatePlanInput) (*plan.Plan, error) {
	panic("unexpected call to UpdatePlan")
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

func TestCreatePlanCreditConfiguration(t *testing.T) {
	const namespace = "test-ns"

	tests := []struct {
		name           string
		creditEnabled  bool
		wantStatusCode int
	}{
		{
			name:           "credit disabled: credit_only settlement mode is rejected",
			creditEnabled:  false,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "credit enabled: credit_only settlement mode is allowed",
			creditEnabled:  true,
			wantStatusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &stubPlanService{
				createPlan: func(_ context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
					return stubPlan(namespace, params.SettlementMode), nil
				},
			}

			h := httpdriver.New(
				namespacedriver.StaticNamespaceDecoder(namespace),
				svc,
			)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/plans", bytes.NewBufferString(planCreateCreditOnlyBody))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h.CreatePlan().With(appconfig.CreditConfiguration{Enabled: tt.creditEnabled}).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatusCode, rec.Code, "response body: %s", rec.Body.String())
			if !tt.creditEnabled {
				assert.Contains(t, rec.Body.String(), "credits are not enabled on this deployment of OpenMeter")
			}
		})
	}
}
