package billing

// Annotation keys for gathering invoice lines.
const (
	// AnnotationKeyTaxable indicates whether a line is taxable.
	// Value should be "true" or "false".
	AnnotationKeyTaxable = "openmeter.io/taxable"

	// AnnotationKeyReason indicates the reason for a line.
	// Common values include "credit-purchase", "usage", "flat-fee", etc.
	AnnotationKeyReason = "openmeter.io/reason"
)

// Annotation values for gathering invoice lines.
const (
	// AnnotationValueReasonCreditPurchase indicates the line is for a credit purchase.
	AnnotationValueReasonCreditPurchase = "credit-purchase"
)

// Metadata keys used on invoices to track sync completion.
// These are set by invoicing apps (e.g., Stripe, custom invoicing) to signal
// that an async sync operation has completed.
const (
	MetadataKeyDraftSyncCompletedAt   = "openmeter.io/billing/draft-sync-completed-at"
	MetadataKeyDraftSyncPlanID        = "openmeter.io/billing/draft-sync-plan-id"
	MetadataKeyIssuingSyncCompletedAt = "openmeter.io/billing/issuing-sync-completed-at"
	MetadataKeyIssuingSyncPlanID      = "openmeter.io/billing/issuing-sync-plan-id"
)
