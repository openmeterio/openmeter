package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.LineEngine = (*LineEngine)(nil)

type LineEngine struct {
	service *service
}

func (e *LineEngine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeUsageBased
}

func (e *LineEngine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return !input.AsOf.Before(input.ResolvedBillablePeriod.To), nil
}

func (e *LineEngine) SplitGatheringLine(_ context.Context, input billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	res := billing.SplitGatheringLineResult{}

	if err := input.Validate(); err != nil {
		return res, fmt.Errorf("validating input: %w", err)
	}

	line := input.Line
	if line.ChargeID == nil || *line.ChargeID == "" {
		return res, fmt.Errorf("usage based gathering line[%s]: charge id is required", line.ID)
	}

	if !line.ServicePeriod.Contains(input.SplitAt) {
		return res, fmt.Errorf("usage based gathering line[%s]: splitAt is not within the line period", line.ID)
	}

	postSplitAtLine, err := line.CloneForCreate(func(l *billing.GatheringLine) {
		l.ServicePeriod.From = input.SplitAt
		l.ChildUniqueReferenceID = nil
	})
	if err != nil {
		return res, fmt.Errorf("cloning post split line: %w", err)
	}

	postSplitAtLineEmpty, err := isUsageBasedSplitPeriodEmpty(postSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if post split line is empty: %w", err)
	}

	if !postSplitAtLineEmpty {
		if err := postSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating post split line: %w", err)
		}
	}

	line.ServicePeriod.To = input.SplitAt
	line.InvoiceAt = input.SplitAt
	line.ChildUniqueReferenceID = nil

	preSplitAtLine := line

	preSplitAtLineEmpty, err := isUsageBasedSplitPeriodEmpty(preSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if pre split line is empty: %w", err)
	}

	if preSplitAtLineEmpty {
		preSplitAtLine.DeletedAt = lo.ToPtr(clock.Now())
	} else {
		if err := preSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating pre split line: %w", err)
		}
	}

	var postSplitAtLinePtr *billing.GatheringLine
	if !postSplitAtLineEmpty {
		postSplitAtLinePtr = &postSplitAtLine
	}

	return billing.SplitGatheringLineResult{
		PreSplitAtLine:  preSplitAtLine,
		PostSplitAtLine: postSplitAtLinePtr,
	}, nil
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
		meta.ExpandDetailedLines,
	}, "gathering preview")
	if err != nil {
		return nil, err
	}

	for _, stdLine := range stdLines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return nil, fmt.Errorf("usage based charge[%s] not found for gathering preview line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		previewResult, err := e.buildGatheringPreviewRun(ctx, charge, stdLine)
		if err != nil {
			return nil, fmt.Errorf("building gathering preview run for line[%s]: %w", stdLine.ID, err)
		}

		if err := populateStandardLineFromRun(stdLine, populateStandardLineFromRunInput{
			Run:        previewResult.Run,
			Runs:       previewResult.Runs,
			UnitConfig: charge.Intent.GetEffectiveUnitConfig(),
		}); err != nil {
			return nil, fmt.Errorf("populating gathering preview line[%s] from run: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating gathering preview line[%s]: %w", stdLine.ID, err)
		}
	}

	return stdLines, nil
}

