package billing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
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

func (i ChangeSource) Require(value ChangeSource) error {
	if err := i.Validate(); err != nil {
		return err
	}

	if i != value {
		return fmt.Errorf("must be %s", value)
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
	OnInvoiceFinalizingInput           = StandardLineEventInput
	OnMutableStandardLinesDeletedInput = StandardLineEventInput
	OnUnsupportedCreditNoteInput       = StandardLineEventInput
	OnInvoiceIssuedInput               = StandardLineEventInput
	OnPaymentAuthorizedInput           = StandardLineEventInput
	OnPaymentSettledInput              = StandardLineEventInput
)

type AreLinesBillableAsOfInput struct {
	Invoice            GatheringInvoice
	AsOf               time.Time
	ProgressiveBilling bool
	Lines              GatheringLines
}

func (i AreLinesBillableAsOfInput) Validate() error {
	var errs []error

	if err := i.Invoice.GetInvoiceID().Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice: %w", err))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	if err := i.Lines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("lines: %w", err))
	}

	for index, line := range i.Lines {
		if line.Namespace != i.Invoice.Namespace {
			errs = append(errs, fmt.Errorf("line[%d]: namespace %s does not match invoice namespace %s", index, line.Namespace, i.Invoice.Namespace))
		}

		if line.InvoiceID != i.Invoice.ID {
			errs = append(errs, fmt.Errorf("line[%d]: invoice ID %s does not match invoice ID %s", index, line.InvoiceID, i.Invoice.ID))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type IsLineBillableAsOfResult struct {
	Billable       bool
	BillablePeriod timeutil.ClosedPeriod
}

func (r IsLineBillableAsOfResult) Validate() error {
	var errs []error

	if r.Billable {
		if err := r.BillablePeriod.ValidateAsRequired(); err != nil {
			errs = append(errs, fmt.Errorf("billable period: %w", err))
		}
	} else if !lo.IsEmpty(r.BillablePeriod) {
		errs = append(errs, errors.New("billable period must be empty when line is not billable"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type SplitGatheringLineInput struct {
	Line    GatheringLine
	SplitAt time.Time
}

func (i SplitGatheringLineInput) Validate() error {
	var errs []error

	if err := i.Line.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line: %w", err))
	}

	if i.SplitAt.IsZero() {
		errs = append(errs, fmt.Errorf("split at is required"))
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

	// AreLinesBillableAsOf returns one result for each line in the same order. Implementations may return usable
	// results together with validation issues. Operational errors make the returned results unusable.
	AreLinesBillableAsOf(ctx context.Context, input AreLinesBillableAsOfInput) ([]IsLineBillableAsOfResult, error)

	// SplitGatheringLine splits a gathering line on an engine-specific boundary if required.
	SplitGatheringLine(ctx context.Context, input SplitGatheringLineInput) (SplitGatheringLineResult, error)
	// BuildStandardInvoiceLines materializes gathering lines into standard lines for a target invoice.
	// Returned standard lines must reuse the exact same line IDs as the input gathering lines. When the lines can be
	// persisted with validation issues, implementations return both the materialized lines and an error tree composed
	// entirely of ValidationIssue values. Operational errors abort invoice creation and make returned lines unusable.
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
	// ValidateMutableInvoiceLineEditViaAPI is invoked before mutable invoice lines are edited through the API.
	// Can be used to reject edits that are not supported by the engine (including deletion, etc.) to prevent the
	// invoice from entering an invalid state without recovery.
	//
	// Additional checks can be performed in OnMutableInvoiceLinesEditedViaAPI but those errors will become
	// validation issues, thus alter the invoice state.
	//
	// For API requests it is better to reject and edit before, the existing validation issue logic is geared
	// towards state machine failures.
	//
	// Implementations must not mutate invoice, charge, ledger, or external state from this hook.
	ValidateMutableInvoiceLineEditViaAPI(ctx context.Context, input OnMutableInvoiceUpdateInput) error
	// OnMutableInvoiceLinesEditedViaAPI is invoked after mutable invoice lines are edited through the API.
	// Implementations must return exactly one CreatedLines entry for each input Created line and
	// exactly one UpdatedLines entry for each input Updated override, even when they only accept
	// the line unchanged.
	// Charge-backed creation semantics are documented in billing/README.md under
	// "Lineengine Charges Integration Plan".
	OnMutableInvoiceLinesEditedViaAPI(ctx context.Context, input OnMutableInvoiceUpdateInput) (OnMutableInvoiceUpdateResult, error)
	// OnUnsupportedCreditNote is invoked when a line deletion targets an immutable invoice but credit-note support is not available yet.
	OnUnsupportedCreditNote(ctx context.Context, input OnUnsupportedCreditNoteInput) error
	// OnInvoiceFinalizing is invoked during issuing before the invoice is sent to the invoicing app.
	// Implementations may persist retry-safe engine-owned effects and must return fully calculated
	// lines with exactly the same IDs as the input.
	OnInvoiceFinalizing(ctx context.Context, input OnInvoiceFinalizingInput) (StandardLines, error)
	// OnInvoiceIssued is invoked after external invoice issuance succeeds, while
	// billing books charge effects, and before entering the issued state.
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
		return errors.New("line engine is required")
	}

	return ValidationWithComponent(LineEngineValidationComponent(engine.GetLineEngineType()), err)
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
