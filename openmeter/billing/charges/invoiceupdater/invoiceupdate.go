package invoiceupdater

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

const invoiceUpdaterComponentName billing.ComponentName = "charges.invoiceupdater"

type Updater struct {
	billingService billing.Service
	logger         *slog.Logger
}

func New(billingService billing.Service, logger *slog.Logger) *Updater {
	return &Updater{
		billingService: billingService,
		logger:         logger,
	}
}

func (u *Updater) ApplyPatches(ctx context.Context, customerID customer.CustomerID, patches []Patch) error {
	patchesParsed, err := u.parsePatches(patches)
	if err != nil {
		return fmt.Errorf("parsing patches: %w", err)
	}

	err = u.provisionUpcomingLines(ctx, customerID, patchesParsed.newLines)
	if err != nil {
		return fmt.Errorf("provisioning upcoming lines: %w", err)
	}

	if err := u.resolveGatheringLineDeletesByChargeID(ctx, customerID, &patchesParsed); err != nil {
		return fmt.Errorf("resolving gathering line deletes by charge ID: %w", err)
	}

	if err := u.resolveGatheringLineUpdatesByChargeID(ctx, customerID, &patchesParsed); err != nil {
		return fmt.Errorf("resolving gathering line updates by charge ID: %w", err)
	}

	invoicesByID, err := u.listInvoicesByID(ctx, customerID.Namespace, lo.Keys(patchesParsed.updatedLinesByInvoiceID))
	if err != nil {
		return fmt.Errorf("listing invoices: %w", err)
	}

	for invoiceID, linePatches := range patchesParsed.updatedLinesByInvoiceID {
		namespacedInvoiceID := billing.InvoiceID{
			Namespace: customerID.Namespace,
			ID:        invoiceID,
		}

		invoice, ok := invoicesByID[invoiceID]
		if !ok {
			return fmt.Errorf("getting invoice: invoice[%s/%s] not found", customerID.Namespace, invoiceID)
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

	return nil
}

func (u *Updater) listInvoicesByID(ctx context.Context, namespace string, invoiceIDs []string) (map[string]billing.Invoice, error) {
	if len(invoiceIDs) == 0 {
		return map[string]billing.Invoice{}, nil
	}

	resp, err := u.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces:     []string{namespace},
		IDs:            invoiceIDs,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, err
	}

	invoicesByID := make(map[string]billing.Invoice, len(resp.Items))
	for _, invoice := range resp.Items {
		genericInvoice, err := invoice.AsGenericInvoice()
		if err != nil {
			return nil, fmt.Errorf("converting invoice to generic invoice: %w", err)
		}

		invoicesByID[genericInvoice.GetID()] = invoice
	}

	return invoicesByID, nil
}

func (u *Updater) LogPatches(patches []Patch, invoicesByID map[string]billing.Invoice) {
	suppressedDryRunPatches := 0

	for _, patch := range patches {
		if !isDryRunLoggablePatch(patch, invoicesByID) {
			suppressedDryRunPatches++
			continue
		}

		patch.Log(u.logger)
	}

	if suppressedDryRunPatches > 0 {
		u.logger.Info("suppressed dry run patches", "count", suppressedDryRunPatches)
	}
}

func isDryRunLoggablePatch(patch Patch, invoicesByID map[string]billing.Invoice) bool {
	switch patch.Op() {
	case PatchOpLineCreate:
		createPatch, err := patch.AsCreateLinePatch()
		if err != nil {
			return true
		}

		// Missing current-period pending lines are expected catch-up work for subscription
		// sync. Dry-run output should focus on actionable drift on already materialized
		// resources, so we suppress create-line logs only when they belong to the current
		// billing period.
		return !isCurrentBillingPeriod(createPatch.Line)
	case PatchOpLineDelete:
		deletePatch, err := patch.AsDeleteLinePatch()
		if err != nil {
			return true
		}

		return isMutableInvoice(deletePatch.InvoiceID, invoicesByID)
	case PatchOpLineUpdate:
		updatePatch, err := patch.AsUpdateLinePatch()
		if err != nil {
			return true
		}

		return isMutableInvoice(updatePatch.TargetState.GetInvoiceID(), invoicesByID)
	case PatchOpDeleteGatheringLineByChargeID:
		return true
	default:
		return true
	}
}

func isCurrentBillingPeriod(line billing.GatheringLine) bool {
	subscriptionRef := line.GetSubscriptionReference()
	if subscriptionRef == nil {
		return false
	}

	now := clock.Now().UTC()
	billingPeriod := subscriptionRef.BillingPeriod

	return !now.Before(billingPeriod.From) && now.Before(billingPeriod.To)
}

func isMutableInvoice(invoiceID string, invoicesByID map[string]billing.Invoice) bool {
	invoice, ok := invoicesByID[invoiceID]
	if !ok {
		return true
	}

	if invoice.Type() == billing.InvoiceTypeGathering {
		return true
	}

	standardInvoice, err := invoice.AsStandardInvoice()
	if err != nil {
		return true
	}

	return !standardInvoice.StatusDetails.Immutable
}

type patchesParsed struct {
	newLines []billing.GatheringLine

	updatedLinesByInvoiceID map[string]invoicePatches

	gatheringLineDeletesByChargeID []string
	gatheringLineUpdatesByChargeID map[string]PatchUpdateGatheringLineByChargeID
}

type invoicePatches struct {
	updatedLines []billing.GenericInvoiceLine
	deletedLines []billing.LineID
}

func (u *Updater) parsePatches(patches []Patch) (patchesParsed, error) {
	parsed := patchesParsed{
		updatedLinesByInvoiceID:        make(map[string]invoicePatches),
		gatheringLineUpdatesByChargeID: make(map[string]PatchUpdateGatheringLineByChargeID),
	}

	for _, patch := range patches {
		switch patch.Op() {
		case PatchOpLineCreate:
			create, err := patch.AsCreateLinePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting line: %w", err)
			}

			parsed.newLines = append(parsed.newLines, create.Line)
		case PatchOpLineDelete:
			deletePatch, err := patch.AsDeleteLinePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting line: %w", err)
			}

			lineUpdates := parsed.updatedLinesByInvoiceID[deletePatch.InvoiceID]
			lineUpdates.deletedLines = append(lineUpdates.deletedLines, deletePatch.Line)
			parsed.updatedLinesByInvoiceID[deletePatch.InvoiceID] = lineUpdates
		case PatchOpLineUpdate:
			update, err := patch.AsUpdateLinePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting line: %w", err)
			}

			lineUpdates := parsed.updatedLinesByInvoiceID[update.TargetState.GetInvoiceID()]
			lineUpdates.updatedLines = append(lineUpdates.updatedLines, update.TargetState)
			parsed.updatedLinesByInvoiceID[update.TargetState.GetInvoiceID()] = lineUpdates
		case PatchOpDeleteGatheringLineByChargeID:
			deletePatch, err := patch.AsDeleteGatheringLineByChargeIDPatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting charge ID: %w", err)
			}

			parsed.gatheringLineDeletesByChargeID = append(parsed.gatheringLineDeletesByChargeID, deletePatch.ChargeID)
		case PatchOpUpdateGatheringLineByChargeID:
			updatePatch, err := patch.AsUpdateGatheringLineByChargeIDPatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting gathering line update: %w", err)
			}

			parsed.gatheringLineUpdatesByChargeID[updatePatch.ChargeID] = updatePatch
		default:
			return patchesParsed{}, fmt.Errorf("unexpected patch operation: %s", patch.Op())
		}
	}

	return parsed, nil
}