func (e *LineEngine) buildGatheringPreviewRun(ctx context.Context, charge usagebased.Charge, stdLine *billing.StandardLine) (usagebasedrun.BuildCreditThenInvoiceGatheringPreviewRunResult, error) {
	if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return usagebasedrun.BuildCreditThenInvoiceGatheringPreviewRunResult{}, fmt.Errorf(
			"usage based standard line[%s]: unsupported settlement mode for gathering preview: %s",
			stdLine.ID,
			charge.Intent.GetSettlementMode(),
		)
	}

	stateMachineConfig, err := e.service.getStateMachineConfigForPatch(ctx, charge)
	if err != nil {
		return usagebasedrun.BuildCreditThenInvoiceGatheringPreviewRunResult{}, fmt.Errorf("getting state machine config for line[%s]: %w", stdLine.ID, err)
	}

	runType := usagebased.RealizationRunTypePartialInvoice
	storedAtLT := meta.NormalizeTimestamp(stdLine.Period.To)
	servicePeriodTo := storedAtLT
	if resolveInvoiceCreatedTrigger(charge, stdLine.Period) == meta.TriggerFinalInvoiceCreated {
		runType = usagebased.RealizationRunTypeFinalRealization
		storedAtLT, _ = stateMachineConfig.CustomerOverride.MergedProfile.WorkflowConfig.Collection.Interval.AddTo(charge.Intent.GetEffectiveServicePeriod().To)
		storedAtLT = meta.NormalizeTimestamp(storedAtLT)
		servicePeriodTo = meta.NormalizeTimestamp(charge.Intent.GetEffectiveServicePeriod().To)
	}

	return e.service.runs.BuildCreditThenInvoiceGatheringPreviewRun(ctx, usagebasedrun.BuildCreditThenInvoiceGatheringPreviewRunInput{
		Charge:             charge,
		CustomerOverride:   stateMachineConfig.CustomerOverride,
		FeatureMeter:       stateMachineConfig.FeatureMeter,
		Type:               runType,
		StoredAtLT:         storedAtLT,
		ServicePeriodTo:    servicePeriodTo,
		LineID:             stdLine.ID,
		InvoiceID:          stdLine.InvoiceID,
		CurrencyCalculator: stateMachineConfig.CurrencyCalculator,
	})
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

		if stateMachine.GetCharge().Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
			return nil, fmt.Errorf(
				"usage based standard line[%s]: unsupported settlement mode for standard invoice creation: %s",
				stdLine.ID,
				stateMachine.GetCharge().Intent.GetSettlementMode(),
			)
		}

		// Becoming active after the service period starts is not an invoice lifecycle event, so we
		// still rely on the generic TriggerNext/AdvanceUntilStateStable flow before invoice-created
		// lifecycle transitions take over.
		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		trigger := resolveInvoiceCreatedTrigger(stateMachine.GetCharge(), stdLine.Period)
		if stateMachine.GetCharge().State.CurrentRealizationRunID != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("line[%s]: %w", stdLine.ID, usagebased.ErrActiveRealizationRunAlreadyExists),
			}
		}

		if err := stateMachine.FireAndActivate(ctx, trigger, invoiceCreatedInput{
			LineID:          stdLine.ID,
			InvoiceID:       input.Invoice.ID,
			ServicePeriodTo: stdLine.Period.To,
		}); err != nil {
			return nil, fmt.Errorf("triggering %s for charge[%s]: %w", trigger, stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s] after %s: %w", stateMachine.GetCharge().ID, trigger, err)
		}

		charge := stateMachine.GetCharge()
		currentRun, err := charge.GetCurrentRealizationRun()
		if err != nil {
			return nil, fmt.Errorf("getting current realization run for charge[%s]: %w", charge.ID, err)
		}

		if err := populateStandardLineFromRun(stdLine, populateStandardLineFromRunInput{
			Run:        currentRun,
			Runs:       charge.Realizations,
			UnitConfig: charge.Intent.GetEffectiveUnitConfig(),
		}); err != nil {
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
			return nil, fmt.Errorf("advancing usage based charge[%s] after collection_completed: %w", stateMachine.GetCharge().ID, err)
		}

		charge := stateMachine.GetCharge()
		currentRun, err := charge.GetCurrentRealizationRun()
		if err != nil {
			return nil, fmt.Errorf("getting current realization run for charge[%s]: %w", charge.ID, err)
		}

		if err := populateStandardLineFromRun(stdLine, populateStandardLineFromRunInput{
			Run:        currentRun,
			Runs:       charge.Realizations,
			UnitConfig: charge.Intent.GetEffectiveUnitConfig(),
		}); err != nil {
			return nil, fmt.Errorf("populating standard line from run for charge[%s]: %w", charge.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return input.Lines, nil
}

func (e *LineEngine) OnMutableInvoiceLinesEditedViaAPI(ctx context.Context, input billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	if err := input.Validate(); err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("validating input: %w", err)
	}

	if len(input.Created) > 0 {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("usage-based charge create: %w", billing.ErrCannotUpdateChargeManagedLine)
	}

	if len(input.Updated) > 0 {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("usage-based charge update: %w", billing.ErrCannotUpdateChargeManagedLine)
	}

	for _, line := range input.Deleted {
		if err := e.handleInvoiceLineDeleteViaAPI(ctx, input.Invoice, line); err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, err
		}
	}

	return billing.OnMutableInvoiceUpdateResult{}, nil
}

