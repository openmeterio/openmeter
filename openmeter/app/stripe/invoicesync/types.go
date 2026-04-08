package invoicesync

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

// SyncPlanPhase represents which invoice state machine phase this plan is for.
type SyncPlanPhase string

const (
	SyncPlanPhaseDraft   SyncPlanPhase = "draft"
	SyncPlanPhaseIssuing SyncPlanPhase = "issuing"
	SyncPlanPhaseDelete  SyncPlanPhase = "delete"
)

func (p SyncPlanPhase) Values() []string {
	return []string{
		string(SyncPlanPhaseDraft),
		string(SyncPlanPhaseIssuing),
		string(SyncPlanPhaseDelete),
	}
}

// OpType represents the type of Stripe API operation.
type OpType string

const (
	OpTypeInvoiceCreate   OpType = "invoice_create"
	OpTypeInvoiceUpdate   OpType = "invoice_update"
	OpTypeInvoiceDelete   OpType = "invoice_delete"
	OpTypeInvoiceFinalize OpType = "invoice_finalize"
	OpTypeLineItemAdd     OpType = "line_item_add"
	OpTypeLineItemUpdate  OpType = "line_item_update"
	OpTypeLineItemRemove  OpType = "line_item_remove"
)

func (t OpType) Values() []string {
	return []string{
		string(OpTypeInvoiceCreate),
		string(OpTypeInvoiceUpdate),
		string(OpTypeInvoiceDelete),
		string(OpTypeInvoiceFinalize),
		string(OpTypeLineItemAdd),
		string(OpTypeLineItemUpdate),
		string(OpTypeLineItemRemove),
	}
}

// OpStatus represents the status of an individual operation.
type OpStatus string

const (
	OpStatusPending   OpStatus = "pending"
	OpStatusCompleted OpStatus = "completed"
	OpStatusFailed    OpStatus = "failed"
)

func (s OpStatus) Values() []string {
	return []string{
		string(OpStatusPending),
		string(OpStatusCompleted),
		string(OpStatusFailed),
	}
}

// PlanStatus represents the overall status of a sync plan.
type PlanStatus string

const (
	PlanStatusPending   PlanStatus = "pending"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
)

func (s PlanStatus) Values() []string {
	return []string{
		string(PlanStatusPending),
		string(PlanStatusExecuting),
		string(PlanStatusCompleted),
		string(PlanStatusFailed),
	}
}

// Metadata keys used to signal sync plan completion on the invoice.
// These are aliases for the generic billing metadata keys.
const (
	MetadataKeyDraftSyncPlanID        = billing.MetadataKeyDraftSyncPlanID
	MetadataKeyDraftSyncCompletedAt   = billing.MetadataKeyDraftSyncCompletedAt
	MetadataKeyIssuingSyncPlanID      = billing.MetadataKeyIssuingSyncPlanID
	MetadataKeyIssuingSyncCompletedAt = billing.MetadataKeyIssuingSyncCompletedAt
)

// SyncPlan represents a persistent, ordered set of Stripe operations for a single sync session.
type SyncPlan struct {
	ID          string
	Namespace   string
	InvoiceID   string
	AppID       string
	SessionID   string
	Phase       SyncPlanPhase
	Status      PlanStatus
	Operations  []SyncOperation
	Error       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// SyncOperation represents a single Stripe API call within a sync plan.
type SyncOperation struct {
	ID             string
	PlanID         string
	Sequence       int
	Type           OpType
	Payload        json.RawMessage
	IdempotencyKey string
	Status         OpStatus
	StripeResponse json.RawMessage
	Error          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// GenerateIdempotencyKey produces a deterministic idempotency key for a Stripe API call.
// Fields are length-prefixed so invoice/session IDs cannot be split ambiguously at delimiter boundaries.
func GenerateIdempotencyKey(invoiceID, sessionID string, sequence int, opType OpType) string {
	var buf bytes.Buffer
	writeFramedString(&buf, invoiceID)
	writeFramedString(&buf, sessionID)
	writeFramedString(&buf, string(opType))
	_ = binary.Write(&buf, binary.BigEndian, int64(sequence))
	h := sha256.New()
	h.Write(buf.Bytes())
	return fmt.Sprintf("%x", h.Sum(nil))
}

func writeFramedString(buf *bytes.Buffer, s string) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(s)))
	buf.WriteString(s)
}

