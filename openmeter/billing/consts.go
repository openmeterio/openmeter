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
	// AnnotationValueReasonOveragePlaceholder indicates the line reserves invoice scheduling
	// for a custom-currency overage whose fiat amount is not materialized yet.
	AnnotationValueReasonOveragePlaceholder = "overage-placeholder"
	// AnnotationValueReasonOverage indicates the line contains usage not covered by credits.
	AnnotationValueReasonOverage = "overage"
)
