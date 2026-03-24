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
