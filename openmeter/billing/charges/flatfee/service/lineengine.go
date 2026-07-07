package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.LineEngine = (*LineEngine)(nil)

type LineEngine struct {
	service *service
}

func (e *LineEngine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeFlatFee
}

func (e *LineEngine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return true, nil
}

func (e *LineEngine) SplitGatheringLine(context.Context, billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("flat fee line is not progressively billed")
}

func (e *LineEngine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := slicesx.MapWithErr(input.GatheringLines, func(gatheringLine billing.GatheringLine) (*billing.StandardLine, error) {
		stdLine, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
		if err != nil {
			return nil, fmt.Errorf("converting gathering line to standard line: %w", err)
		}

		return stdLine, nil
	})
	if err != nil {
		return nil, err
	}

	return stdLines, nil
}

func (e *LineEngine) BuildStandardLinesForGatheringPreview(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	stdLines, err := input.GatheringLines.ToStandardLines(input.Invoice.ID)
	if err != nil {
		return nil, err
	}

	chargesByID, err := e.getChargesForStandardLineEvent(ctx, billing.StandardLineEventInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
	}, meta.Expands{
		meta.ExpandRealizations,
	})
	if err != nil {
		return nil, err
	}

	for _, stdLine := range stdLines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return nil, fmt.Errorf("flat fee charge[%s] not found for gathering preview line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		previewResult, err := e.service.realizations.BuildCreditThenInvoiceGatheringPreviewRun(flatfeerealizations.BuildCreditThenInvoiceGatheringPreviewRunInput{
			Charge: charge,
			Line:   *stdLine,
		})
		if err != nil {
			return nil, fmt.Errorf("building gathering preview run for line[%s]: %w", stdLine.ID, err)
		}

		if err := populateFlatFeeStandardLineFromRun(stdLine, previewResult.Run); err != nil {
			return nil, fmt.Errorf("populating gathering preview line[%s] from run: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating gathering preview line[%s]: %w", stdLine.ID, err)
		}
	}

	return stdLines, nil
}

