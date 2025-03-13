package appsandbox

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
)

type AppFactory interface {
	NewApp(ctx context.Context, appBase app.AppBase) (app.App, error)
}

type InvoiceUpsertCallback func(billing.Invoice) *billing.UpsertInvoiceResult

type MockApp struct {
	validateCustomerResponse mo.Option[error]
	validateCustomerCalled   bool

	validateInvoiceResponse       mo.Option[error]
	validateInvoiceResponseCalled bool

	upsertInvoiceCallback mo.Option[InvoiceUpsertCallback]
	upsertInvoiceCalled   bool

	finalizeInvoiceResponse mo.Option[*billing.FinalizeInvoiceResult]
	finalizeInvoiceCalled   bool

	deleteInvoiceResponse mo.Option[error]
	deleteInvoiceCalled   bool
}

func NewMockApp(_ *testing.T) *MockApp {
	return &MockApp{}
}

func (m *MockApp) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return nil, nil
}

func (m *MockApp) UpsertCustomerData(ctx context.Context, input app.UpsertAppInstanceCustomerDataInput) error {
	return nil
}

func (m *MockApp) DeleteCustomerData(ctx context.Context, input app.DeleteAppInstanceCustomerDataInput) error {
	return nil
}

func (m *MockApp) ValidateCustomer(appID string, customer *customer.Customer, capabilities []app.CapabilityType) error {
	m.validateCustomerCalled = true
	return m.validateCustomerResponse.MustGet()
}

func (m *MockApp) OnValidateCustomer(err error) {
	m.validateCustomerResponse = mo.Some(err)
}

// InvoicingApp

func (m *MockApp) ValidateInvoice(appID string, invoice billing.Invoice) error {
	m.validateInvoiceResponseCalled = true
	return m.validateInvoiceResponse.MustGet()
}

func (m *MockApp) OnValidateInvoice(err error) {
	m.validateInvoiceResponse = mo.Some(err)
}

func (m *MockApp) UpsertInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	m.upsertInvoiceCalled = true

	if m.upsertInvoiceCallback.IsPresent() && m.upsertInvoiceCallback.MustGet() != nil {
		return m.upsertInvoiceCallback.MustGet()(invoice), nil
	}

	return billing.NewUpsertInvoiceResult(), nil
}

func (m *MockApp) OnUpsertInvoice(cb InvoiceUpsertCallback) {
	m.upsertInvoiceCallback = mo.Some(cb)
}

func (m *MockApp) FinalizeInvoice(ctx context.Context, invoice billing.Invoice) (*billing.FinalizeInvoiceResult, error) {
	m.finalizeInvoiceCalled = true
	return m.finalizeInvoiceResponse.MustGet(), nil
}

func (m *MockApp) OnFinalizeInvoice(result *billing.FinalizeInvoiceResult) {
	m.finalizeInvoiceResponse = mo.Some(result)
}

func (m *MockApp) DeleteInvoice(ctx context.Context, invoice billing.Invoice) error {
	m.deleteInvoiceCalled = true
	return m.deleteInvoiceResponse.MustGet()
}

func (m *MockApp) OnDeleteInvoice(err error) {
	m.deleteInvoiceResponse = mo.Some(err)
}

func (m *MockApp) Reset(t *testing.T) {
	t.Helper()

	m.AssertExpectations(t)

	m.validateCustomerResponse = mo.None[error]()
	m.validateCustomerCalled = false

	m.validateInvoiceResponse = mo.None[error]()
	m.validateInvoiceResponseCalled = false

	m.upsertInvoiceCallback = mo.None[InvoiceUpsertCallback]()
	m.upsertInvoiceCalled = false

	m.finalizeInvoiceResponse = mo.None[*billing.FinalizeInvoiceResult]()
	m.finalizeInvoiceCalled = false

	m.deleteInvoiceResponse = mo.None[error]()
	m.deleteInvoiceCalled = false
}

