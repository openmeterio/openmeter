package billing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ChangeSource string

const (
	ChangeSourceSystem     ChangeSource = "system"
	ChangeSourceAPIRequest ChangeSource = "api_request"
)

func (ChangeSource) Values() []string {
	return []string{
		string(ChangeSourceSystem),
		string(ChangeSourceAPIRequest),
	}
}

func (i ChangeSource) Validate() error {
	if !slices.Contains(ChangeSource("").Values(), string(i)) {
		return fmt.Errorf("invalid change source: %s", i)
	}

	return nil
}

type LineEngineType string

const (
	LineEngineTypeInvoice              LineEngineType = "invoicing"
	LineEngineTypeChargeFlatFee        LineEngineType = "charge_flatfee"
	LineEngineTypeChargeUsageBased     LineEngineType = "charge_usagebased"
	LineEngineTypeChargeCreditPurchase LineEngineType = "charge_creditpurchase"
)

func (b LineEngineType) Values() []string {
	return []string{
		string(LineEngineTypeInvoice),
		string(LineEngineTypeChargeFlatFee),
		string(LineEngineTypeChargeUsageBased),
		string(LineEngineTypeChargeCreditPurchase),
	}
}

func (b LineEngineType) Validate() error {
	if !slices.Contains(b.Values(), string(b)) {
		return fmt.Errorf("invalid line engine type: %s", b)
	}

	return nil
}

func (b LineEngineType) IsCharge() bool {
	switch b {
	case LineEngineTypeChargeFlatFee, LineEngineTypeChargeUsageBased, LineEngineTypeChargeCreditPurchase:
		return true
	default:
		return false
	}
}

type LineBillability struct {
	IsBillable      bool
	ValidationError error
}

type LineBillabilities []LineBillability

type BuildStandardInvoiceLinesInput struct {
	// Invoice is the target standard invoice that will own the built lines.
	Invoice StandardInvoice
	// GatheringLines are the source lines already assigned to this engine.
	GatheringLines GatheringLines
}

func (i BuildStandardInvoiceLinesInput) Validate() error {
	var errs []error

	if i.Invoice.ID == "" {
		errs = append(errs, fmt.Errorf("invoice id is required"))
	}

	if len(i.GatheringLines) == 0 {
		errs = append(errs, fmt.Errorf("gathering lines are required"))
	}

	if err := i.GatheringLines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("gathering lines: %w", err))
	}

	return errors.Join(errs...)
}

type CalculateLinesInput struct {
	// Invoice is the standard invoice owning the lines being recalculated.
	Invoice StandardInvoice
	// Lines are the standard invoice lines already assigned to this engine.
	Lines StandardLines
}

func (i CalculateLinesInput) Validate() error {
	var errs []error

	if i.Invoice.ID == "" {
		errs = append(errs, fmt.Errorf("invoice id is required"))
	}

	if len(i.Lines) == 0 {
		errs = append(errs, fmt.Errorf("lines are required"))
	}

	if err := i.Lines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("lines: %w", err))
	}

	return errors.Join(errs...)
}

type StandardLineEventInput struct {
	// Invoice is the standard invoice whose lines are being processed for a lifecycle event.
	Invoice StandardInvoice
	// Lines are the standard invoice lines already assigned to this engine.
	Lines StandardLines
}

func (i StandardLineEventInput) Validate() error {
	var errs []error

	if i.Invoice.ID == "" {
		errs = append(errs, fmt.Errorf("invoice id is required"))
	}

	if len(i.Lines) == 0 {
		errs = append(errs, fmt.Errorf("lines are required"))
	}

	if err := i.Lines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("lines: %w", err))
	}

	return errors.Join(errs...)
}

type (
	OnStandardInvoiceCreatedInput      = StandardLineEventInput
	OnCollectionCompletedInput         = StandardLineEventInput
	OnMutableStandardLinesDeletedInput = StandardLineEventInput
	OnUnsupportedCreditNoteInput       = StandardLineEventInput
	OnInvoiceIssuedInput               = StandardLineEventInput
	OnPaymentAuthorizedInput           = StandardLineEventInput
	OnPaymentSettledInput              = StandardLineEventInput
)

type IsLineBillableAsOfInput struct {
	Line                   GatheringLine
	AsOf                   time.Time
	ProgressiveBilling     bool
	FeatureMeters          feature.FeatureMeters
	ResolvedBillablePeriod timeutil.ClosedPeriod
}

func (i IsLineBillableAsOfInput) Validate() error {
	if err := i.ResolvedBillablePeriod.Validate(); err != nil {
		return fmt.Errorf("validating resolved billable period: %w", err)
	}

	if i.AsOf.IsZero() {
		return fmt.Errorf("as of is required")
	}

	return nil
}

type SplitGatheringLineInput struct {
	Line          GatheringLine
	FeatureMeters feature.FeatureMeters
	SplitAt       time.Time
}

