package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type InvoiceUpdaterConfig struct {
	BillingService  billing.Service
	ChargesService  charges.Service
	BackfillCharges bool
	Logger          *slog.Logger
}

func (c InvoiceUpdaterConfig) Validate() error {
	errs := []error{}

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service is required"))
	}

	if c.ChargesService == nil {
		errs = append(errs, errors.New("charges service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

type InvoiceUpdater struct {
	billingService  billing.Service
	chargesService  charges.Service
	backfillCharges bool
	logger          *slog.Logger
}

func NewInvoiceUpdater(config InvoiceUpdaterConfig) (*InvoiceUpdater, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating invoice updater config: %w", err)
	}

	return &InvoiceUpdater{
		billingService:  config.BillingService,
		chargesService:  config.ChargesService,
		backfillCharges: config.BackfillCharges,
		logger:          config.Logger,
	}, nil
}

func (u *InvoiceUpdater) ApplyPatches(ctx context.Context, customerID customer.CustomerID, patches []linePatch) error {
	patchesParsed, err := u.parsePatches(patches)
	if err != nil {
		return fmt.Errorf("parsing patches: %w", err)
	}

	// Let's provision pending lines
	err = u.provisionUpcomingLines(ctx, customerID, patchesParsed.newLines)
	if err != nil {
		return fmt.Errorf("provisioning upcoming lines: %w", err)
	}

	// Let's split line patches by invoiceID
	for invoiceID, linePatches := range patchesParsed.updatedLinesByInvoiceID {
		namespacedInvoiceID := billing.InvoiceID{
			Namespace: customerID.Namespace,
			ID:        invoiceID,
		}

		invoice, err := u.billingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: namespacedInvoiceID,
		})
		if err != nil {
			return fmt.Errorf("getting invoice: %w", err)
		}

		if invoice.Type() == billing.InvoiceTypeGathering {
			if err := u.updateGatheringInvoice(ctx, namespacedInvoiceID, linePatches); err != nil {
				return fmt.Errorf("updating gathering invoice: %w", err)
			}

			continue
		}

		standardInvoice, err := invoice.AsStandardInvoice()
		if err != nil {
			return fmt.Errorf("converting invoice to standard invoice: %w", err)
		}

		if !standardInvoice.StatusDetails.Immutable {
			if err := u.updateMutableStandardInvoice(ctx, standardInvoice, linePatches); err != nil {
				return fmt.Errorf("updating mutable invoice: %w", err)
			}

			continue
		}

		if err := u.updateImmutableInvoice(ctx, standardInvoice, linePatches); err != nil {
			return fmt.Errorf("updating immutable invoice: %w", err)
		}
	}

	// Let's update split line groups
	err = u.upsertSplitLineGroups(ctx, customerID, patchesParsed.splitLineGroups)
	if err != nil {
		return fmt.Errorf("upserting split line groups: %w", err)
	}

	// Let's make sure charges are in sync
	err = u.syncCharges(ctx, syncChargesInput{
		CustomerID: customerID,
		Upserts:    patchesParsed.chargeUpserts,
		Deletes: lo.Map(patchesParsed.chargeDeletes, func(delete deleteChargeByUniqueReferenceIDPatch, _ int) string {
			return delete.UniqueReferenceID
		}),
	})
	if err != nil {
		return fmt.Errorf("upserting charges: %w", err)
	}

	return nil
}

type patchesParsed struct {
	newLines []billing.GatheringLine

	updatedLinesByInvoiceID map[string]invoicePatches

	splitLineGroups splitLineGroupPatches

	chargeUpserts []upsertChargeAndAssociateLinesPatch
	chargeDeletes []deleteChargeByUniqueReferenceIDPatch
}

type invoicePatches struct {
	updatedLines []billing.GenericInvoiceLine
	deletedLines []billing.LineID
}

type splitLineGroupPatches struct {
	deleted []models.NamespacedID
	updated []billing.SplitLineGroupUpdate
}