func (e *LineEngine) handleInvoiceLineDeleteViaAPI(ctx context.Context, invoice billing.GenericInvoiceReader, line billing.GenericInvoiceLine) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}

	chargeID := line.GetChargeID()
	if chargeID == nil || *chargeID == "" {
		return fmt.Errorf("usage based line[%s]: charge id is required", line.GetID())
	}

	charge, err := e.service.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: line.GetLineID().Namespace,
			ID:        *chargeID,
		},
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDetailedLines,
		},
	})
	if err != nil {
		return fmt.Errorf("getting usage based charge for deleted line[%s]: %w", line.GetID(), err)
	}

	if charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return fmt.Errorf(
			"usage based line[%s]: unsupported settlement mode for API delete: %s",
			line.GetID(),
			charge.Intent.GetSettlementMode(),
		)
	}

	nonVoidedRuns := charge.Realizations.WithoutVoidedBillingHistory()
	switch line.AsInvoiceLine().Type() {
	case billing.InvoiceLineTypeGathering:
		if len(nonVoidedRuns) > 0 {
			// TODO: Treat deleting the remaining gathering tail after prior
			// usage-based realizations as a service-period shortening instead
			// of deleting the whole charge and touching already-realized
			// invoice history.
			return fmt.Errorf("usage based gathering line[%s] cannot be deleted with existing realization runs: %w",
				line.GetID(),
				billing.ErrCannotEditProgressivelyBilledUsageBasedLine)
		}

		// TODO: implement
		return billing.ErrCannotUpdateChargeManagedLine
	case billing.InvoiceLineTypeStandard:
		if len(nonVoidedRuns) > 1 {
			return fmt.Errorf("usage based standard line[%s] cannot be deleted with multiple realization runs: %w",
				line.GetID(),
				billing.ErrCannotEditProgressivelyBilledUsageBasedLine)
		}

		if len(nonVoidedRuns) == 0 {
			// This is an internal consistency error, we are not supposed to surface this to the user, so no typed error wrapping.
			return fmt.Errorf("usage based standard line[%s] cannot be deleted with no realization runs", line.GetID())
		}

		deletePatch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
			Policy:       meta.RefundAsCreditsDeletePolicy,
		})
		if err != nil {
			return fmt.Errorf("creating usage based charge[%s] API delete patch: %w", charge.ID, err)
		}

		charge, patches, err := e.applyChargePatchForInvoiceLineEditViaAPI(ctx, charge, deletePatch)
		if err != nil {
			return fmt.Errorf("usage based line[%s]: applying charge delete patch for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		standardInvoice, err := invoice.AsInvoice().AsStandardInvoice()
		if err != nil {
			return fmt.Errorf("usage based line[%s]: getting standard invoice: %w", line.GetID(), err)
		}

		stdInvoicePatches, rest, err := patches.BisectByStandardInvoiceID(standardInvoice.ID)
		if err != nil {
			return fmt.Errorf("usage based line[%s]: bisecting invoice patches for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		if len(stdInvoicePatches) != 1 {
			return fmt.Errorf("received unexpected number of standard invoice patches for line[%s]: count=%d %v", line.GetID(), len(stdInvoicePatches), stdInvoicePatches)
		}

		stdInvoicePatch, err := stdInvoicePatches.RequireSingularStandardInvoiceLineDeletePatch()
		if err != nil {
			return fmt.Errorf("usage based line[%s]: requiring singular standard invoice line delete patch for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		if err := stdInvoicePatch.RequireTarget(line); err != nil {
			return fmt.Errorf("usage based line[%s]: validating standard invoice line delete patch target for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		standardLine, err := line.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return fmt.Errorf("usage based line[%s]: getting standard line for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		_, err = e.deleteMutableStandardLineRealization(ctx, charge, standardInvoice, &standardLine)
		if err != nil {
			return fmt.Errorf("usage based line[%s]: deleting mutable standard line realization for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		// Handle the remaining gathering line patches
		if err := rest.RequireType(invoiceupdater.PatchOpDeleteGatheringLineByChargeID, invoiceupdater.CountLessThanOrEqualTo(1)); err != nil {
			return fmt.Errorf("usage based line[%s]: validating remaining gathering line delete patches for charge[%s]: %w", line.GetID(), charge.ID, err)
		}

		if len(rest) > 0 {
			err := e.service.invoiceUpdater.ApplyPatches(ctx, invoice.GetCustomerID(), rest)
			if err != nil {
				return fmt.Errorf("usage based line[%s]: applying remaining gathering line delete patches for charge[%s]: %w", line.GetID(), charge.ID, err)
			}
		}

		return nil
	default:
		return fmt.Errorf("usage based line[%s]: unexpected line type: %s", line.GetID(), line.AsInvoiceLine().Type())
	}
}

func (e *LineEngine) applyChargePatchForInvoiceLineEditViaAPI(ctx context.Context, charge usagebased.Charge, patch meta.Patch) (usagebased.Charge, invoiceupdater.Patches, error) {
	if err := patch.Validate(); err != nil {
		return usagebased.Charge{}, nil, fmt.Errorf("validating usage based charge[%s] API line edit patch: %w", charge.ID, err)
	}

	stateMachineConfig, err := e.service.getStateMachineConfigForPatch(ctx, charge)
	if err != nil {
		return usagebased.Charge{}, nil, fmt.Errorf("getting state machine config for charge[%s]: %w", charge.ID, err)
	}

	stateMachine, err := e.service.newStateMachine(stateMachineConfig)
	if err != nil {
		return usagebased.Charge{}, nil, fmt.Errorf("new state machine for usage based charge[%s]: %w", charge.ID, err)
	}

	creditThenInvoiceStateMachine, ok := stateMachine.(*CreditThenInvoiceStateMachine)
	if !ok {
		return usagebased.Charge{}, nil, fmt.Errorf("BUG: usage based charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
	}

	if err := creditThenInvoiceStateMachine.FireAndActivate(ctx, patch.Trigger(), patch); err != nil {
		return usagebased.Charge{}, nil, fmt.Errorf("triggering %s for charge[%s]: %w", patch.Trigger(), charge.ID, err)
	}

	return creditThenInvoiceStateMachine.GetCharge(), creditThenInvoiceStateMachine.DrainInvoicePatches(), nil
}

func (e *LineEngine) OnMutableStandardLinesDeletedBySystem(ctx context.Context, input billing.OnMutableStandardLinesDeletedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	chargesByID, err := e.getChargesForStandardLineEvent(ctx, input, meta.Expands{
		meta.ExpandRealizations,
		meta.ExpandDetailedLines,
	}, "deleted standard lines")
	if err != nil {
		return err
	}

	gatheringLineDeletePatches := make(invoiceupdater.Patches, 0, len(input.Lines))
	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("usage based charge[%s] not found for deleted standard line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		charge, err = e.deleteMutableStandardLineRealization(ctx, charge, input.Invoice, stdLine)
		if err != nil {
			return err
		}

		chargesByID[*stdLine.ChargeID] = charge
		gatheringLineDeletePatches = append(gatheringLineDeletePatches, invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(*stdLine.ChargeID))
	}

	if len(gatheringLineDeletePatches) > 0 {
		if err := e.service.invoiceUpdater.ApplyPatches(ctx, input.Invoice.GetCustomerID(), gatheringLineDeletePatches); err != nil {
			return fmt.Errorf("applying gathering line delete patches for deleted usage based standard lines: %w", err)
		}
	}

	return nil
}

// deleteMutableStandardLineRealization removes the usage-based realization
// backing a mutable deleted standard invoice line, including credit correction
// and current-run detachment.
func (e *LineEngine) deleteMutableStandardLineRealization(
	ctx context.Context,
	charge usagebased.Charge,
	invoice billing.StandardInvoice,
	stdLine *billing.StandardLine,
) (usagebased.Charge, error) {
	run, err := charge.Realizations.GetByLineID(stdLine.ID)
	if err != nil {
		return usagebased.Charge{}, err
	}
	// Deleted realizations have already been cleaned up through a prior line deletion,
	// so billing must not run the cleanup path for them again.
	if run.DeletedAt != nil {
		return usagebased.Charge{}, fmt.Errorf("usage based standard line[%s] cannot be deleted because realization run[%s] is already deleted", stdLine.ID, run.ID.ID)
	}

	if run.InvoiceID == nil || *run.InvoiceID != invoice.ID {
		return usagebased.Charge{}, fmt.Errorf("usage based standard line[%s] cannot be deleted because realization run[%s] is not associated with invoice[%s]", stdLine.ID, run.ID.ID, invoice.ID)
	}

	if run.Payment != nil {
		return usagebased.Charge{}, fmt.Errorf("usage based standard line[%s] cannot be deleted because realization run[%s] has payment allocation", stdLine.ID, run.ID.ID)
	}

	if run.InvoiceUsage != nil {
		return usagebased.Charge{}, fmt.Errorf("usage based standard line[%s] cannot be deleted because realization run[%s] has invoice accrued allocation", stdLine.ID, run.ID.ID)
	}

	currencyCalculator, err := charge.Intent.GetCurrency().Calculator()
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("getting currency calculator for charge[%s]: %w", charge.ID, err)
	}

	now := clock.Now()
	if _, err := e.service.runs.CorrectAllCredits(ctx, usagebasedrun.CorrectAllCreditRealizationsInput{
		Charge:             charge,
		Run:                run,
		AllocateAt:         run.ServicePeriodTo,
		CurrencyCalculator: currencyCalculator,
	}); err != nil {
		return usagebased.Charge{}, fmt.Errorf("correcting credits for deleted usage based standard line[%s] run[%s]: %w", stdLine.ID, run.ID.ID, err)
	}

	charge, err = e.markMutableStandardLineRunDeleted(ctx, charge, run, now)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("marking realization run[%s] deleted for usage based standard line[%s]: %w", run.ID.ID, stdLine.ID, err)
	}

	return charge, nil
}

func (e *LineEngine) OnUnsupportedCreditNote(ctx context.Context, input billing.OnUnsupportedCreditNoteInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	chargesByID, err := e.getChargesForStandardLineEvent(ctx, input, meta.Expands{
		meta.ExpandRealizations,
	}, "unsupported credit note")
	if err != nil {
		return err
	}

	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("usage based charge[%s] not found for unsupported credit note line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		// Unsupported credit notes void the run for future rating history, but
		// they must not mark it deleted; deleted runs mean invoice/ledger cleanup
		// already happened, while this state preserves audit history.
		run, err := charge.Realizations.GetByLineID(stdLine.ID)
		if err != nil {
			return err
		}

		if run.InvoiceID == nil || *run.InvoiceID != input.Invoice.ID {
			return fmt.Errorf("usage based standard line[%s] cannot be marked unsupported credit note because realization run[%s] is not associated with invoice[%s]", stdLine.ID, run.ID.ID, input.Invoice.ID)
		}

		if run.DeletedAt != nil {
			return fmt.Errorf("usage based standard line[%s] cannot be marked unsupported credit note because realization run[%s] is already deleted", stdLine.ID, run.ID.ID)
		}

		if run.Type == usagebased.RealizationRunTypeInvalidDueToUnsupportedCreditNote {
			continue
		}

		// We need to mark the run as invalid to prevent it from being considered in further realization runs.
		if _, err := e.service.adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
			ID:   run.ID,
			Type: mo.Some(usagebased.RealizationRunTypeInvalidDueToUnsupportedCreditNote),
		}); err != nil {
			return fmt.Errorf("marking realization run[%s] invalid due to unsupported credit note for usage based standard line[%s]: %w", run.ID.ID, stdLine.ID, err)
		}
	}

	return nil
}

func (e *LineEngine) markMutableStandardLineRunDeleted(
	ctx context.Context,
	charge usagebased.Charge,
	run usagebased.RealizationRun,
	deletedAt time.Time,
) (usagebased.Charge, error) {
	if _, err := e.service.adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:        run.ID,
		DeletedAt: mo.Some(lo.ToPtr(deletedAt)),
	}); err != nil {
		return usagebased.Charge{}, err
	}

	charge.Realizations = charge.Realizations.Without(run.ID)

	currentRunDeleted := charge.State.CurrentRealizationRunID != nil && *charge.State.CurrentRealizationRunID == run.ID.ID
	if currentRunDeleted {
		charge.State.CurrentRealizationRunID = nil
		if charge.Status != usagebased.StatusDeleted {
			charge.Status = usagebased.StatusActive
			charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(charge.Intent.GetEffectiveServicePeriod().To))
		}

		updatedChargeBase, err := e.service.adapter.UpdateCharge(ctx, charge.ChargeBase)
		if err != nil {
			return usagebased.Charge{}, err
		}

		charge.ChargeBase = updatedChargeBase
	}

	return charge, nil
}

