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
	GatheringInvoice GatheringInvoice
	FeatureMeters    feature.FeatureMeters
	LineID           string
	SplitAt          time.Time
}

func (i SplitGatheringLineInput) Validate() error {
	var errs []error

	if i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if i.SplitAt.IsZero() {
		errs = append(errs, fmt.Errorf("split at is required"))
	}

	if i.GatheringInvoice.Lines.IsAbsent() {
		errs = append(errs, fmt.Errorf("gathering invoice must have lines expanded"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	return errors.Join(errs...)
}

type SplitGatheringLineResult struct {
	PreSplitAtLine   GatheringLine
	PostSplitAtLine  GatheringLine
	GatheringInvoice GatheringInvoice
}

type LineEngine interface {
	// GetLineEngineType returns the discriminator owned by this engine implementation.
	GetLineEngineType() LineEngineType

	// IsLineBillableAsOf returns true if the line is billable as of the given time.
	IsLineBillableAsOf(ctx context.Context, input IsLineBillableAsOfInput) (bool, error)

	// SplitGatheringLine splits a gathering line on an engine-specific boundary if required.
	SplitGatheringLine(ctx context.Context, input SplitGatheringLineInput) (SplitGatheringLineResult, error)
	// BuildStandardInvoiceLines materializes gathering lines into standard lines for a target invoice.
	BuildStandardInvoiceLines(ctx context.Context, input BuildStandardInvoiceLinesInput) (StandardLines, error)

	// TODO[later]: implement this lifecycle event/hook.
	// SnapshotCollection(ctx context.Context, lines StandardLines) (StandardLines, error)
	// TODO[later]: implement this lifecycle event/hook.
	// OnInvoiceIssued(ctx context.Context, lines StandardLines) error
	// TODO[later]: implement this lifecycle event/hook.
	// OnPaymentAuthorized(ctx context.Context, lines StandardLines) error
	// TODO[later]: implement this lifecycle event/hook.
	// OnPaymentSettled(ctx context.Context, lines StandardLines) error
}