func (e *LineEngine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	stdLines, err := slicesx.MapWithErr(input.Lines, func(stdLine *billing.StandardLine) (*billing.StandardLine, error) {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return nil, err
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing flat fee charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerFinalInvoiceCreated, billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return nil, fmt.Errorf("triggering %s for charge[%s]: %w", meta.TriggerFinalInvoiceCreated, stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing flat fee charge[%s] after %s: %w", stateMachine.GetCharge().ID, meta.TriggerFinalInvoiceCreated, err)
		}

		charge := stateMachine.GetCharge()
		if charge.Realizations.CurrentRun == nil {
			return nil, fmt.Errorf("flat fee charge[%s]: current run is required for line[%s]", charge.ID, stdLine.ID)
		}

		if err := populateFlatFeeStandardLineFromRun(stdLine, *charge.Realizations.CurrentRun); err != nil {
			return nil, fmt.Errorf("populating standard line from run for charge[%s]: %w", charge.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}

		return stdLine, nil
	})
	if err != nil {
		return nil, err
	}

	return stdLines, nil
}

func (e *LineEngine) OnCollectionCompleted(ctx context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return nil, err
		}

		canFire, err := stateMachine.CanFire(ctx, meta.TriggerCollectionCompleted)
		if err != nil {
			return nil, fmt.Errorf("checking collection_completed for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if !canFire {
			continue
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerCollectionCompleted); err != nil {
			return nil, fmt.Errorf("triggering collection_completed for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing flat fee charge[%s] after collection_completed: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return input.Lines, nil
}

func (e *LineEngine) OnMutableInvoiceLinesEditedViaAPI(ctx context.Context, input billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	if err := input.Validate(); err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("validating input: %w", err)
	}

	createdLines, err := e.createManualInvoiceLines(ctx, input)
	if err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, err
	}

	updatedLines, err := slicesx.MapWithErr(input.Updated, func(override billing.InvoiceLineOverride) (billing.GenericInvoiceLine, error) {
		chargeID := override.ExistingLine.GetChargeID()
		if chargeID == nil || *chargeID == "" {
			return nil, fmt.Errorf("flat fee line[%s]: charge id is required", override.ExistingLine.GetID())
		}

		charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
			ChargeID: meta.ChargeID{
				Namespace: override.ExistingLine.GetLineID().Namespace,
				ID:        *chargeID,
			},
			Expands: meta.Expands{
				meta.ExpandRealizations,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("getting flat fee charge for line[%s]: %w", override.ExistingLine.GetID(), err)
		}

		if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
			return nil, fmt.Errorf(
				"flat fee line[%s]: unsupported settlement mode for API edit: %s",
				override.ExistingLine.GetID(),
				charge.Intent.GetSettlementMode(),
			)
		}

		stateMachine, err := e.service.newStateMachine(StateMachineConfig{
			Charge:               charge,
			Adapter:              e.service.adapter,
			Realizations:         e.service.realizations,
			Service:              e.service,
			CreditNotesSupported: e.service.creditNotesSupported.Load(),
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine for flat fee charge[%s]: %w", charge.ID, err)
		}

		creditThenInvoiceStateMachine, ok := stateMachine.(*CreditThenInvoiceStateMachine)
		if !ok {
			return nil, fmt.Errorf("BUG: flat fee charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
		}

		lineManualEditPatch, err := meta.NewPatchLineManualEdit(meta.NewPatchLineManualEditInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
			Override:     override,
		})
		if err != nil {
			return nil, fmt.Errorf("creating flat-fee line manual edit patch for line[%s]: %w", override.ExistingLine.GetID(), err)
		}

		if err := creditThenInvoiceStateMachine.FireAndActivate(ctx, meta.TriggerLineManualEdit, lineManualEditPatch); err != nil {
			return nil, fmt.Errorf("triggering %s for charge[%s]: %w", meta.TriggerLineManualEdit, charge.ID, err)
		}

		patches := creditThenInvoiceStateMachine.DrainInvoicePatches()
		var targetLine billing.GenericInvoiceLine
		switch override.ExistingLine.AsInvoiceLine().Type() {
		case billing.InvoiceLineTypeStandard:
			updatePatch, err := patches.RequireSingularLineUpdatePatchForTarget(override.ExistingLine)
			if err != nil {
				return nil, fmt.Errorf("line[%s]: validating line manual edit update patch target: %w", override.ExistingLine.GetID(), err)
			}

			targetLine = updatePatch.TargetState
		case billing.InvoiceLineTypeGathering:
			gatheringPatch, err := patches.RequireSingularGatheringLinePatchForCharge(*chargeID)
			if err != nil {
				return nil, fmt.Errorf("line[%s]: validating line manual edit gathering patch target: %w", override.ExistingLine.GetID(), err)
			}

			switch gatheringPatch.Op() {
			case invoiceupdater.PatchOpUpsertGatheringLineByChargeID:
				upsertPatch, err := gatheringPatch.AsUpsertGatheringLineByChargeIDPatch()
				if err != nil {
					return nil, fmt.Errorf("line[%s]: getting line manual edit gathering upsert patch: %w", override.ExistingLine.GetID(), err)
				}

				targetLine = upsertPatch.TargetState.AsGenericLine()
			case invoiceupdater.PatchOpDeleteGatheringLineByChargeID:
				// TODO: support zero-proration manual gathering-line edits by
				// modeling the API result as a line deletion/detach instead of
				// an updated line.
				return nil, fmt.Errorf("line[%s]: zero-proration manual gathering-line edits are not supported yet: %w", override.ExistingLine.GetID(), billing.ErrInvoiceLineZeroAmountDeleteInstead)
			default:
				return nil, fmt.Errorf("line[%s]: expected line manual edit gathering patch, got %s", override.ExistingLine.GetID(), gatheringPatch.Op())
			}
		default:
			return nil, billing.ErrCannotUpdateChargeManagedLine
		}

		updatedLine, err := override.ExistingLine.WithTargetState(targetLine)
		if err != nil {
			return nil, fmt.Errorf("line[%s]: merging line manual edit patch target state: %w", override.ExistingLine.GetID(), err)
		}

		return updatedLine, nil
	})
	if err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, err
	}

	for _, line := range input.Deleted {
		chargeID := line.GetChargeID()
		if chargeID == nil || *chargeID == "" {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("flat fee line[%s]: charge id is required", line.GetID())
		}

		charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
			ChargeID: meta.ChargeID{
				Namespace: line.GetLineID().Namespace,
				ID:        *chargeID,
			},
			Expands: meta.Expands{
				meta.ExpandRealizations,
			},
		})
		if err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("getting flat fee charge for deleted line[%s]: %w", line.GetID(), err)
		}

		if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf(
				"flat fee line[%s]: unsupported settlement mode for API delete: %s",
				line.GetID(),
				charge.Intent.GetSettlementMode(),
			)
		}

		if err := validateManualDeleteLine(charge, line); err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, err
		}

		stateMachine, err := e.service.newStateMachine(StateMachineConfig{
			Charge:               charge,
			Adapter:              e.service.adapter,
			Realizations:         e.service.realizations,
			Service:              e.service,
			CreditNotesSupported: e.service.creditNotesSupported.Load(),
		})
		if err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("new state machine for flat fee charge[%s]: %w", charge.ID, err)
		}

		creditThenInvoiceStateMachine, ok := stateMachine.(*CreditThenInvoiceStateMachine)
		if !ok {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("BUG: flat fee charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
		}

		deletePatch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
			Policy:       meta.RefundAsCreditsDeletePolicy,
		})
		if err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("creating flat fee line[%s] manual delete patch: %w", line.GetID(), err)
		}

		if err := creditThenInvoiceStateMachine.FireAndActivate(ctx, meta.TriggerDelete, deletePatch); err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("triggering %s for charge[%s]: %w", meta.TriggerDelete, charge.ID, err)
		}

		if err := e.handleManualDeleteInvoicePatches(ctx, input.Invoice, line, *chargeID, creditThenInvoiceStateMachine.DrainInvoicePatches()); err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, err
		}
	}

	return billing.OnMutableInvoiceUpdateResult{
		CreatedLines: createdLines,
		UpdatedLines: updatedLines,
	}, nil
}

