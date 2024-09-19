package invoice

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/currency"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

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

type (
	InvoiceID     models.NamespacedID
	InvoiceItemID models.NamespacedID
)

// TODO: move this to the customer package when it's ready
type CustomerID models.NamespacedID

type InvoiceItem struct {
	ID InvoiceItemID `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Metadata map[string]string `json:"metadata"`
	Invoice  InvoiceID         `json:"invoice,omitempty"`
	Customer CustomerID        `json:"customer"`

	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`

	InvoiceAt time.Time `json:"invoiceAt"`

	Quantity  alpacadecimal.Decimal `json:"quantity"`
	UnitPrice alpacadecimal.Decimal `json:"unitPrice"`
	Currency  currency.Currency     `json:"currency"`

	TaxCodeOverride TaxOverrides `json:"taxCodeOverride"`
}

type Invoice struct {
	Invoice bill.Invoice `json:"invoice"`

	ID   InvoiceID   `json:"id"`
	Key  string      `json:"key"`
	Type InvoiceType `json:"type"`

	Customer CustomerID `json:"customer"`
	// TODO: expand?
	BillingProfileID  string                 `json:"billingProfile"`
	WorkflowConfigID  string                 `json:"workflowConfig"`
	ProviderConfig    provider.Configuration `json:"providerConfig"`
	ProviderReference provider.Reference     `json:"providerReference,omitempty"`

	Metadata map[string]string `json:"metadata"`

	Currency currency.Currency `json:"currency,omitempty"`
	Status   InvoiceStatus     `json:"status"`

	PeriodStart time.Time `json:"periodStart,omitempty"`
	PeriodEnd   time.Time `json:"periodEnd,omitempty"`

	DueDate time.Time `json:"dueDate,omitempty"`

	CreatedAt time.Time  `json:"createdAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt,omitempty"`
	VoidedAt  *time.Time `json:"voidedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// Line items
	Items       []InvoiceItem         `json:"items,omitempty"`
	TotalAmount alpacadecimal.Decimal `json:"totalAmount"`
}

type InvoiceSupplier struct {
	Name           string              `json:"name"`
	TaxCountryCode l10n.TaxCountryCode `json:"taxCode"`
}

func (s InvoiceSupplier) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("supplier name is required")
	}

	// TODO: lookup the country code
	if s.TaxCountryCode == "" {
		return fmt.Errorf("supplier tax country code is required")
	}

	return nil
}

type CustomerMetadata struct {
	Name string `json:"name"`
}

type GOBLMetadata struct {
	Supplier InvoiceSupplier `json:"supplier"`
	// TODO: Customer should be coming from the customer entity
	Customer CustomerMetadata `json:"customer"`
}

func (m GOBLMetadata) Validate() error {
	if err := m.Supplier.Validate(); err != nil {
		return fmt.Errorf("error validating supplier: %w", err)
	}
}

func (i *Invoice) ToGOBL(meta GOBLMetadata) (bill.Invoice, error) {
	invoice := bill.Invoice{
		// TODO: does this worth it or should we just validate ourselfs?
		Type:      i.Type.CBCKey(),
		Series:    "",                  // TODO,
		Code:      "",                  // TODO,
		IssueDate: dbInvoice.CreatedAt, // TODO?!
		Currency:  "",                  // TODO,
		Suplier: &org.Party{
			Name: meta.Supplier.Name, // TODO,
			TaxID: &tax.Identity{
				Country: l10n.TaxCountryCode(meta.Supplier.TaxCountryCode),
			},
		},
		Customer: &org.Party{
			Name: "", // TODO[when customer is ready]
		},
		Payment: &bill.Payment{
			Key: pay.TermKeyDueDate,
			DueDates: []*pay.DueDate{
				{
					Date:   nil, // TODO
					Amount: nil, // TODO
				},
			},
		},
		Meta: dbInvoice.Metadata,
	}

	// TODO: line items => let's add the period etc. as metadata field
	// Series will most probably end up in Complements

	return invoice
}