func (u *InvoiceUpdater) parsePatches(patches []linePatch) (patchesParsed, error) {
	parsed := patchesParsed{
		updatedLinesByInvoiceID: make(map[string]invoicePatches),
	}

	for _, patch := range patches {
		switch patch.Op() {
		case patchOpLineCreate:
			create, err := patch.AsCreateLinePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting line: %w", err)
			}

			parsed.newLines = append(parsed.newLines, create.Line)
		case patchOpLineDelete:
			delete, err := patch.AsDeleteLinePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting line: %w", err)
			}

			lineUpdates := parsed.updatedLinesByInvoiceID[delete.InvoiceID]
			lineUpdates.deletedLines = append(lineUpdates.deletedLines, delete.Line)
			parsed.updatedLinesByInvoiceID[delete.InvoiceID] = lineUpdates
		case patchOpLineUpdate:
			update, err := patch.AsUpdateLinePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting line: %w", err)
			}

			lineUpdates := parsed.updatedLinesByInvoiceID[update.TargetState.GetInvoiceID()]
			lineUpdates.updatedLines = append(lineUpdates.updatedLines, update.TargetState)
			parsed.updatedLinesByInvoiceID[update.TargetState.GetInvoiceID()] = lineUpdates
		case patchOpSplitLineGroupDelete:
			delete, err := patch.AsDeleteSplitLineGroupPatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting split line group: %w", err)
			}

			parsed.splitLineGroups.deleted = append(parsed.splitLineGroups.deleted, delete.Group)
		case patchOpSplitLineGroupUpdate:
			update, err := patch.AsUpdateSplitLineGroupPatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting split line group: %w", err)
			}

			parsed.splitLineGroups.updated = append(parsed.splitLineGroups.updated, update.TargetState)
		case patchOpUpsertChargeAndAssociateLines:
			upsert, err := patch.AsUpsertChargeAndAssociateLinesPatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting charge: %w", err)
			}

			parsed.chargeUpserts = append(parsed.chargeUpserts, upsert)
		case patchOpDeleteChargeByUniqueReferenceID:
			delete, err := patch.AsDeleteChargeByUniqueReferenceIDPatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting charge: %w", err)
			}

			parsed.chargeDeletes = append(parsed.chargeDeletes, delete)
		default:
			return patchesParsed{}, fmt.Errorf("unexpected patch operation: %s", patch.Op())
		}
	}

	return parsed, nil
}

func (u *InvoiceUpdater) provisionUpcomingLines(ctx context.Context, customerID customer.CustomerID, lines []billing.GatheringLine) error {
	if len(lines) == 0 {
		return nil
	}

	linesByCurrency := lo.GroupBy(lines, func(l billing.GatheringLine) currencyx.Code {
		return l.Currency
	})

	for currency, lines := range linesByCurrency {
		_, err := u.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: customerID,
			Currency: currency,
			Lines:    lines,
		})
		if err != nil {
			return fmt.Errorf("creating pending invoice lines: %w", err)
		}
	}

	return nil
}