func (e *LineEngine) ValidateMutableInvoiceLineEditViaAPI(ctx context.Context, input billing.OnMutableInvoiceUpdateInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, line := range input.Created {
		if _, err := intentFromManualCreatedLine(ctx, input.Invoice, line, input.DefaultTaxCodeResolvers.Invoicing); err != nil {
			if line == nil {
				return fmt.Errorf("building manually created flat-fee charge intent: %w", err)
			}

			return fmt.Errorf("building manually created flat-fee charge intent for line[%s]: %w", line.GetID(), err)
		}
	}

	for _, override := range input.Updated {
		if err := e.validateManualUpdateLineViaAPI(ctx, override); err != nil {
			return err
		}
	}

	for _, line := range input.Deleted {
		if err := e.validateManualDeleteLineViaAPI(ctx, line); err != nil {
			return err
		}
	}

	return nil
}

func (e *LineEngine) validateManualUpdateLineViaAPI(ctx context.Context, override billing.InvoiceLineOverride) error {
	chargeID := override.ExistingLine.GetChargeID()
	if chargeID == nil || *chargeID == "" {
		return fmt.Errorf("flat fee line[%s]: charge id is required", override.ExistingLine.GetID())
	}

	charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: override.ExistingLine.GetLineID().Namespace,
			ID:        *chargeID,
		},
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	if err != nil {
		return fmt.Errorf("getting flat fee charge for line[%s]: %w", override.ExistingLine.GetID(), err)
	}

	if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return fmt.Errorf(
			"flat fee line[%s]: unsupported settlement mode for API edit: %s",
			override.ExistingLine.GetID(),
			charge.Intent.GetSettlementMode(),
		)
	}

	if err := override.ExistingLine.AsInvoiceLine().Type().Require(billing.InvoiceLineTypeStandard, billing.InvoiceLineTypeGathering); err != nil {
		return fmt.Errorf("flat fee line[%s]: unsupported line type for API edit: %s", override.ExistingLine.GetID(), override.ExistingLine.AsInvoiceLine().Type())
	}

	if _, err := meta.NewPatchLineManualEdit(meta.NewPatchLineManualEditInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Override:     override,
	}); err != nil {
		return fmt.Errorf("validating flat-fee line manual edit patch for line[%s]: %w", override.ExistingLine.GetID(), err)
	}

	return nil
}

func (e *LineEngine) validateManualDeleteLineViaAPI(ctx context.Context, line billing.GenericInvoiceLine) error {
	chargeID := line.GetChargeID()
	if chargeID == nil || *chargeID == "" {
		return fmt.Errorf("flat fee line[%s]: charge id is required", line.GetID())
	}

	charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: line.GetLineID().Namespace,
			ID:        *chargeID,
		},
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	if err != nil {
		return fmt.Errorf("getting flat fee charge for deleted line[%s]: %w", line.GetID(), err)
	}

	if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return fmt.Errorf(
			"flat fee line[%s]: unsupported settlement mode for API delete: %s",
			line.GetID(),
			charge.Intent.GetSettlementMode(),
		)
	}

	return validateManualDeleteLine(charge, line)
}

