package billing

import (
	"fmt"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

type InvoiceType cbc.Key

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

func (t InvoiceType) CBCKey() cbc.Key {
	return cbc.Key(t)
}

type InvoiceStatus string

const (
	// InvoiceStatusPendingCreation is the status of an invoice summarizing the pending items.
	InvoiceStatusPendingCreation InvoiceStatus = "pending_creation"
	// InvoiceStatusCreated is the status of an invoice that has been created.
	InvoiceStatusCreated InvoiceStatus = "created"
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

type InvoiceNumber struct {
	// Number is {SERIES}-{CODE}

	Series string `json:"series"`
	Code   string `json:"code"`
}

type Invoice struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	InvoiceNumber InvoiceNumber `json:"invoiceNumber"`

	Type InvoiceType `json:"type"`

	Metadata map[string]string `json:"metadata"`

	Currency currencyx.Code    `json:"currency,omitempty"`
	Timezone timezone.Timezone `json:"timezone,omitempty"`
	Status   InvoiceStatus     `json:"status"`

	PeriodStart time.Time `json:"periodStart,omitempty"`
	PeriodEnd   time.Time `json:"periodEnd,omitempty"`

	DueDate *time.Time `json:"dueDate,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	VoidedAt  *time.Time `json:"voidedAt,omitempty"`
	IssuedAt  *time.Time `json:"issuedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// Customer is either a snapshot of the contact information of the customer at the time of invoice being sent
	// or the data from the customer entity (draft state)
	// This is required so that we are not modifying the invoice after it has been sent to the customer.
	Profile  Profile         `json:"profile"`
	Customer InvoiceCustomer `json:"customer"`

	// Line items
	Items []InvoiceItem `json:"items,omitempty"`
}

type InvoiceWithValidation struct {
	Invoice          *Invoice
	ValidationErrors []error
}

type InvoiceCustomer customerentity.Customer

func (i *InvoiceCustomer) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type CustomerMetadata struct {
	Name string `json:"name"`
}
