package billing

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	// AnnotationSubscriptionSyncIgnore is used to mark a line or hierarchy as ignored in subscription syncing.
	// Should be used in case there is a breaking change in the subscription synchronization process, preventing billing
	// from issuing credit notes for the past periods.
	AnnotationSubscriptionSyncIgnore = "billing.subscription.sync.ignore"

	// AnnotationSubscriptionSyncForceContinuousLines is used to force the creation of continuous subscription item lines.
	// If the sync process finds a previously existing line with this annotation, and the next line generated will not start at the end of the previously
	// found line, the sync process will adjust the start of the next line to the end of the previously found line, so that we don't have gaps in the
	// invoices.
	AnnotationSubscriptionSyncForceContinuousLines = "billing.subscription.sync.force-continuous-lines"

	// AnnotationInvoiceLineCorrection marks a billing-owned line whose amount reverses part of an immutable invoice line.
	AnnotationInvoiceLineCorrection = "billing.invoice.line.correction"
	// AnnotationInvoiceLineCorrectionOfInvoiceID identifies the immutable invoice containing the corrected line.
	AnnotationInvoiceLineCorrectionOfInvoiceID = "billing.invoice.line.correction.of-invoice-id"
	// AnnotationInvoiceLineCorrectionOfLineID identifies the immutable invoice line being corrected.
	AnnotationInvoiceLineCorrectionOfLineID = "billing.invoice.line.correction.of-line-id"
)

// IsInvoiceLineCorrection reports whether the line has correction semantics. Correction lines keep
// ordinary prices non-negative and apply the reversing sign during billing-owned line calculation.
func IsInvoiceLineCorrection(annotations models.Annotations) bool {
	return annotations.GetBool(AnnotationInvoiceLineCorrection)
}

func validateInvoiceLineCorrectionAnnotations(annotations models.Annotations) error {
	if !IsInvoiceLineCorrection(annotations) {
		return nil
	}

	if invoiceID, ok := annotations.GetString(AnnotationInvoiceLineCorrectionOfInvoiceID); !ok || invoiceID == "" {
		return errors.New("correction source invoice id is required")
	}

	if lineID, ok := annotations.GetString(AnnotationInvoiceLineCorrectionOfLineID); !ok || lineID == "" {
		return errors.New("correction source line id is required")
	}

	return nil
}