type manualCreatedInvoiceLine struct {
	sourceLine billing.GenericInvoiceLine
	intent     flatfee.Intent
}

func (e *LineEngine) createManualInvoiceLines(ctx context.Context, input billing.OnMutableInvoiceUpdateInput) ([]billing.GenericInvoiceLine, error) {
	if len(input.Created) == 0 {
		return nil, nil
	}

	if input.Invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}

	created, err := slicesx.MapWithErr(input.Created, func(line billing.GenericInvoiceLine) (manualCreatedInvoiceLine, error) {
		intent, err := intentFromManualCreatedLine(ctx, input.Invoice, line, input.DefaultTaxCodeResolvers.Invoicing)
		if err != nil {
			if line == nil {
				return manualCreatedInvoiceLine{}, fmt.Errorf("building manually created flat-fee charge intent: %w", err)
			}

			return manualCreatedInvoiceLine{}, fmt.Errorf("building manually created flat-fee charge intent for line[%s]: %w", line.GetID(), err)
		}

		return manualCreatedInvoiceLine{
			sourceLine: line,
			intent:     intent,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	createdCharges, err := e.service.Create(ctx, flatfee.CreateInput{
		Namespace: input.Invoice.GetInvoiceID().Namespace,
		Intents: lo.Map(created, func(line manualCreatedInvoiceLine, _ int) flatfee.Intent {
			return line.intent
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("creating manually managed flat-fee charges: %w", err)
	}

	if len(createdCharges) != len(created) {
		return nil, fmt.Errorf("expected %d manually created flat-fee charges, got %d", len(created), len(createdCharges))
	}

	out := make([]billing.GenericInvoiceLine, 0, len(createdCharges))
	for idx, charge := range createdCharges {
		sourceLine := created[idx].sourceLine
		switch sourceLine.AsInvoiceLine().Type() {
		case billing.InvoiceLineTypeGathering:
			if charge.GatheringLineToCreate == nil {
				return nil, fmt.Errorf("line[%s]: manually created flat-fee charge[%s] did not create a gathering line", sourceLine.GetID(), charge.Charge.ID)
			}

			line, err := sourceLine.WithTargetState(charge.GatheringLineToCreate.AsGenericLine())
			if err != nil {
				return nil, fmt.Errorf("line[%s]: merging manually created flat-fee charge target state: %w", sourceLine.GetID(), err)
			}

			out = append(out, line)
		case billing.InvoiceLineTypeStandard:
			line, err := e.attachManualStandardLine(ctx, input.Invoice, sourceLine, charge.Charge)
			if err != nil {
				return nil, err
			}

			out = append(out, line)
		default:
			return nil, fmt.Errorf("unsupported manually created flat-fee line type [charge_id=%s,line_id=%s,line_type=%s]: %w",
				charge.Charge.ID,
				sourceLine.GetID(),
				sourceLine.AsInvoiceLine().Type(),
				billing.ErrCannotUpdateChargeManagedLine)
		}
	}

	return out, nil
}

func (e *LineEngine) attachManualStandardLine(ctx context.Context, invoice billing.GenericInvoiceReader, sourceLine billing.GenericInvoiceLine, charge flatfee.Charge) (billing.GenericInvoiceLine, error) {
	standardInvoice, err := invoice.AsInvoice().AsStandardInvoice()
	if err != nil {
		return nil, fmt.Errorf("getting standard invoice for created line[%s]: %w", sourceLine.GetID(), err)
	}

	standardLine, err := sourceLine.AsInvoiceLine().AsStandardLine()
	if err != nil {
		return nil, fmt.Errorf("getting created standard line[%s]: %w", sourceLine.GetID(), err)
	}

	stateMachine, err := e.service.newStateMachine(StateMachineConfig{
		Charge:               charge,
		Adapter:              e.service.adapter,
		Realizations:         e.service.realizations,
		Service:              e.service,
		CreditNotesSupported: e.service.creditNotesSupported.Load(),
	})
	if err != nil {
		return nil, fmt.Errorf("new state machine for flat fee charge[%s]: %w", charge.ID, err)
	}

	creditThenInvoiceStateMachine, ok := stateMachine.(*CreditThenInvoiceStateMachine)
	if !ok {
		return nil, fmt.Errorf("BUG: flat fee charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
	}

	if err := creditThenInvoiceStateMachine.FireAndActivate(ctx, meta.TriggerAttachInvoiceLine, billing.StandardLineWithInvoiceHeader{
		Line:    &standardLine,
		Invoice: standardInvoice,
	}); err != nil {
		return nil, fmt.Errorf("triggering %s for charge[%s]: %w", meta.TriggerAttachInvoiceLine, charge.ID, err)
	}

	patches := creditThenInvoiceStateMachine.DrainInvoicePatches()
	updatePatch, err := patches.RequireSingularLineUpdatePatchForTarget(sourceLine)
	if err != nil {
		return nil, fmt.Errorf("line[%s]: validating attach update patch target: %w", sourceLine.GetID(), err)
	}

	line, err := sourceLine.WithTargetState(updatePatch.TargetState)
	if err != nil {
		return nil, fmt.Errorf("line[%s]: merging attach patch target state: %w", sourceLine.GetID(), err)
	}

	return line, nil
}

func validateManualDeleteLine(charge flatfee.Charge, line billing.GenericInvoiceLine) error {
	switch line.AsInvoiceLine().Type() {
	case billing.InvoiceLineTypeGathering:
		if charge.Realizations.CurrentRun != nil {
			return fmt.Errorf("cannot delete gathering line with current run [charge_id=%s,run_id=%s,line_id=%s]: %w",
				charge.ID,
				charge.Realizations.CurrentRun.ID.ID,
				line.GetID(),
				billing.ErrCannotUpdateChargeManagedLine)
		}
	case billing.InvoiceLineTypeStandard:
		currentRun := charge.Realizations.CurrentRun
		if currentRun == nil {
			return fmt.Errorf("missing current run [charge_id=%s,line_id=%s]: %w", charge.ID, line.GetID(), billing.ErrCannotUpdateChargeManagedLine)
		}

		if currentRun.Immutable {
			return fmt.Errorf("immutable current run [charge_id=%s,run_id=%s,line_id=%s]: %w", charge.ID, currentRun.ID.ID, line.GetID(), billing.ErrCannotUpdateChargeManagedLine)
		}

		if currentRun.LineID == nil || *currentRun.LineID != line.GetID() {
			return fmt.Errorf("line[%s]: current realization run must be attached to deleted line", line.GetID())
		}

		if currentRun.InvoiceID == nil || *currentRun.InvoiceID != line.GetInvoiceID() {
			return fmt.Errorf("line[%s]: current realization run must be attached to deleted invoice", line.GetID())
		}
	default:
		return billing.ErrCannotUpdateChargeManagedLine
	}

	return nil
}

func (e *LineEngine) handleManualDeleteInvoicePatches(ctx context.Context, invoice billing.GenericInvoiceReader, line billing.GenericInvoiceLine, chargeID string, patches invoiceupdater.Patches) error {
	if len(patches) == 0 {
		return fmt.Errorf("line[%s]: expected manual delete invoice patches", line.GetID())
	}

	for _, patch := range patches {
		switch patch.Op() {
		case invoiceupdater.PatchOpDeleteGatheringLineByChargeID:
			deletePatch, err := patch.AsDeleteGatheringLineByChargeIDPatch()
			if err != nil {
				return fmt.Errorf("line[%s]: getting manual delete gathering-line patch: %w", line.GetID(), err)
			}

			if err := deletePatch.RequireCharge(chargeID); err != nil {
				return fmt.Errorf("line[%s]: validating manual delete gathering-line patch target: %w", line.GetID(), err)
			}
		case invoiceupdater.PatchOpLineDelete:
			deletePatch, err := patch.AsDeleteLinePatch()
			if err != nil {
				return fmt.Errorf("line[%s]: getting manual delete line patch: %w", line.GetID(), err)
			}

			if err := deletePatch.RequireTarget(line); err != nil {
				return fmt.Errorf("line[%s]: validating manual delete line patch target: %w", line.GetID(), err)
			}

			standardInvoice, err := invoice.AsInvoice().AsStandardInvoice()
			if err != nil {
				return fmt.Errorf("line[%s]: getting standard invoice for manual delete cleanup: %w", line.GetID(), err)
			}

			standardLine, err := line.AsInvoiceLine().AsStandardLine()
			if err != nil {
				return fmt.Errorf("line[%s]: getting standard line for manual delete cleanup: %w", line.GetID(), err)
			}

			if err := e.cleanupDeletedStandardLines(ctx, billing.StandardLineEventInput{
				Invoice: standardInvoice,
				Lines:   billing.StandardLines{&standardLine},
			}); err != nil {
				return fmt.Errorf("line[%s]: cleaning up manual delete line patch: %w", line.GetID(), err)
			}
		default:
			return fmt.Errorf("line[%s]: unexpected manual delete invoice patch %s", line.GetID(), patch.Op())
		}
	}

	return nil
}

func (e *LineEngine) OnMutableStandardLinesDeletedBySystem(ctx context.Context, input billing.OnMutableStandardLinesDeletedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.cleanupDeletedStandardLines(ctx, input)
}

func (e *LineEngine) cleanupDeletedStandardLines(ctx context.Context, input billing.StandardLineEventInput) error {
	chargesByID, err := e.getChargesForStandardLineEvent(ctx, input, meta.Expands{
		meta.ExpandRealizations,
	})
	if err != nil {
		return fmt.Errorf("getting flat fee charges for deleted standard lines: %w", err)
	}

	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("flat fee charge[%s] not found for deleted standard line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		run, err := charge.Realizations.GetByLineID(stdLine.ID)
		if err != nil {
			return err
		}

		if run.DeletedAt != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] is already deleted", stdLine.ID, run.ID.ID)
		}

		if run.InvoiceID == nil || *run.InvoiceID != input.Invoice.ID {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] is not associated with invoice[%s]", stdLine.ID, run.ID.ID, input.Invoice.ID)
		}

		if run.AccruedUsage != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] has invoice accrued allocation", stdLine.ID, run.ID.ID)
		}

		if run.Payment != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] has payment allocation", stdLine.ID, run.ID.ID)
		}

		if charge.Realizations.CurrentRun != nil && charge.Realizations.CurrentRun.ID.ID == run.ID.ID {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] is still current for charge[%s]", stdLine.ID, run.ID.ID, charge.ID)
		}

		currencyCalculator, err := charge.Intent.GetCurrency().Calculator()
		if err != nil {
			return fmt.Errorf("getting currency calculator for charge[%s]: %w", charge.ID, err)
		}

		if _, err := e.service.realizations.CorrectAllCredits(ctx, flatfeerealizations.CorrectAllCreditRealizationsInput{
			Charge:             charge,
			Run:                run,
			AllocateAt:         flatfee.UsageBookedAt(charge.Intent.GetEffectivePaymentTerm(), run.ServicePeriod),
			CurrencyCalculator: currencyCalculator,
		}); err != nil {
			return fmt.Errorf("correcting credits for deleted flat fee standard line[%s] run[%s]: %w", stdLine.ID, run.ID.ID, err)
		}

		if err := e.service.adapter.UpsertDetailedLines(ctx, run.ID, nil); err != nil {
			return fmt.Errorf("deleting detailed lines for deleted flat fee standard line[%s] run[%s]: %w", stdLine.ID, run.ID.ID, err)
		}

		if _, err := e.service.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:        run.ID,
			DeletedAt: mo.Some(lo.ToPtr(clock.Now())),
		}); err != nil {
			return fmt.Errorf("marking realization run[%s] deleted for flat fee standard line[%s]: %w", run.ID.ID, stdLine.ID, err)
		}
	}

	return nil
}