func (e *LineEngine) getChargesForStandardLineEvent(ctx context.Context, input billing.StandardLineEventInput, expands meta.Expands, operation string) (map[string]usagebased.Charge, error) {
	chargeIDs := make([]string, 0, len(input.Lines))
	seenChargeIDs := make(map[string]struct{}, len(input.Lines))

	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil || *stdLine.ChargeID == "" {
			return nil, fmt.Errorf("usage based standard line[%s]: charge id is required", stdLine.ID)
		}

		if stdLine.Namespace != input.Invoice.Namespace {
			return nil, fmt.Errorf("usage based standard line[%s]: namespace %s does not match invoice namespace %s", stdLine.ID, stdLine.Namespace, input.Invoice.Namespace)
		}

		if _, ok := seenChargeIDs[*stdLine.ChargeID]; ok {
			continue
		}

		seenChargeIDs[*stdLine.ChargeID] = struct{}{}
		chargeIDs = append(chargeIDs, *stdLine.ChargeID)
	}

	charges, err := e.service.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: input.Invoice.Namespace,
		IDs:       chargeIDs,
		Expands:   expands,
	})
	if err != nil {
		return nil, fmt.Errorf("getting usage based charges for %s: %w", operation, err)
	}

	return lo.KeyBy(charges, func(charge usagebased.Charge) string {
		return charge.ID
	}), nil
}