func (u *Updater) provisionUpcomingLines(ctx context.Context, customerID customer.CustomerID, lines []billing.GatheringLine) error {
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

func (u *Updater) resolveGatheringLineDeletesByChargeID(ctx context.Context, customerID customer.CustomerID, parsed *patchesParsed) error {
	if len(parsed.gatheringLineDeletesByChargeID) == 0 {
		return nil
	}

	chargeIDs := make(map[string]struct{}, len(parsed.gatheringLineDeletesByChargeID))
	for _, chargeID := range parsed.gatheringLineDeletesByChargeID {
		chargeIDs[chargeID] = struct{}{}
	}

	invoices, err := u.billingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{customerID.Namespace},
		Customers:  []string{customerID.ID},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
		},
	})
	if err != nil {
		return fmt.Errorf("listing gathering invoices: %w", err)
	}

	for _, invoice := range invoices.Items {
		for _, line := range invoice.Lines.OrEmpty() {
			if line.DeletedAt != nil || line.ChargeID == nil {
				continue
			}

			if _, ok := chargeIDs[*line.ChargeID]; !ok {
				continue
			}

			lineUpdates := parsed.updatedLinesByInvoiceID[invoice.ID]
			lineUpdates.deletedLines = append(lineUpdates.deletedLines, line.GetLineID())
			parsed.updatedLinesByInvoiceID[invoice.ID] = lineUpdates
		}
	}

	return nil
}

func (u *Updater) resolveGatheringLineUpdatesByChargeID(ctx context.Context, customerID customer.CustomerID, parsed *patchesParsed) error {
	if len(parsed.gatheringLineUpdatesByChargeID) == 0 {
		return nil
	}

	invoices, err := u.billingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{customerID.Namespace},
		Customers:  []string{customerID.ID},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
		},
	})
	if err != nil {
		return fmt.Errorf("listing gathering invoices: %w", err)
	}

	for _, invoice := range invoices.Items {
		for _, line := range invoice.Lines.OrEmpty() {
			if line.DeletedAt != nil || line.ChargeID == nil {
				continue
			}

			updatePatch, ok := parsed.gatheringLineUpdatesByChargeID[*line.ChargeID]
			if !ok {
				continue
			}

			line.ServicePeriod.To = updatePatch.ServicePeriodTo
			line.InvoiceAt = updatePatch.ServicePeriodTo

			genericLine, err := line.AsInvoiceLine().AsGenericLine()
			if err != nil {
				return fmt.Errorf("converting gathering line[%s] to generic line: %w", line.ID, err)
			}

			lineUpdates := parsed.updatedLinesByInvoiceID[invoice.ID]
			lineUpdates.updatedLines = append(lineUpdates.updatedLines, genericLine)
			parsed.updatedLinesByInvoiceID[invoice.ID] = lineUpdates
		}
	}

	return nil
}