func (u *InvoiceUpdater) updateMutableStandardInvoice(ctx context.Context, invoice billing.StandardInvoice, linePatches invoicePatches) error {
	updatedInvoice, err := u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:             invoice.GetInvoiceID(),
		IncludeDeletedLines: true,
		EditFn: func(invoice *billing.StandardInvoice) error {
			// Let's delete lines if needed
			for _, lineID := range linePatches.deletedLines {
				line := invoice.Lines.GetByID(lineID.ID)
				if line == nil {
					return fmt.Errorf("line[%s] not found in the invoice, cannot delete", lineID)
				}

				line.DeletedAt = lo.ToPtr(clock.Now())
			}

			// let's update lines if needed
			for _, targetState := range linePatches.updatedLines {
				targetStandardLine, err := targetState.AsInvoiceLine().AsStandardLine()
				if err != nil {
					return fmt.Errorf("line[%s] is not a standard line, cannot update: %w", targetState.GetID(), err)
				}

				line := invoice.Lines.GetByID(targetStandardLine.ID)
				if line == nil {
					return fmt.Errorf("line[%s] not found in the invoice, cannot update", targetStandardLine.ID)
				}

				// We need to update the quantities of the usage based lines, to compensate for any changes in the period
				// of the line

				updatedQtyLine, err := u.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
					Invoice: invoice,
					Line:    &targetStandardLine,
				})
				if err != nil {
					return fmt.Errorf("recalculating line[%s]: %w", targetStandardLine.ID, err)
				}

				targetStandardLine = *updatedQtyLine

				if ok := invoice.Lines.ReplaceByID(targetStandardLine.ID, &targetStandardLine); !ok {
					return fmt.Errorf("line[%s/%s] not found in the invoice, cannot update", targetStandardLine.ID, lo.FromPtrOr(targetStandardLine.ChildUniqueReferenceID, "nil"))
				}
			}

			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("updating invoice[%s]: %w", invoice.ID, err)
	}

	if updatedInvoice.Lines.NonDeletedLineCount() == 0 {
		if updatedInvoice.Status == billing.StandardInvoiceStatusGathering {
			// Gathering invoice deletion is handled by the service layer if they are empty
			return nil
		}

		// The invoice has no lines, so let's just delete it
		invoice, err := u.billingService.DeleteInvoice(ctx, updatedInvoice.GetInvoiceID())
		if err != nil {
			return fmt.Errorf("deleting empty invoice: %w", err)
		}

		if invoice.Status == billing.StandardInvoiceStatusDeleteFailed {
			u.logger.WarnContext(ctx, "empty invoice deletion failed",
				"invoice.id", invoice.ID,
				"invoice.namespace", invoice.Namespace,
				"validation_issues", strings.Join(
					lo.Map(invoice.ValidationIssues, func(i billing.ValidationIssue, _ int) string {
						return fmt.Sprintf("[id=%s] %s: %s", i.ID, i.Code, i.Message)
					}),
					", "))
		}
	}

	return err
}

func (u *InvoiceUpdater) updateGatheringInvoice(ctx context.Context, invoiceID billing.InvoiceID, linePatches invoicePatches) error {
	return u.billingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
		Invoice:             invoiceID,
		IncludeDeletedLines: true,
		EditFn: func(invoice *billing.GatheringInvoice) error {
			// Let's delete lines if needed
			for _, lineID := range linePatches.deletedLines {
				line, ok := invoice.Lines.GetByID(lineID.ID)
				if !ok {
					return fmt.Errorf("line[%s] not found in the invoice, cannot delete", lineID)
				}

				line.DeletedAt = lo.ToPtr(clock.Now())

				if err := invoice.Lines.ReplaceByID(line); err != nil {
					return fmt.Errorf("setting line[%s]: %w", lineID, err)
				}
			}

			// let's update lines if needed
			for _, targetStateGeneric := range linePatches.updatedLines {
				targetGatheringLine, err := targetStateGeneric.AsInvoiceLine().AsGatheringLine()
				if err != nil {
					return fmt.Errorf("line[%s] is not a gathering line, cannot update: %w", targetStateGeneric.GetID(), err)
				}

				if err := invoice.Lines.ReplaceByID(targetGatheringLine); err != nil {
					return fmt.Errorf("setting line[%s]: %w", targetGatheringLine.ID, err)
				}
			}

			return nil
		},
	})
}