func (e *LineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.fireLineTrigger(ctx, fireLineTriggerInput{
		Lines:   input.Lines,
		Trigger: meta.TriggerInvoiceIssued,
		InputFn: func(stdLine *billing.StandardLine) models.Validator {
			return billing.StandardLineWithInvoiceHeader{
				Line:    stdLine,
				Invoice: input.Invoice,
			}
		},
		AdvanceUntilStateStable: true,
	})
}

func (e *LineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.recordRunPayments(ctx, recordRunPaymentsInput{
		Lines:    input.Lines,
		Invoice:  input.Invoice,
		RecordFn: e.recordPaymentAuthorized,
	})
}

func (e *LineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.recordRunPayments(ctx, recordRunPaymentsInput{
		Lines:    input.Lines,
		Invoice:  input.Invoice,
		RecordFn: e.recordPaymentSettled,
	})
}

type fireLineTriggerInput struct {
	Lines                   billing.StandardLines
	Trigger                 meta.Trigger
	InputFn                 func(*billing.StandardLine) models.Validator
	AdvanceUntilStateStable bool
}

func (i fireLineTriggerInput) Validate() error {
	if len(i.Lines) == 0 {
		return fmt.Errorf("lines are required")
	}

	if i.Trigger == "" {
		return fmt.Errorf("trigger is required")
	}

	if i.InputFn == nil {
		return fmt.Errorf("inputFn is required")
	}

	return nil
}