func (u *Updater) updateMutableStandardInvoice(ctx context.Context, invoice billing.StandardInvoice, linePatches invoicePatches) error {
	updatedInvoice, err := u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:             invoice.GetInvoiceID(),
		IncludeDeletedLines: true,
		EditFn: func(invoice *billing.StandardInvoice) error {
			for _, lineID := range linePatches.deletedLines {
				line := invoice.Lines.GetByID(lineID.ID)
				if line == nil {
					return fmt.Errorf("line[%s] not found in the invoice, cannot delete", lineID)
				}

				line.DeletedAt = lo.ToPtr(clock.Now())
			}

			for _, targetState := range linePatches.updatedLines {
				targetStandardLine, err := targetState.AsInvoiceLine().AsStandardLine()
				if err != nil {
					return fmt.Errorf("line[%s] is not a standard line, cannot update: %w", targetState.GetID(), err)
				}

				line := invoice.Lines.GetByID(targetStandardLine.ID)
				if line == nil {
					return fmt.Errorf("line[%s] not found in the invoice, cannot update", targetStandardLine.ID)
				}

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
			return nil
		}

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

	return nil
}

func (u *Updater) updateGatheringInvoice(ctx context.Context, invoiceID billing.InvoiceID, linePatches invoicePatches) error {
	return u.billingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
		Invoice:             invoiceID,
		IncludeDeletedLines: true,
		EditFn: func(invoice *billing.GatheringInvoice) error {
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

func (u *Updater) updateImmutableInvoice(ctx context.Context, invoice billing.StandardInvoice, linePatches invoicePatches) error {
	invoice, err := u.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	if err != nil {
		return fmt.Errorf("getting invoice: %w", err)
	}

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

		if IsFlatFee(targetState) {
			existingPerUnitAmount, err := GetFlatFeePerUnitAmount(existingLine)
			if err != nil {
				return fmt.Errorf("getting flat fee per unit amount: %w", err)
			}

			targetPerUnitAmount, err := GetFlatFeePerUnitAmount(targetState)
			if err != nil {
				return fmt.Errorf("getting flat fee per unit amount: %w", err)
			}

			if !existingPerUnitAmount.Equal(targetPerUnitAmount) {
				validationIssues = append(validationIssues,
					newValidationIssueOnLine(existingLine, "flat fee line's per unit amount cannot be changed on immutable invoice (new per unit amount: %s)",
						targetPerUnitAmount.String()),
				)

				continue
			}

			if !targetState.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).Equal(existingLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration)) {
				validationIssues = append(validationIssues,
					newValidationIssueOnLine(existingLine, "flat fee line's service period cannot be changed on immutable invoice"),
				)
			}

			continue
		}

		if !targetState.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).Equal(existingLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration)) {
			targetStandardLine, err := targetState.AsInvoiceLine().AsStandardLine()
			if err != nil {
				return fmt.Errorf("line[%s] is not a standard line, cannot update: %w", targetState.GetID(), err)
			}

			targetStateWithUpdatedQty, err := u.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
				Invoice: &invoice,
				Line:    &targetStandardLine,
			})
			if err != nil {
				return fmt.Errorf("snapshotting quantity for line[%s]: %w", targetState.GetID(), err)
			}

			existingQty := existingLine.UsageBased.Quantity
			if existingQty == nil || !targetStateWithUpdatedQty.UsageBased.Quantity.Equal(*existingQty) {
				validationIssues = append(validationIssues,
					newValidationIssueOnLine(existingLine, "usage based line's quantity cannot be changed on immutable invoice (new qty: %s)",
						targetStateWithUpdatedQty.UsageBased.Quantity.String()),
				)
			}
		}
	}

	if len(validationIssues) > 0 {
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
			Component: invoiceUpdaterComponentName,
			Path:      "lines/nil",
		}
	}

	return billing.ValidationIssue{
		Severity:  billing.ValidationIssueSeverityWarning,
		Message:   fmt.Sprintf(message, a...),
		Code:      billing.ImmutableInvoiceHandlingNotSupportedErrorCode,
		Component: invoiceUpdaterComponentName,
		Path:      fmt.Sprintf("lines/%s", line.ID),
	}
}

func (u *Updater) mergeValidationIssues(invoice billing.StandardInvoice, issues []billing.ValidationIssue) (billing.ValidationIssues, bool) {
	changed := false

	for _, issue := range issues {
		_, found := lo.Find(invoice.ValidationIssues, func(i billing.ValidationIssue) bool {
			return i.Path == issue.Path && i.Component == invoiceUpdaterComponentName && i.Code == billing.ImmutableInvoiceHandlingNotSupportedErrorCode &&
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