func (u *InvoiceUpdater) updateImmutableInvoice(ctx context.Context, invoice billing.StandardInvoice, linePatches invoicePatches) error {
	invoice, err := u.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	if err != nil {
		return fmt.Errorf("getting invoice: %w", err)
	}

	// Given we don't have credit notes support we can only signal that the invoice would have needed a credit note
	validationIssues := []billing.ValidationIssue{}

	for _, line := range linePatches.deletedLines {
		validationIssues = append(validationIssues,
			newValidationIssueOnLine(invoice.Lines.GetByID(line.ID), "line should be deleted, but the invoice is immutable"),
		)
	}

	for _, targetState := range linePatches.updatedLines {
		existingLine := invoice.Lines.GetByID(targetState.GetID())
		if existingLine == nil {
			return fmt.Errorf("line[%s] not found in the invoice, cannot update", targetState.GetID())
		}

		if isFlatFee(targetState) {
			existingPerUnitAmount, err := getFlatFeePerUnitAmount(existingLine)
			if err != nil {
				return fmt.Errorf("getting flat fee per unit amount: %w", err)
			}

			targetPerUnitAmount, err := getFlatFeePerUnitAmount(targetState)
			if err != nil {
				return fmt.Errorf("getting flat fee per unit amount: %w", err)
			}

			if !existingPerUnitAmount.Equal(targetPerUnitAmount) {
				validationIssues = append(validationIssues,
					newValidationIssueOnLine(existingLine, "flat fee line's per unit amount cannot be changed on immutable invoice (new per unit amount: %s)",
						targetPerUnitAmount.String()),
				)
			}

			continue
		}

		if !targetState.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).Equal(existingLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration)) {
			targetStandardLine, err := targetState.AsInvoiceLine().AsStandardLine()
			if err != nil {
				return fmt.Errorf("line[%s] is not a standard line, cannot update: %w", targetState.GetID(), err)
			}

			// The period of the line has changed => we need to refetch the quantity
			targetStateWithUpdatedQty, err := u.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
				Invoice: &invoice,
				Line:    &targetStandardLine,
			})
			if err != nil {
				return fmt.Errorf("snapshotting quantity for line[%s]: %w", targetState.GetID(), err)
			}

			if !targetStateWithUpdatedQty.UsageBased.Quantity.Equal(lo.FromPtr(existingLine.UsageBased.Quantity)) {
				validationIssues = append(validationIssues,
					newValidationIssueOnLine(existingLine, "usage based line's quantity cannot be changed on immutable invoice (new qty: %s)",
						targetStateWithUpdatedQty.UsageBased.Quantity.String()),
				)
			}
		}
	}

	if len(validationIssues) > 0 {
		// These calculations are not idempontent, as we are only executing it against the in-scope part of the
		// subscription, so we cannot rely on the component based replace features of the validation issues member
		// of the invoice, so let's manually merge the issues.

		mergedValidationIssues, wasChange := u.mergeValidationIssues(invoice, validationIssues)
		if !wasChange {
			return nil
		}

		return u.billingService.UpsertValidationIssues(ctx, billing.UpsertValidationIssuesInput{
			Invoice: invoice.GetInvoiceID(),
			Issues:  mergedValidationIssues,
		})
	}

	return nil
}

func newValidationIssueOnLine(line *billing.StandardLine, message string, a ...any) billing.ValidationIssue {
	if line == nil {
		return billing.ValidationIssue{
			Severity:  billing.ValidationIssueSeverityCritical,
			Message:   "line not found in the invoice, cannot update",
			Code:      billing.ImmutableInvoiceHandlingNotSupportedErrorCode,
			Component: SubscriptionSyncComponentName,
			Path:      "lines/nil",
		}
	}

	return billing.ValidationIssue{
		// We use warning here, to prevent the state machine from being locked up due to present
		// validation errors
		Severity:  billing.ValidationIssueSeverityWarning,
		Message:   fmt.Sprintf(message, a...),
		Code:      billing.ImmutableInvoiceHandlingNotSupportedErrorCode,
		Component: SubscriptionSyncComponentName,
		Path:      fmt.Sprintf("lines/%s", line.ID),
	}
}

