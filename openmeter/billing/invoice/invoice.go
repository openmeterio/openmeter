package invoice

import (
	"fmt"
	"time"

	"cloud.google.com/go/civil"
	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	goblcurrency "github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
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

type InvoiceItem struct {
	ID InvoiceItemID `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Metadata   map[string]string `json:"metadata"`
	Invoice    *InvoiceID        `json:"invoice,omitempty"`
	CustomerID string            `json:"customer"`

	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`

	InvoiceAt time.Time `json:"invoiceAt"`

	Quantity  alpacadecimal.Decimal `json:"quantity"`
	UnitPrice alpacadecimal.Decimal `json:"unitPrice"`
	Currency  currencyx.Code        `json:"currency"`

	TaxCodeOverride TaxOverrides `json:"taxCodeOverride"`
}

type Invoice struct {
	ID   InvoiceID   `json:"id"`
	Key  string      `json:"key"`
	Type InvoiceType `json:"type"`

	BillingProfileID  string                 `json:"billingProfile"`
	WorkflowConfig    *WorkflowConfig        `json:"workflowConfig"`
	ProviderConfig    provider.Configuration `json:"providerConfig"`
	ProviderReference provider.Reference     `json:"providerReference,omitempty"`

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
	Customer InvoiceCustomer `json:"customerSnapshot"`

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

type InvoiceCustomer struct {
	CustomerID     string          `json:"customerID"`
	Name           string          `json:"name"`
	BillingAddress *models.Address `json:"billingAddress"`
}

// TODO: Name is required at least!

func (i *InvoiceCustomer) ToParty() *org.Party {
	party := &org.Party{
		Name: i.Name,
		Addresses: []*org.Address{
			{
				Country:     l10n.ISOCountryCode(lo.FromPtrOr(i.BillingAddress.Country, "")),
				Street:      lo.FromPtrOr(i.BillingAddress.Line1, ""),
				StreetExtra: lo.FromPtrOr(i.BillingAddress.Line2, ""),
				Region:      lo.FromPtrOr(i.BillingAddress.State, ""),
				Locality:    lo.FromPtrOr(i.BillingAddress.City, ""),

				Code: lo.FromPtrOr(i.BillingAddress.PostalCode, ""),
			},
		},
	}

	if i.BillingAddress.PhoneNumber != nil {
		party.Telephones = append(party.Telephones, &org.Telephone{
			Number: *i.BillingAddress.PhoneNumber,
		})
	}
	return party
}

type CustomerMetadata struct {
	Name string `json:"name"`
}

type GOBLMetadata struct {
	Supplier InvoiceSupplier `json:"supplier"`
}

func (m GOBLMetadata) Validate() error {
	if err := m.Supplier.Validate(); err != nil {
		return fmt.Errorf("error validating supplier: %w", err)
	}

	return nil
}

func (i *Invoice) ToGOBL(meta GOBLMetadata) (*bill.Invoice, error) {
	if i.WorkflowConfig == nil {
		return nil, fmt.Errorf("workflow config is required to generate GOBL invoice")
	}

	loc, err := i.Timezone.LoadLocation()
	if err != nil {
		return nil, fmt.Errorf("error loading timezone location[%s]: %w", i.Timezone, err)
	}

	invoice := &bill.Invoice{
		Type:   i.Type.CBCKey(),
		Series: "", // TODO,
		Code:   "", // TODO,
		IssueDate: cal.Date{
			Date: civil.DateOf(lo.FromPtrOr(i.IssuedAt, i.CreatedAt).In(loc)),
		},
		Currency: goblcurrency.Code(i.Currency),
		Supplier: &org.Party{
			Name: meta.Supplier.Name,
			TaxID: &tax.Identity{
				Country: l10n.TaxCountryCode(meta.Supplier.TaxCountryCode),
			},
		},
		Customer: i.Customer.ToParty(),
		Meta:     convert.MetadataToGOBLMeta(i.Metadata),
	}

	switch i.WorkflowConfig.Invoicing.CollectionMethod {
	case billing.CollectionMethodChargeAutomatically:
		invoice.Payment = &bill.Payment{
			Terms: &pay.Terms{
				Key: pay.TermKeyInstant,
			},
		}

	case billing.CollectionMethodSendInvoice:
		invoice.Payment = &bill.Payment{
			Terms: &pay.Terms{
				Key: pay.TermKeyDueDate,
				DueDates: []*pay.DueDate{
					{
						Date: &cal.Date{
							Date: civil.DateOf(i.DueDate.In(loc)),
						},
						Amount: num.AmountZero, // TODO
					},
				},
			},
		}
	}

	// TODO: line items => let's add the period etc. as metadata field
	// Series will most probably end up in Complements

	return invoice, nil
}
