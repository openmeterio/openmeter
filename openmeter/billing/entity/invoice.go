package billingentity

import (
	"fmt"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

type InvoiceType string

const (
	InvoiceTypeStandard   InvoiceType = InvoiceType(bill.InvoiceTypeStandard)
	InvoiceTypeCreditNote InvoiceType = InvoiceType(bill.InvoiceTypeCreditNote)
)

func (t InvoiceType) Values() []string {
	return []string{
		string(InvoiceTypeStandard),
		string(InvoiceTypeCreditNote),
	}
}

func (t InvoiceType) Validate() error {
	for _, status := range t.Values() {
		if string(t) == status {
			return nil
		}
	}

	return fmt.Errorf("invalid invoice type: %s", t)
}

// TODO: remove with gobl (once the calculations are in place)
func (t InvoiceType) CBCKey() cbc.Key {
	return cbc.Key(t)
}

type InvoiceStatus string

const (
	// InvoiceStatusGathering is the status of an invoice that is gathering the items to be invoiced.
	InvoiceStatusGathering InvoiceStatus = "gathering"
	// InvoiceStatusPendingCreation is the status of an invoice summarizing the pending items.
	InvoiceStatusPendingCreation InvoiceStatus = "pending_creation"
	// InvoiceStatusCreated is the status of an invoice that has been created.
	InvoiceStatusCreated InvoiceStatus = "created"
	// InvoiceStatusValidationFailed is the status of an invoice that failed validation.
	InvoiceStatusValidationFailed InvoiceStatus = "validation_failed"
	// InvoiceStatusDraft is the status of an invoice that is in draft both on OpenMeter and the provider side.
	InvoiceStatusDraft InvoiceStatus = "draft"
	// InvoiceStatusDraftSync is the status of an invoice that is being synced with the provider.
	InvoiceStatusDraftSync InvoiceStatus = "draft_sync"
	// InvoiceStatusDraftSyncFailed is the status of an invoice that failed to sync with the provider.
	InvoiceStatusDraftSyncFailed InvoiceStatus = "draft_sync_failed"
	// InvoiceStatusIssuing is the status of an invoice that is being issued.
	InvoiceStatusIssuing InvoiceStatus = "issuing"
	// InvoiceStatusIssued is the status of an invoice that has been issued both on OpenMeter and provider side.
	InvoiceStatusIssued InvoiceStatus = "issued"
	// InvoiceStatusIssuingFailed is the status of an invoice that failed to issue on the provider or OpenMeter side.
	InvoiceStatusIssuingFailed InvoiceStatus = "issuing_failed"
	// InvoiceStatusManualApprovalNeeded is the status of an invoice that needs manual approval. (due to AutoApprove is disabled)
	InvoiceStatusManualApprovalNeeded InvoiceStatus = "manual_approval_needed"
	// InvoiceStatusDeleted is the status of an invoice that has been deleted (e.g. removed from the database before being issued).
	InvoiceStatusDeleted InvoiceStatus = "deleted"
)

// InvoiceImmutableStatuses are the statuses that forbid any changes to the invoice.
var InvoiceImmutableStatuses = []InvoiceStatus{
	InvoiceStatusIssued,
	InvoiceStatusDeleted,
}

func (s InvoiceStatus) Values() []string {
	return lo.Map(
		[]InvoiceStatus{
			InvoiceStatusGathering,
			InvoiceStatusCreated,
			InvoiceStatusDraft,
			InvoiceStatusDraftSync,
			InvoiceStatusDraftSyncFailed,
			InvoiceStatusIssuing,
			InvoiceStatusIssued,
			InvoiceStatusIssuingFailed,
			InvoiceStatusManualApprovalNeeded,
		},
		func(item InvoiceStatus, _ int) string {
			return string(item)
		},
	)
}

func (s InvoiceStatus) Validate() error {
	for _, status := range s.Values() {
		if string(s) == status {
			return nil
		}
	}

	return fmt.Errorf("invalid invoice status: %s", s)
}

func (s InvoiceStatus) IsMutable() bool {
	for _, status := range InvoiceImmutableStatuses {
		if s == status {
			return false
		}
	}

	return true
}

type InvoiceID models.NamespacedID

func (i InvoiceID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type Invoice struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	Number      *string `json:"number,omitempty"`
	Description *string `json:"description,omitempty"`

	Type InvoiceType `json:"type"`

	Metadata map[string]string `json:"metadata"`

	Currency currencyx.Code    `json:"currency,omitempty"`
	Timezone timezone.Timezone `json:"timezone,omitempty"`
	Status   InvoiceStatus     `json:"status"`

	Period *Period `json:"period,omitempty"`

	DueAt *time.Time `json:"dueDate,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	VoidedAt  *time.Time `json:"voidedAt,omitempty"`
	IssuedAt  *time.Time `json:"issuedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// Customer is either a snapshot of the contact information of the customer at the time of invoice being sent
	// or the data from the customer entity (draft state)
	// This is required so that we are not modifying the invoice after it has been sent to the customer.
	Customer InvoiceCustomer  `json:"customer"`
	Supplier SupplierContact  `json:"supplier"`
	Workflow *InvoiceWorkflow `json:"workflow,omitempty"`

	// Line items
	Lines []Line `json:"lines,omitempty"`
}

type InvoiceWithValidation struct {
	Invoice          *Invoice
	ValidationErrors []error
}

type InvoiceCustomer struct {
	CustomerID string `json:"customerId,omitempty"`

	Name           string             `json:"name"`
	BillingAddress *models.Address    `json:"billingAddress,omitempty"`
	Timezone       *timezone.Timezone `json:"timezone,omitempty"`
}

func (i *InvoiceCustomer) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type CustomerMetadata struct {
	Name string `json:"name"`
}