func (e *LineEngine) OnUnsupportedCreditNote(ctx context.Context, input billing.OnUnsupportedCreditNoteInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	chargesByID, err := e.getChargesForStandardLineEvent(ctx, input, meta.Expands{
		meta.ExpandRealizations,
	})
	if err != nil {
		return fmt.Errorf("getting flat fee charges for unsupported credit note: %w", err)
	}

	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("flat fee charge[%s] not found for unsupported credit note line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		// Unsupported credit notes void the run for future billing history, but
		// they must not mark it deleted; deleted runs mean invoice/ledger cleanup
		// already happened, while this state preserves audit history.
		run, err := charge.Realizations.GetByLineID(stdLine.ID)
		if err != nil {
			return err
		}

		if run.InvoiceID == nil || *run.InvoiceID != input.Invoice.ID {
			return fmt.Errorf("flat fee standard line[%s] cannot be marked unsupported credit note because realization run[%s] is not associated with invoice[%s]", stdLine.ID, run.ID.ID, input.Invoice.ID)
		}

		if run.DeletedAt != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be marked unsupported credit note because realization run[%s] is already deleted", stdLine.ID, run.ID.ID)
		}

		if run.Type == flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote {
			continue
		}

		if _, err := e.service.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:   run.ID,
			Type: mo.Some(flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote),
		}); err != nil {
			return fmt.Errorf("marking realization run[%s] invalid due to unsupported credit note for flat fee standard line[%s]: %w", run.ID.ID, stdLine.ID, err)
		}
	}

	return nil
}

