package service

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
	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoiceUpdater struct {
	billingService billing.Service
	logger         *slog.Logger
}

func NewInvoiceUpdater(billingService billing.Service, logger *slog.Logger) *InvoiceUpdater {
	return &InvoiceUpdater{
		billingService: billingService,
		logger:         logger,
	}
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
		invoice, err := u.billingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: customerID.Namespace,
				ID:        invoiceID,
			},
			Expand: billing.InvoiceExpand{},
		})
		if err != nil {
			return fmt.Errorf("getting invoice: %w", err)
		}

		if !invoice.StatusDetails.Immutable {
			if err := u.updateMutableInvoice(ctx, invoice, linePatches); err != nil {
				return fmt.Errorf("updating mutable invoice: %w", err)
			}
			continue
		}

		if err := u.updateImmutableInvoice(ctx, invoice, linePatches); err != nil {
			return fmt.Errorf("updating immutable invoice: %w", err)
		}
	}

	// Let's update split line groups
	err = u.upsertSplitLineGroups(ctx, customerID, patchesParsed.splitLineGroups)
	if err != nil {
		return fmt.Errorf("upserting split line groups: %w", err)
	}

	return nil
}

type patchesParsed struct {
	newLines []*billing.StandardLine

	updatedLinesByInvoiceID map[string]invoicePatches

	splitLineGroups splitLineGroupPatches
}

type invoicePatches struct {
	updatedLines []*billing.StandardLine
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

			parsed.newLines = append(parsed.newLines, &create.Line)
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

			lineUpdates := parsed.updatedLinesByInvoiceID[update.TargetState.InvoiceID]
			lineUpdates.updatedLines = append(lineUpdates.updatedLines, update.TargetState)
			parsed.updatedLinesByInvoiceID[update.TargetState.InvoiceID] = lineUpdates
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
		default:
			return patchesParsed{}, fmt.Errorf("unexpected patch operation: %s", patch.Op())
		}
	}

	return parsed, nil
}

func (u *InvoiceUpdater) provisionUpcomingLines(ctx context.Context, customerID customer.CustomerID, lines []*billing.StandardLine) error {
	if len(lines) == 0 {
		return nil
	}

	linesByCurrency := lo.GroupBy(lines, func(l *billing.StandardLine) currencyx.Code {
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

func (u *InvoiceUpdater) updateMutableInvoice(ctx context.Context, invoice billing.StandardInvoice, linePatches invoicePatches) error {
	updatedInvoice, err := u.billingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice:             invoice.InvoiceID(),
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
				line := invoice.Lines.GetByID(targetState.ID)
				if line == nil {
					return fmt.Errorf("line[%s] not found in the invoice, cannot update", targetState.ID)
				}

				// update
				if invoice.Status != billing.StandardInvoiceStatusGathering {
					// We need to update the quantities of the usage based lines, to compensate for any changes in the period
					// of the line

					updatedQtyLine, err := u.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
						Invoice: invoice,
						Line:    targetState,
					})
					if err != nil {
						return fmt.Errorf("recalculating line[%s]: %w", targetState.ID, err)
					}

					targetState = updatedQtyLine
				}

				if ok := invoice.Lines.ReplaceByID(targetState.ID, targetState); !ok {
					return fmt.Errorf("line[%s/%s] not found in the invoice, cannot update", targetState.ID, lo.FromPtrOr(targetState.ChildUniqueReferenceID, "nil"))
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
		invoice, err := u.billingService.DeleteInvoice(ctx, updatedInvoice.InvoiceID())
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

func (u *InvoiceUpdater) updateImmutableInvoice(ctx context.Context, invoice billing.StandardInvoice, linePatches invoicePatches) error {
	invoice, err := u.billingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
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
		existingLine := invoice.Lines.GetByID(targetState.ID)
		if existingLine == nil {
			return fmt.Errorf("line[%s] not found in the invoice, cannot update", targetState.ID)
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

		if !targetState.Period.Truncate(streaming.MinimumWindowSizeDuration).Equal(existingLine.Period.Truncate(streaming.MinimumWindowSizeDuration)) {
			// The period of the line has changed => we need to refetch the quantity
			targetStateWithUpdatedQty, err := u.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
				Invoice: &invoice,
				Line:    targetState,
			})
			if err != nil {
				return fmt.Errorf("recalculating line[%s]: %w", targetState.ID, err)
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
			Invoice: invoice.InvoiceID(),
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
