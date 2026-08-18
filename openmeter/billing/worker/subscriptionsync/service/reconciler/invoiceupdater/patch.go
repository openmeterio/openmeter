package invoiceupdater

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type PatchOperation string

const (
	PatchOpLineCreate           PatchOperation = "line_create"
	PatchOpLineDelete           PatchOperation = "line_delete"
	PatchOpLineUpdate           PatchOperation = "line_update"
	PatchOpSplitLineGroupDelete PatchOperation = "split_line_group_delete"
	PatchOpSplitLineGroupUpdate PatchOperation = "split_line_group_update"
)

type PatchLineCreate struct {
	Line billing.GatheringLine
}

type PatchLineDelete struct {
	Line      billing.LineID
	InvoiceID string
}

// NewUpdateLinePatchInput describes only the line fields subscription sync intends to change.
// An absent option preserves persisted state; DeletedAt containing nil explicitly restores a line.
type NewUpdateLinePatchInput struct {
	Line      billing.LineID
	InvoiceID string

	ServicePeriod        mo.Option[timeutil.ClosedPeriod]
	InvoiceAt            mo.Option[time.Time]
	DeletedAt            mo.Option[*time.Time]
	FlatFeePerUnitAmount mo.Option[alpacadecimal.Decimal]
}

func (i NewUpdateLinePatchInput) IsNoop() bool {
	return !i.ServicePeriod.IsPresent() &&
		!i.InvoiceAt.IsPresent() &&
		!i.DeletedAt.IsPresent() &&
		!i.FlatFeePerUnitAmount.IsPresent()
}

func (i NewUpdateLinePatchInput) Validate() error {
	var errs []error

	if err := i.Line.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line: %w", err))
	}

	if i.InvoiceID == "" {
		errs = append(errs, errors.New("invoice id is required"))
	}

	if period, ok := i.ServicePeriod.Get(); ok {
		if err := period.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("service period: %w", err))
		}
	}

	if invoiceAt, ok := i.InvoiceAt.Get(); ok && invoiceAt.IsZero() {
		errs = append(errs, errors.New("invoice at is required"))
	}

	if perUnitAmount, ok := i.FlatFeePerUnitAmount.Get(); ok && perUnitAmount.IsNegative() {
		errs = append(errs, errors.New("flat fee per unit amount must not be negative"))
	}

	if i.IsNoop() {
		errs = append(errs, errors.New("at least one line update is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type PatchLineUpdate struct {
	NewUpdateLinePatchInput
}

func (p PatchLineUpdate) Apply(line billing.GenericInvoiceLine) (billing.GenericInvoiceLine, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validating update: %w", err)
	}

	if line == nil {
		return nil, errors.New("line is required")
	}

	if line.GetLineID() != p.Line {
		return nil, fmt.Errorf("line id mismatch: expected %s, got %s", p.Line.ID, line.GetLineID().ID)
	}

	if line.GetInvoiceID() != p.InvoiceID {
		return nil, fmt.Errorf("invoice id mismatch: expected %s, got %s", p.InvoiceID, line.GetInvoiceID())
	}

	updatedLine, err := line.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning line: %w", err)
	}

	if line.AsInvoiceLine().Type() == billing.InvoiceLineTypeGathering {
		existingGatheringLine, err := line.AsInvoiceLine().AsGatheringLine()
		if err != nil {
			return nil, fmt.Errorf("converting existing line to gathering line: %w", err)
		}

		if existingGatheringLine.SplitLineHierarchy != nil {
			hierarchy, err := existingGatheringLine.SplitLineHierarchy.Clone()
			if err != nil {
				return nil, fmt.Errorf("cloning split line hierarchy: %w", err)
			}

			updatedGatheringLine, err := updatedLine.AsInvoiceLine().AsGatheringLine()
			if err != nil {
				return nil, fmt.Errorf("converting updated line to gathering line: %w", err)
			}

			updatedGatheringLine.SplitLineHierarchy = &hierarchy
			updatedLine = updatedGatheringLine.AsGenericLine()
		}
	}

	if period, ok := p.ServicePeriod.Get(); ok {
		updatedLine.UpdateServicePeriod(func(current *timeutil.ClosedPeriod) {
			*current = period
		})
	}

	if invoiceAt, ok := p.InvoiceAt.Get(); ok {
		invoiceAtAccessor, ok := updatedLine.(billing.InvoiceAtAccessor)
		if !ok {
			return nil, fmt.Errorf("line[%s] does not support invoice at updates", p.Line.ID)
		}

		invoiceAtAccessor.SetInvoiceAt(invoiceAt)
	}

	if deletedAt, ok := p.DeletedAt.Get(); ok {
		updatedLine.SetDeletedAt(deletedAt)
	}

	if perUnitAmount, ok := p.FlatFeePerUnitAmount.Get(); ok {
		if err := SetFlatFeePerUnitAmount(updatedLine, perUnitAmount); err != nil {
			return nil, fmt.Errorf("setting flat fee per unit amount: %w", err)
		}
	}

	return updatedLine, nil
}

type PatchSplitLineGroupDelete struct {
	Group models.NamespacedID
}

type PatchSplitLineGroupUpdate struct {
	TargetState billing.SplitLineGroupUpdate
}

type Patch struct {
	op PatchOperation

	createLinePatch PatchLineCreate
	deleteLinePatch PatchLineDelete
	updateLinePatch PatchLineUpdate

	deleteSplitLineGroupPatch PatchSplitLineGroupDelete
	updateSplitLineGroupPatch PatchSplitLineGroupUpdate
}

