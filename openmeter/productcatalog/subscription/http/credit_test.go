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
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	httpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// stubCustomerService is a minimal customer.Service for handler tests.
// Only GetCustomer is implemented; all other methods panic.
type stubCustomerService struct {
	getCustomer func(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error)
}

func (s *stubCustomerService) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return s.getCustomer(ctx, input)
}

func (s *stubCustomerService) ListCustomers(_ context.Context, _ customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	panic("unexpected call to ListCustomers")
}

func (s *stubCustomerService) ListCustomerUsageAttributions(_ context.Context, _ customer.ListCustomerUsageAttributionsInput) (pagination.Result[streaming.CustomerUsageAttribution], error) {
	panic("unexpected call to ListCustomerUsageAttributions")
}

func (s *stubCustomerService) CreateCustomer(_ context.Context, _ customer.CreateCustomerInput) (*customer.Customer, error) {
	panic("unexpected call to CreateCustomer")
}

func (s *stubCustomerService) DeleteCustomer(_ context.Context, _ customer.DeleteCustomerInput) error {
	panic("unexpected call to DeleteCustomer")
}

func (s *stubCustomerService) GetCustomerByUsageAttribution(_ context.Context, _ customer.GetCustomerByUsageAttributionInput) (*customer.Customer, error) {
	panic("unexpected call to GetCustomerByUsageAttribution")
}

func (s *stubCustomerService) UpdateCustomer(_ context.Context, _ customer.UpdateCustomerInput) (*customer.Customer, error) {
	panic("unexpected call to UpdateCustomer")
}

func (s *stubCustomerService) RegisterRequestValidator(_ customer.RequestValidator) {}

func (s *stubCustomerService) RegisterHooks(_ ...models.ServiceHook[customer.Customer]) {}

// stubPlanSubscriptionService is a minimal plansubscription.PlanSubscriptionService for handler tests.
type stubPlanSubscriptionService struct {
	create func(ctx context.Context, request plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error)
	change func(ctx context.Context, request plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error)
}

func (s *stubPlanSubscriptionService) Create(ctx context.Context, request plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error) {
	return s.create(ctx, request)
}

func (s *stubPlanSubscriptionService) Change(ctx context.Context, request plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	return s.change(ctx, request)
}

func (s *stubPlanSubscriptionService) Migrate(_ context.Context, _ plansubscription.MigrateSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	panic("unexpected call to Migrate")
}

// stubCustomer returns a minimal non-deleted customer suitable as a service response.
func stubCustomer(ns, id string) *customer.Customer {
	return &customer.Customer{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: ns},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ID: id,
		},
	}
}

// stubSubscription returns a minimal subscription suitable as a service response.
func stubSubscription(ns string) subscription.Subscription {
	billingCadence, _ := datetime.ISODurationString("P1M").Parse()
	return subscription.Subscription{
		NamespacedID: models.NamespacedID{ID: "test-sub-id", Namespace: ns},
		ManagedModel: models.ManagedModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		BillingCadence: billingCadence,
		SettlementMode: productcatalog.CreditOnlySettlementMode,
	}
}

// stubSubscriptionView returns a minimal subscription view suitable as a service response for change.
func stubSubscriptionView(ns string) subscription.SubscriptionView {
	sub := stubSubscription(ns)
	billingCadence, _ := datetime.ISODurationString("P1M").Parse()
	activeFrom := time.Now().Add(-24 * time.Hour)
	zeroDuration, _ := datetime.ISODurationString("P0D").Parse()

	return subscription.SubscriptionView{
		Subscription: sub,
		Spec: subscription.SubscriptionSpec{
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				BillingCadence: billingCadence,
			},
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				ActiveFrom:    activeFrom,
				BillingAnchor: activeFrom,
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"default": {
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey:   "default",
						Name:       "Default Phase",
						StartAfter: zeroDuration,
					},
				},
			},
		},
	}
}

// subscriptionCreateCreditOnlyBody is a minimal valid CreateSubscription JSON with credit_only settlement mode.
const subscriptionCreateCreditOnlyBody = `{
	"customerId": "test-customer-id",
	"customPlan": {
		"name": "Test Plan",
		"currency": "USD",
		"billingCadence": "P1M",
		"settlementMode": "credit_only",
		"phases": [{"key": "default", "name": "Default Phase", "rateCards": []}]
	}
}`

// subscriptionChangeCreditOnlyBody is a minimal valid ChangeSubscription JSON with credit_only settlement mode.
const subscriptionChangeCreditOnlyBody = `{
	"timing": "immediate",
	"customPlan": {
		"name": "Test Plan",
		"currency": "USD",
		"billingCadence": "P1M",
		"settlementMode": "credit_only",
		"phases": [{"key": "default", "name": "Default Phase", "rateCards": []}]
	}
}`

func TestCreateSubscriptionCreditConfiguration(t *testing.T) {
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
			customerSvc := &stubCustomerService{
				getCustomer: func(_ context.Context, _ customer.GetCustomerInput) (*customer.Customer, error) {
					return stubCustomer(namespace, "test-customer-id"), nil
				},
			}

			planSubSvc := &stubPlanSubscriptionService{
				create: func(_ context.Context, _ plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error) {
					return stubSubscription(namespace), nil
				},
			}

			h := httpdriver.NewHandler(httpdriver.HandlerConfig{
				NamespaceDecoder:        namespacedriver.StaticNamespaceDecoder(namespace),
				CustomerService:         customerSvc,
				PlanSubscriptionService: planSubSvc,
			})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBufferString(subscriptionCreateCreditOnlyBody))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h.CreateSubscription().With(appconfig.CreditConfiguration{Enabled: tt.creditEnabled}).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatusCode, rec.Code, "response body: %s", rec.Body.String())
			if !tt.creditEnabled {
				assert.Contains(t, rec.Body.String(), "credits are not enabled on this deployment of OpenMeter")
			}
		})
	}
}

func TestChangeSubscriptionCreditConfiguration(t *testing.T) {
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
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planSubSvc := &stubPlanSubscriptionService{
				change: func(_ context.Context, _ plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
					return plansubscription.SubscriptionChangeResponse{
						Current: stubSubscription(namespace),
						Next:    stubSubscriptionView(namespace),
					}, nil
				},
			}

			h := httpdriver.NewHandler(httpdriver.HandlerConfig{
				NamespaceDecoder:        namespacedriver.StaticNamespaceDecoder(namespace),
				PlanSubscriptionService: planSubSvc,
				Credit:                  appconfig.CreditConfiguration{Enabled: tt.creditEnabled},
			})

			req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/test-sub-id", bytes.NewBufferString(subscriptionChangeCreditOnlyBody))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			h.ChangeSubscription().With(httpdriver.ChangeSubscriptionParams{ID: "test-sub-id"}).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatusCode, rec.Code, "response body: %s", rec.Body.String())
			if !tt.creditEnabled {
				assert.Contains(t, rec.Body.String(), "credits are not enabled on this deployment of OpenMeter")
			}
		})
	}
}