func (e *LineEngine) newStateMachineForStandardLine(ctx context.Context, stdLine *billing.StandardLine) (*CreditThenInvoiceStateMachine, error) {
	if stdLine == nil {
		return nil, fmt.Errorf("flat fee standard line is nil")
	}

	if stdLine.ChargeID == nil || *stdLine.ChargeID == "" {
		return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
	}

	charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: stdLine.Namespace,
			ID:        *stdLine.ChargeID,
		},
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting flat fee charge for line[%s]: %w", stdLine.ID, err)
	}

	if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return nil, fmt.Errorf(
			"flat fee standard line[%s]: unsupported settlement mode for standard invoice lifecycle: %s",
			stdLine.ID,
			charge.Intent.GetSettlementMode(),
		)
	}

	stateMachine, err := e.service.newStateMachine(StateMachineConfig{
		Charge:               charge,
		Adapter:              e.service.adapter,
		Realizations:         e.service.realizations,
		Service:              e.service,
		CreditNotesSupported: e.service.creditNotesSupported.Load(),
	})
	if err != nil {
		return nil, fmt.Errorf("new state machine for flat fee charge[%s]: %w", charge.ID, err)
	}

	creditThenInvoiceStateMachine, ok := stateMachine.(*CreditThenInvoiceStateMachine)
	if !ok {
		return nil, fmt.Errorf("BUG: flat fee charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
	}

	return creditThenInvoiceStateMachine, nil
}