func (i SplitGatheringLineInput) Validate() error {
	var errs []error

	if err := i.Line.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line: %w", err))
	}

	if i.SplitAt.IsZero() {
		errs = append(errs, fmt.Errorf("split at is required"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	return errors.Join(errs...)
}

type SplitGatheringLineResult struct {
	PreSplitAtLine  GatheringLine
	PostSplitAtLine *GatheringLine
}

func (r SplitGatheringLineResult) Validate() error {
	var errs []error

	if err := r.PreSplitAtLine.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("pre split at line: %w", err))
	}

	if r.PostSplitAtLine != nil {
		if err := r.PostSplitAtLine.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("post split at line: %w", err))
		}
	}

	return errors.Join(errs...)
}

type LineEngine interface {
	// GetLineEngineType returns the discriminator owned by this engine implementation.
	GetLineEngineType() LineEngineType

	// IsLineBillableAsOf returns true if the line is billable as of the given time.
	IsLineBillableAsOf(ctx context.Context, input IsLineBillableAsOfInput) (bool, error)

	// SplitGatheringLine splits a gathering line on an engine-specific boundary if required.
	SplitGatheringLine(ctx context.Context, input SplitGatheringLineInput) (SplitGatheringLineResult, error)
	// BuildStandardInvoiceLines materializes gathering lines into standard lines for a target invoice.
	// Returned standard lines must reuse the exact same line IDs as the input gathering lines.
	BuildStandardInvoiceLines(ctx context.Context, input BuildStandardInvoiceLinesInput) (StandardLines, error)
	// BuildStandardLinesForGatheringPreview materializes gathering lines from BuildStandardInvoiceLinesInput
	// into transient StandardLines for a read-only standard invoice preview. Implementations must be
	// side-effect free: they must not persist realization state, modify or allocate credits, mutate
	// input IDs, emit events, or perform external billing side effects. Returned StandardLines must
	// reuse the exact same line IDs as the input gathering lines.
	BuildStandardLinesForGatheringPreview(ctx context.Context, input BuildStandardInvoiceLinesInput) (StandardLines, error)
	// OnStandardInvoiceCreated is invoked after the standard invoice and its standard lines have been persisted.
	OnStandardInvoiceCreated(ctx context.Context, input OnStandardInvoiceCreatedInput) (StandardLines, error)
	// OnCollectionCompleted is invoked when a standard invoice collection window closes.
	OnCollectionCompleted(ctx context.Context, input OnCollectionCompletedInput) (StandardLines, error)
	// OnMutableStandardLinesDeletedBySystem is invoked after mutable standard invoice lines are marked deleted by the system.
	OnMutableStandardLinesDeletedBySystem(ctx context.Context, input OnMutableStandardLinesDeletedInput) error
	// OnMutableInvoiceLinesEditedViaAPI is invoked after mutable invoice lines are edited through the API.
	// Implementations must return exactly one CreatedLines entry for each input Created line and
	// exactly one UpdatedLines entry for each input Updated override, even when they only accept
	// the line unchanged.
	// Charge-backed creation semantics are documented in billing/README.md under
	// "Lineengine Charges Integration Plan".
	OnMutableInvoiceLinesEditedViaAPI(ctx context.Context, input OnMutableInvoiceUpdateInput) (OnMutableInvoiceUpdateResult, error)
	// OnUnsupportedCreditNote is invoked when a line deletion targets an immutable invoice but credit-note support is not available yet.
	OnUnsupportedCreditNote(ctx context.Context, input OnUnsupportedCreditNoteInput) error
	// OnInvoiceIssued is invoked when a standard invoice reaches the issued state.
	OnInvoiceIssued(ctx context.Context, input OnInvoiceIssuedInput) error
	// OnPaymentAuthorized is invoked when a standard invoice reaches the payment authorized state.
	OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) error
	// OnPaymentSettled is invoked when a standard invoice reaches the paid state.
	OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) error
}

type LineCalculator interface {
	// CalculateLines recalculates detailed lines and totals for standard-invoice lines owned by this engine.
	CalculateLines(input CalculateLinesInput) (StandardLines, error)
}

func LineEngineValidationComponent(engineType LineEngineType) ComponentName {
	return ComponentName(fmt.Sprintf("openmeter.lineengine.%s", engineType))
}

func NewLineEngineValidationError(engine LineEngine, err error) error {
	if err == nil {
		return nil
	}

	if engine == nil {
		return fmt.Errorf("line engine is required")
	}

	component := LineEngineValidationComponent(engine.GetLineEngineType())
	validationErr := ValidationWithComponent(component, err)

	if _, convertErr := ToValidationIssues(validationErr); convertErr == nil {
		return validationErr
	}

	return ValidationWithComponent(
		component,
		ValidationIssue{
			Severity:  ValidationIssueSeverityCritical,
			Code:      ValidationIssueCodeLineEngineCollectionCompletedFailed,
			Message:   err.Error(),
			Component: component,
		},
	)
}

type CreateLineRouter interface {
	GetLineEngineForCreateLine(line GenericInvoiceLineReader) (LineEngineType, error)
}

type DefaultCreateLineRouter struct{}

func (DefaultCreateLineRouter) GetLineEngineForCreateLine(line GenericInvoiceLineReader) (LineEngineType, error) {
	if line == nil {
		return "", fmt.Errorf("line is required")
	}

	return LineEngineTypeInvoice, nil
}