func (p Patch) Op() PatchOperation {
	return p.op
}

func (p Patch) AsCreateLinePatch() (PatchLineCreate, error) {
	if p.op != PatchOpLineCreate {
		return PatchLineCreate{}, fmt.Errorf("expected create line patch, got %s", p.op)
	}

	return p.createLinePatch, nil
}

func (p Patch) AsDeleteLinePatch() (PatchLineDelete, error) {
	if p.op != PatchOpLineDelete {
		return PatchLineDelete{}, fmt.Errorf("expected delete line patch, got %s", p.op)
	}

	return p.deleteLinePatch, nil
}

func (p Patch) AsUpdateLinePatch() (PatchLineUpdate, error) {
	if p.op != PatchOpLineUpdate {
		return PatchLineUpdate{}, fmt.Errorf("expected update line patch, got %s", p.op)
	}

	return p.updateLinePatch, nil
}

func (p Patch) AsDeleteSplitLineGroupPatch() (PatchSplitLineGroupDelete, error) {
	if p.op != PatchOpSplitLineGroupDelete {
		return PatchSplitLineGroupDelete{}, fmt.Errorf("expected delete split line group patch, got %s", p.op)
	}

	return p.deleteSplitLineGroupPatch, nil
}

func (p Patch) AsUpdateSplitLineGroupPatch() (PatchSplitLineGroupUpdate, error) {
	if p.op != PatchOpSplitLineGroupUpdate {
		return PatchSplitLineGroupUpdate{}, fmt.Errorf("expected update split line group patch, got %s", p.op)
	}

	return p.updateSplitLineGroupPatch, nil
}

func NewDeleteLinePatch(lineID billing.LineID, invoiceID string) Patch {
	return Patch{
		op: PatchOpLineDelete,
		deleteLinePatch: PatchLineDelete{
			Line:      lineID,
			InvoiceID: invoiceID,
		},
	}
}

func NewUpdateLinePatch(input NewUpdateLinePatchInput) (Patch, error) {
	if err := input.Validate(); err != nil {
		return Patch{}, err
	}

	return Patch{
		op: PatchOpLineUpdate,
		updateLinePatch: PatchLineUpdate{
			NewUpdateLinePatchInput: input,
		},
	}, nil
}

func NewDeleteSplitLineGroupPatch(groupID models.NamespacedID) Patch {
	return Patch{
		op: PatchOpSplitLineGroupDelete,
		deleteSplitLineGroupPatch: PatchSplitLineGroupDelete{
			Group: groupID,
		},
	}
}

func NewUpdateSplitLineGroupPatch(group billing.SplitLineGroupUpdate) Patch {
	return Patch{
		op: PatchOpSplitLineGroupUpdate,
		updateSplitLineGroupPatch: PatchSplitLineGroupUpdate{
			TargetState: group,
		},
	}
}

func NewCreateLinePatch(line billing.GatheringLine) Patch {
	return Patch{
		op: PatchOpLineCreate,
		createLinePatch: PatchLineCreate{
			Line: line,
		},
	}
}

func (p Patch) Log(logger *slog.Logger) {
	switch p.op {
	case PatchOpLineCreate:
		logger.Info("create line patch", "line_id", p.createLinePatch.Line.GetLineID().ID, "new_service_period_from", p.createLinePatch.Line.GetServicePeriod().From, "new_service_period_to", p.createLinePatch.Line.GetServicePeriod().To, "unique_reference_id", p.createLinePatch.Line.GetChildUniqueReferenceID())
	case PatchOpLineDelete:
		logger.Info("delete line patch", "line_id", p.deleteLinePatch.Line, "invoice_id", p.deleteLinePatch.InvoiceID)
	case PatchOpLineUpdate:
		args := []any{
			"line_id", p.updateLinePatch.Line.ID,
			"invoice_id", p.updateLinePatch.InvoiceID,
		}
		updatedFields := make([]string, 0, 4)

		if period, ok := p.updateLinePatch.ServicePeriod.Get(); ok {
			updatedFields = append(updatedFields, "service_period")
			args = append(args, "new_service_period_from", period.From, "new_service_period_to", period.To)
		}

		if invoiceAt, ok := p.updateLinePatch.InvoiceAt.Get(); ok {
			updatedFields = append(updatedFields, "invoice_at")
			args = append(args, "new_invoice_at", invoiceAt)
		}

		if deletedAt, ok := p.updateLinePatch.DeletedAt.Get(); ok {
			updatedFields = append(updatedFields, "deleted_at")
			args = append(args, "new_deleted_at", deletedAt)
		}

		if perUnitAmount, ok := p.updateLinePatch.FlatFeePerUnitAmount.Get(); ok {
			updatedFields = append(updatedFields, "flat_fee_per_unit_amount")
			args = append(args, "new_flat_fee_per_unit_amount", perUnitAmount.String())
		}

		args = append(args, "updated_fields", updatedFields)
		logger.Info("update line patch", args...)
	case PatchOpSplitLineGroupDelete:
		logger.Info("delete split line group patch", "group_id", p.deleteSplitLineGroupPatch.Group.ID)
	case PatchOpSplitLineGroupUpdate:
		logger.Info("update split line group patch", "group_id", p.updateSplitLineGroupPatch.TargetState.ID, "new_service_period_from", p.updateSplitLineGroupPatch.TargetState.ServicePeriod.From, "new_service_period_to", p.updateSplitLineGroupPatch.TargetState.ServicePeriod.To)
	default:
		logger.Info("unknown patch operation", "operation", p.op)
	}
}