func (e *LineEngine) getChargesForStandardLineEvent(ctx context.Context, input billing.StandardLineEventInput, expands meta.Expands) (map[string]flatfee.Charge, error) {
	chargeIDs := make([]string, 0, len(input.Lines))
	seenChargeIDs := make(map[string]struct{}, len(input.Lines))

	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil || *stdLine.ChargeID == "" {
			return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
		}

		if stdLine.Namespace != input.Invoice.Namespace {
			return nil, fmt.Errorf("flat fee standard line[%s]: namespace %s does not match invoice namespace %s", stdLine.ID, stdLine.Namespace, input.Invoice.Namespace)
		}

		if _, ok := seenChargeIDs[*stdLine.ChargeID]; ok {
			continue
		}

		seenChargeIDs[*stdLine.ChargeID] = struct{}{}
		chargeIDs = append(chargeIDs, *stdLine.ChargeID)
	}

	charges, err := e.service.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: input.Invoice.Namespace,
		IDs:       chargeIDs,
		Expands:   expands,
	})
	if err != nil {
		return nil, fmt.Errorf("getting flat fee charges: %w", err)
	}

	return lo.KeyBy(charges, func(charge flatfee.Charge) string {
		return charge.ID
	}), nil
}

func (e *LineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerInvoiceIssued, billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("triggering invoice_issued for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return fmt.Errorf("advancing flat fee charge[%s] after invoice_issued: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return nil
}

func (e *LineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := e.service.postInvoicePaymentAuthorized(ctx, stateMachine.GetCharge(), billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("authorizing invoice payment for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return nil
}

func (e *LineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := e.service.postInvoicePaymentSettled(ctx, stateMachine.GetCharge(), billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("settling invoice payment for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := stateMachine.RefetchCharge(ctx); err != nil {
			return fmt.Errorf("refetching flat fee charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		canFire, err := stateMachine.CanFire(ctx, meta.TriggerAllPaymentsSettled)
		if err != nil {
			return fmt.Errorf("checking all_payments_settled for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if !canFire {
			continue
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerAllPaymentsSettled); err != nil {
			return fmt.Errorf("triggering all_payments_settled for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return nil
}