func (e *LineEngine) fireLineTrigger(ctx context.Context, input fireLineTriggerInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating fire line trigger input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		canFire, err := stateMachine.CanFire(ctx, input.Trigger)
		if err != nil {
			return fmt.Errorf("checking %s for charge[%s]: %w", input.Trigger, stateMachine.GetCharge().ID, err)
		}

		if !canFire {
			return fmt.Errorf(
				"charge[%s] in status %s cannot handle %s for standard line[%s]",
				stateMachine.GetCharge().ID,
				stateMachine.GetCharge().Status,
				input.Trigger,
				stdLine.ID,
			)
		}

		if err := stateMachine.FireAndActivate(ctx, input.Trigger, input.InputFn(stdLine)); err != nil {
			return fmt.Errorf("triggering %s for charge[%s]: %w", input.Trigger, stateMachine.GetCharge().ID, err)
		}

		if input.AdvanceUntilStateStable {
			if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
				return fmt.Errorf("advancing usage based charge[%s] after %s: %w", stateMachine.GetCharge().ID, input.Trigger, err)
			}
		}
	}

	return nil
}

func (e *LineEngine) newStateMachineForStandardLine(ctx context.Context, stdLine *billing.StandardLine) (StateMachine, error) {
	if stdLine.ChargeID == nil {
		return nil, fmt.Errorf("usage based standard line[%s]: charge id is required", stdLine.ID)
	}

	charge, err := e.service.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: stdLine.Namespace,
			ID:        *stdLine.ChargeID,
		},
		Expands: meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return nil, fmt.Errorf("getting usage based charge for line[%s]: %w", stdLine.ID, err)
	}

	stateMachineConfig, err := e.service.getStateMachineConfigForPatch(ctx, charge)
	if err != nil {
		return nil, fmt.Errorf("getting state machine config for line[%s]: %w", stdLine.ID, err)
	}

	stateMachine, err := e.service.newStateMachine(stateMachineConfig)
	if err != nil {
		return nil, fmt.Errorf("creating state machine for line[%s]: %w", stdLine.ID, err)
	}

	return stateMachine, nil
}

func isUsageBasedSplitPeriodEmpty(line billing.GatheringLine) (bool, error) {
	price := line.GetPrice()
	if price == nil {
		return false, fmt.Errorf("price is nil")
	}

	if price.Type() == productcatalog.FlatPriceType {
		return false, nil
	}

	return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil
}