func (m *MockApp) AssertExpectations(t *testing.T) {
	t.Helper()

	if m.validateCustomerResponse.IsPresent() && !m.validateCustomerCalled {
		t.Errorf("expected ValidateCustomer to be called")
	}

	if m.validateInvoiceResponse.IsPresent() && !m.validateInvoiceResponseCalled {
		t.Errorf("expected ValidateInvoice to be called")
	}

	if m.upsertInvoiceCallback.IsPresent() && !m.upsertInvoiceCalled {
		t.Errorf("expected UpsertInvoice to be called")
	}

	if m.finalizeInvoiceResponse.IsPresent() && !m.finalizeInvoiceCalled {
		t.Errorf("expected FinalizeInvoice to be called")
	}

	if m.deleteInvoiceResponse.IsPresent() && !m.deleteInvoiceCalled {
		t.Errorf("expected DeleteInvoice to be called")
	}
}

func (m *MockApp) NewApp(_ context.Context, app app.AppBase) (app.App, error) {
	return &mockAppInstance{
		AppBase: app,
		parent:  m,
	}, nil
}

type mockAppInstance struct {
	app.AppBase

	parent *MockApp
}

var (
	_ billing.InvoicingApp = (*mockAppInstance)(nil)
	_ customerapp.App      = (*mockAppInstance)(nil)
)

func (m *mockAppInstance) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return m.parent.GetCustomerData(ctx, input)
}

func (m *mockAppInstance) UpsertCustomerData(ctx context.Context, input app.UpsertAppInstanceCustomerDataInput) error {
	return m.parent.UpsertCustomerData(ctx, input)
}

func (m *mockAppInstance) DeleteCustomerData(ctx context.Context, input app.DeleteAppInstanceCustomerDataInput) error {
	return m.parent.DeleteCustomerData(ctx, input)
}

func (m *mockAppInstance) ValidateCustomer(ctx context.Context, customer *customer.Customer, capabilities []app.CapabilityType) error {
	return m.parent.ValidateCustomer(m.GetID().ID, customer, capabilities)
}

func (m *mockAppInstance) ValidateInvoice(ctx context.Context, invoice billing.Invoice) error {
	return m.parent.ValidateInvoice(m.GetID().ID, invoice)
}

func (m *mockAppInstance) UpsertInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	return m.parent.UpsertInvoice(ctx, invoice)
}

func (m *mockAppInstance) FinalizeInvoice(ctx context.Context, invoice billing.Invoice) (*billing.FinalizeInvoiceResult, error) {
	return m.parent.FinalizeInvoice(ctx, invoice)
}

func (m *mockAppInstance) DeleteInvoice(ctx context.Context, invoice billing.Invoice) error {
	return m.parent.DeleteInvoice(ctx, invoice)
}

type MockableFactory struct {
	*Factory

	overrideFactory AppFactory
}

type mockConfig struct {
	OverrideType app.AppType
}

type mockConfigOption func(*mockConfig)

func MockWithAppType(t app.AppType) mockConfigOption {
	return func(c *mockConfig) {
		c.OverrideType = t
	}
}

func NewMockableFactory(_ *testing.T, config Config, opts ...mockConfigOption) (*MockableFactory, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	fact := &MockableFactory{
		Factory: &Factory{
			appService:     config.AppService,
			billingService: config.BillingService,
		},
	}

	mockConfig := &mockConfig{}
	for _, opt := range opts {
		opt(mockConfig)
	}

	listing := MarketplaceListing

	if mockConfig.OverrideType != "" {
		listing.Type = mockConfig.OverrideType
	}

	err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{
		Listing: listing,
		Factory: fact,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register marketplace listing: %w", err)
	}

	return fact, nil
}

func (m *MockableFactory) NewApp(ctx context.Context, appBase app.AppBase) (app.App, error) {
	if m.overrideFactory != nil {
		return m.overrideFactory.NewApp(ctx, appBase)
	}

	return m.Factory.NewApp(ctx, appBase)
}

func (m *MockableFactory) EnableMock(t *testing.T) *MockApp {
	mock := NewMockApp(t)

	m.overrideFactory = mock

	return mock
}

func (m *MockableFactory) DisableMock() {
	m.overrideFactory = nil
}