func (u *InvoiceUpdater) mergeValidationIssues(invoice billing.StandardInvoice, issues []billing.ValidationIssue) (billing.ValidationIssues, bool) {
	changed := false

	// We don't expect much issues, and this is temporary until we have credits so let's just
	// use this simple approach.

	for _, issue := range issues {
		_, found := lo.Find(invoice.ValidationIssues, func(i billing.ValidationIssue) bool {
			return i.Path == issue.Path && i.Component == SubscriptionSyncComponentName && i.Code == billing.ImmutableInvoiceHandlingNotSupportedErrorCode &&
				i.Message == issue.Message
		})

		if found {
			continue
		}

		changed = true

		invoice.ValidationIssues = append(invoice.ValidationIssues, issue)
	}

	return invoice.ValidationIssues, changed
}

func (u *InvoiceUpdater) upsertSplitLineGroups(ctx context.Context, customerID customer.CustomerID, changes splitLineGroupPatches) error {
	if len(changes.deleted) == 0 && len(changes.updated) == 0 {
		return nil
	}

	// Let's delete split line groups if needed
	for _, groupID := range changes.deleted {
		if err := u.billingService.DeleteSplitLineGroup(ctx, groupID); err != nil {
			return fmt.Errorf("deleting split line group: %w", err)
		}
	}

	// Let's upsert split line groups if needed
	for _, group := range changes.updated {
		if _, err := u.billingService.UpdateSplitLineGroup(ctx, group); err != nil {
			return fmt.Errorf("upserting split line group: %w", err)
		}
	}

	return nil
}

type handleChargesInput struct {
	CustomerID                        customer.CustomerID
	NewCharges                        []charges.Charge
	UpdatedCharges                    []charges.Charge
	DeletedChargesByUniqueReferenceID []string
}

func (i handleChargesInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("customer ID: %w", err)
	}

	return nil
}

type syncChargesInput struct {
	CustomerID customer.CustomerID
	Upserts    []upsertChargeAndAssociateLinesPatch
	Deletes    []string
}

func (i syncChargesInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("customer ID: %w", err)
	}

	// Validate uniqueness
	upsertUniqueReferenceIDs, err := slicesx.MapWithErr(i.Upserts, func(upsert upsertChargeAndAssociateLinesPatch) (string, error) {
		ref := upsert.Charge.Intent.UniqueReferenceID
		if ref == nil {
			return "", fmt.Errorf("upsert charge has no unique reference ID")
		}

		return *ref, nil
	})
	if err != nil {
		return fmt.Errorf("mapping upsert charges: %w", err)
	}

	if len(lo.Uniq(upsertUniqueReferenceIDs)) != len(upsertUniqueReferenceIDs) {
		return fmt.Errorf("upsert charges have duplicate unique reference IDs")
	}

	if len(lo.Uniq(i.Deletes)) != len(i.Deletes) {
		return fmt.Errorf("delete charges have duplicate unique reference IDs")
	}

	allUniqueReferenceIDs := slices.Concat(upsertUniqueReferenceIDs, i.Deletes)
	if len(lo.Uniq(allUniqueReferenceIDs)) != len(allUniqueReferenceIDs) {
		return fmt.Errorf("upsert and delete charges have duplicate unique reference IDs")
	}

	return nil
}