// InvoiceCreatePayload is the payload for OpTypeInvoiceCreate.
type InvoiceCreatePayload struct {
	AppID                        string                   `json:"app_id"`
	Namespace                    string                   `json:"namespace"`
	CustomerID                   string                   `json:"customer_id"`
	InvoiceID                    string                   `json:"invoice_id"`
	AutomaticTaxEnabled          bool                     `json:"automatic_tax_enabled"`
	CollectionMethod             billing.CollectionMethod `json:"collection_method"`
	Currency                     string                   `json:"currency"`
	DaysUntilDue                 *int64                   `json:"days_until_due,omitempty"`
	StripeCustomerID             string                   `json:"stripe_customer_id"`
	StripeDefaultPaymentMethodID string                   `json:"stripe_default_payment_method_id,omitempty"`
}

// InvoiceUpdatePayload is the payload for OpTypeInvoiceUpdate.
type InvoiceUpdatePayload struct {
	StripeInvoiceID     string `json:"stripe_invoice_id"`
	AutomaticTaxEnabled bool   `json:"automatic_tax_enabled"`
}

// InvoiceDeletePayload is the payload for OpTypeInvoiceDelete.
type InvoiceDeletePayload struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
}

// InvoiceFinalizePayload is the payload for OpTypeInvoiceFinalize.
type InvoiceFinalizePayload struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
	AutoAdvance     bool   `json:"auto_advance"`
	TaxEnforced     bool   `json:"tax_enforced"`
}

// LineItemAddPayload is the payload for OpTypeLineItemAdd.
type LineItemAddPayload struct {
	StripeInvoiceID string           `json:"stripe_invoice_id"`
	Lines           []LineItemParams `json:"lines"`
}

// LineItemUpdatePayload is the payload for OpTypeLineItemUpdate.
type LineItemUpdatePayload struct {
	StripeInvoiceID string                 `json:"stripe_invoice_id"`
	Lines           []LineItemUpdateParams `json:"lines"`
}

// LineItemRemovePayload is the payload for OpTypeLineItemRemove.
type LineItemRemovePayload struct {
	StripeInvoiceID string   `json:"stripe_invoice_id"`
	LineIDs         []string `json:"line_ids"`
}

// LineItemParams represents the parameters for adding a line item to a Stripe invoice.
type LineItemParams struct {
	Description string            `json:"description"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	CustomerID  string            `json:"customer_id"`
	PeriodStart int64             `json:"period_start"`
	PeriodEnd   int64             `json:"period_end"`
	Metadata    map[string]string `json:"metadata"`
	TaxBehavior *string           `json:"tax_behavior,omitempty"`
	TaxCode     *string           `json:"tax_code,omitempty"`
}

// LineItemUpdateParams represents the parameters for updating a line item on a Stripe invoice.
type LineItemUpdateParams struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	PeriodStart int64             `json:"period_start"`
	PeriodEnd   int64             `json:"period_end"`
	Metadata    map[string]string `json:"metadata"`
	TaxBehavior *string           `json:"tax_behavior,omitempty"`
	TaxCode     *string           `json:"tax_code,omitempty"`
}

// InvoiceCreateResponse is stored in StripeResponse after a successful InvoiceCreate operation.
type InvoiceCreateResponse struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
	InvoiceNumber   string `json:"invoice_number"`
}

// LineItemAddResponse is stored in StripeResponse after a successful LineItemAdd operation.
type LineItemAddResponse struct {
	// Maps OpenMeter line ID -> Stripe line item ID
	LineExternalIDs         map[string]string `json:"line_external_ids"`
	LineDiscountExternalIDs map[string]string `json:"line_discount_external_ids"`
}

// InvoiceUpdateResponse is stored in StripeResponse after a successful InvoiceUpdate operation.
type InvoiceUpdateResponse struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
	InvoiceNumber   string `json:"invoice_number"`
}

// InvoiceDeleteResponse is stored in StripeResponse after a successful InvoiceDelete operation.
type InvoiceDeleteResponse struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
}

// LineItemUpdateResponse is stored in StripeResponse after a successful LineItemUpdate operation.
type LineItemUpdateResponse struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
}

// LineItemRemoveResponse is stored in StripeResponse after a successful LineItemRemove operation.
type LineItemRemoveResponse struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
}

// InvoiceFinalizeResponse is stored in StripeResponse after a successful InvoiceFinalize operation.
type InvoiceFinalizeResponse struct {
	InvoiceNumber     string  `json:"invoice_number"`
	PaymentExternalID *string `json:"payment_external_id,omitempty"`
}
