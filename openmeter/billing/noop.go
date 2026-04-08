package billing

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ Service = (*NoopService)(nil)

// NoopService implements billing.Service with no-op operations, useful for tests
// that need a billing.Service but don't exercise billing logic.
type NoopService struct{}

// ProfileService

func (n NoopService) CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error) {
	return &Profile{}, nil
}

func (n NoopService) GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*Profile, error) {
	return &Profile{}, nil
}

func (n NoopService) GetProfile(ctx context.Context, input GetProfileInput) (*Profile, error) {
	return &Profile{}, nil
}

func (n NoopService) ListProfiles(ctx context.Context, input ListProfilesInput) (ListProfilesResult, error) {
	return ListProfilesResult{}, nil
}

func (n NoopService) DeleteProfile(ctx context.Context, input DeleteProfileInput) error {
	return nil
}

func (n NoopService) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*Profile, error) {
	return &Profile{}, nil
}

func (n NoopService) ProvisionDefaultBillingProfile(ctx context.Context, namespace string) error {
	return nil
}

func (n NoopService) IsAppUsed(ctx context.Context, appID app.AppID) error {
	return nil
}

func (n NoopService) ResolveStripeAppIDFromBillingProfile(ctx context.Context, namespace string, customerId *customer.CustomerID) (app.AppID, error) {
	return app.AppID{}, nil
}

// CustomerOverrideService

func (n NoopService) UpsertCustomerOverride(ctx context.Context, input UpsertCustomerOverrideInput) (CustomerOverrideWithDetails, error) {
	return CustomerOverrideWithDetails{}, nil
}

func (n NoopService) DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error {
	return nil
}

func (n NoopService) GetCustomerOverride(ctx context.Context, input GetCustomerOverrideInput) (CustomerOverrideWithDetails, error) {
	return CustomerOverrideWithDetails{}, nil
}

func (n NoopService) GetCustomerApp(ctx context.Context, input GetCustomerAppInput) (app.App, error) {
	return nil, nil
}

func (n NoopService) ListCustomerOverrides(ctx context.Context, input ListCustomerOverridesInput) (ListCustomerOverridesResult, error) {
	return ListCustomerOverridesResult{}, nil
}

// InvoiceLineService

func (n NoopService) GetLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) ([]LineOrHierarchy, error) {
	return []LineOrHierarchy{}, nil
}

func (n NoopService) SnapshotLineQuantity(ctx context.Context, input SnapshotLineQuantityInput) (*StandardLine, error) {
	return &StandardLine{}, nil
}

// SplitLineGroupService

func (n NoopService) DeleteSplitLineGroup(ctx context.Context, input DeleteSplitLineGroupInput) error {
	return nil
}

func (n NoopService) UpdateSplitLineGroup(ctx context.Context, input UpdateSplitLineGroupInput) (SplitLineGroup, error) {
	return SplitLineGroup{}, nil
}

func (n NoopService) GetSplitLineGroup(ctx context.Context, input GetSplitLineGroupInput) (SplitLineHierarchy, error) {
	return SplitLineHierarchy{}, nil
}

// InvoiceService

func (n NoopService) ListInvoices(ctx context.Context, input ListInvoicesInput) (ListInvoicesResponse, error) {
	return ListInvoicesResponse{}, nil
}

func (n NoopService) GetInvoiceById(ctx context.Context, input GetInvoiceByIdInput) (Invoice, error) {
	return Invoice{}, nil
}

func (n NoopService) InvoicePendingLines(ctx context.Context, input InvoicePendingLinesInput) ([]StandardInvoice, error) {
	return []StandardInvoice{}, nil
}

func (n NoopService) AdvanceInvoice(ctx context.Context, input AdvanceInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) SnapshotQuantities(ctx context.Context, input SnapshotQuantitiesInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) ApproveInvoice(ctx context.Context, input ApproveInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) RetryInvoice(ctx context.Context, input RetryInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (Invoice, error) {
	return Invoice{}, nil
}

func (n NoopService) SimulateInvoice(ctx context.Context, input SimulateInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) UpsertValidationIssues(ctx context.Context, input UpsertValidationIssuesInput) error {
	return nil
}

func (n NoopService) RecalculateGatheringInvoices(ctx context.Context, input RecalculateGatheringInvoicesInput) error {
	return nil
}

// StandardInvoiceService

func (n NoopService) UpdateStandardInvoice(ctx context.Context, input UpdateStandardInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) GetStandardInvoiceById(ctx context.Context, input GetStandardInvoiceByIdInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) ListStandardInvoices(ctx context.Context, input ListStandardInvoicesInput) (ListStandardInvoicesResponse, error) {
	return ListStandardInvoicesResponse{}, nil
}

func (n NoopService) CreateStandardInvoiceFromGatheringLines(ctx context.Context, input CreateStandardInvoiceFromGatheringLinesInput) (*StandardInvoice, error) {
	return &StandardInvoice{}, nil
}

func (n NoopService) RegisterStandardInvoiceHooks(hooks ...StandardInvoiceHook) {}

// GatheringInvoiceService

func (n NoopService) CreatePendingInvoiceLines(ctx context.Context, input CreatePendingInvoiceLinesInput) (*CreatePendingInvoiceLinesResult, error) {
	return &CreatePendingInvoiceLinesResult{}, nil
}

func (n NoopService) PrepareBillableLines(ctx context.Context, input PrepareBillableLinesInput) (*PrepareBillableLinesResult, error) {
	return &PrepareBillableLinesResult{}, nil
}

func (n NoopService) ListGatheringInvoices(ctx context.Context, input ListGatheringInvoicesInput) (pagination.Result[GatheringInvoice], error) {
	return pagination.Result[GatheringInvoice]{}, nil
}

func (n NoopService) GetGatheringInvoiceById(ctx context.Context, input GetGatheringInvoiceByIdInput) (GatheringInvoice, error) {
	return GatheringInvoice{}, nil
}

func (n NoopService) UpdateGatheringInvoice(ctx context.Context, input UpdateGatheringInvoiceInput) error {
	return nil
}

// SequenceService

func (n NoopService) GenerateInvoiceSequenceNumber(ctx context.Context, in SequenceGenerationInput, def SequenceDefinition) (string, error) {
	return "", nil
}

// InvoiceAppService

func (n NoopService) TriggerInvoice(ctx context.Context, input InvoiceTriggerServiceInput) error {
	return nil
}

func (n NoopService) UpdateInvoiceFields(ctx context.Context, input UpdateInvoiceFieldsInput) error {
	return nil
}

func (n NoopService) SyncDraftInvoice(ctx context.Context, input SyncDraftStandardInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) SyncIssuingInvoice(ctx context.Context, input SyncIssuingStandardInvoiceInput) (StandardInvoice, error) {
	return StandardInvoice{}, nil
}

func (n NoopService) SyncExternalIDs(ctx context.Context, input SyncExternalIDsInput) error {
	return nil
}

func (n NoopService) FailSyncInvoice(ctx context.Context, input FailSyncInvoiceInput) error {
	return nil
}

// ConfigService

func (n NoopService) GetAdvancementStrategy() AdvancementStrategy {
	return ForegroundAdvancementStrategy
}

func (n NoopService) WithAdvancementStrategy(strategy AdvancementStrategy) Service {
	return n
}

func (n NoopService) WithLockedNamespaces(namespaces []string) Service {
	return n
}

// LockableService

func (n NoopService) WithLock(ctx context.Context, customerID customer.CustomerID, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