// syncCharges is responsible for upserting charges and associating them to lines and split line groups
// this is a temporary solution so that the charges are available in the database (backfill)
//
// once charges are fully functional, the charge service will handle the state instead of this upsert
func (u *InvoiceUpdater) syncCharges(ctx context.Context, input syncChargesInput) error {
	if u.chargesService == nil {
		u.logger.WarnContext(ctx, "charges service is not available, skipping charge sync", "customer.id", input.CustomerID)
		return nil
	}

	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating sync charges input: %w", err)
	}

	if len(input.Upserts) == 0 && len(input.Deletes) == 0 {
		return nil
	}

	lineIDsToAssociate := make([]string, 0, len(input.Upserts))
	for _, upsert := range input.Upserts {
		lineIDsToAssociate = append(lineIDsToAssociate, lo.Map(upsert.LinesIDsToAssociate, func(lineID billing.LineID, _ int) string {
			return lineID.ID
		})...)
	}

	// Let's fetch the lines with invoice headers to populate the realization statuses
	linesWithInvoiceHeaders, err := u.billingService.GetInvoiceLinesWithInvoiceHeaders(ctx, billing.GetInvoiceLinesWithInvoiceHeadersInput{
		Namespace: input.CustomerID.Namespace,
		LineIDs:   lineIDsToAssociate,
	})
	if err != nil {
		return fmt.Errorf("getting invoice lines with invoice headers: %w", err)
	}

	linesByLineID := lo.SliceToMap(linesWithInvoiceHeaders, func(lineWithInvoiceHeader billing.LineWithInvoiceHeader) (string, billing.LineWithInvoiceHeader) {
		return lineWithInvoiceHeader.Line.GetID(), lineWithInvoiceHeader
	})

	upserts, err := slicesx.MapWithErr(input.Upserts, func(upsert upsertChargeAndAssociateLinesPatch) (upsertChargeAndAssociateLinesPatch, error) {
		return upsertWithRealizations(upsert, linesByLineID)
	})
	if err != nil {
		return fmt.Errorf("mapping upsert charges: %w", err)
	}

	// Let's upsert charges
	upsertedCharges, err := u.chargesService.UpsertChargesByChildUniqueReferenceID(ctx, charges.UpsertChargesByChildUniqueReferenceIDInput{
		Customer: input.CustomerID,
		Charges: lo.Map(upserts, func(upsert upsertChargeAndAssociateLinesPatch, _ int) charges.Charge {
			return upsert.Charge
		}),
	})
	if err != nil {
		return fmt.Errorf("upserting charges: %w", err)
	}

	upsertedChargesByUniqueReferenceID := make(map[string]charges.Charge)
	for _, charge := range upsertedCharges {
		ref := charge.Intent.UniqueReferenceID
		if ref == nil {
			return fmt.Errorf("upsert charge has no unique reference ID")
		}

		upsertedChargesByUniqueReferenceID[*ref] = charge
	}

	// Let's execute the associations for lines
	lineIDToChargeID := make(map[string]string)
	groupIDToChargeID := make(map[string]string)
	for _, upsert := range upserts {
		uniqueReferenceID := *upsert.Charge.Intent.UniqueReferenceID
		charge, ok := upsertedChargesByUniqueReferenceID[uniqueReferenceID]
		if !ok {
			return fmt.Errorf("upsert charge with unique reference ID %s not found", uniqueReferenceID)
		}

		for _, lineID := range upsert.LinesIDsToAssociate {
			lineIDToChargeID[lineID.ID] = charge.ID
		}

		if upsert.SplitLineGroupIDToAssociate != nil {
			groupIDToChargeID[upsert.SplitLineGroupIDToAssociate.ID] = charge.ID
		}
	}

	err = u.billingService.SetChargeIDsOnInvoiceLines(ctx, billing.SetChargeIDsOnInvoiceLinesInput{
		Namespace:        input.CustomerID.Namespace,
		LineIDToChargeID: lineIDToChargeID,
	})
	if err != nil {
		return fmt.Errorf("setting charge IDs on invoice lines: %w", err)
	}

	err = u.billingService.SetChargeIDsOnSplitlineGroups(ctx, billing.SetChargeIDsOnSplitlineGroupsInput{
		Namespace:         input.CustomerID.Namespace,
		GroupIDToChargeID: groupIDToChargeID,
	})
	if err != nil {
		return fmt.Errorf("setting charge IDs on split line groups: %w", err)
	}

	// Let's (soft) delete charges
	err = u.chargesService.DeleteChargesByUniqueReferenceID(ctx, charges.DeleteChargesByUniqueReferenceIDInput{
		Customer:           input.CustomerID,
		UniqueReferenceIDs: input.Deletes,
	})
	if err != nil {
		return fmt.Errorf("deleting charges: %w", err)
	}

	return nil
}
