package billingentity

import (
	"fmt"
	"strings"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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

type InvoiceStatus string

const (
	// InvoiceStatusGathering is the status of an invoice that is gathering the items to be invoiced.
	InvoiceStatusGathering InvoiceStatus = "gathering"

	InvoiceStatusDraftCreated              InvoiceStatus = "draft_created"
	InvoiceStatusDraftUpdating             InvoiceStatus = "draft_updating"
	InvoiceStatusDraftManualApprovalNeeded InvoiceStatus = "draft_manual_approval_needed"
	InvoiceStatusDraftValidating           InvoiceStatus = "draft_validating"
	InvoiceStatusDraftInvalid              InvoiceStatus = "draft_invalid"
	InvoiceStatusDraftSyncing              InvoiceStatus = "draft_syncing"
	InvoiceStatusDraftSyncFailed           InvoiceStatus = "draft_sync_failed"
	InvoiceStatusDraftWaitingAutoApproval  InvoiceStatus = "draft_waiting_auto_approval"
	InvoiceStatusDraftReadyToIssue         InvoiceStatus = "draft_ready_to_issue"

	InvoiceStatusDeleteInProgress InvoiceStatus = "delete_in_progress"
	InvoiceStatusDeleteSyncing    InvoiceStatus = "delete_syncing"
	InvoiceStatusDeleteFailed     InvoiceStatus = "delete_failed"
	InvoiceStatusDeleted          InvoiceStatus = "deleted"

	InvoiceStatusIssuing           InvoiceStatus = "issuing_syncing"
	InvoiceStatusIssuingSyncFailed InvoiceStatus = "issuing_sync_failed"

	// InvoiceStatusIssued is the status of an invoice that has been issued.
	InvoiceStatusIssued InvoiceStatus = "issued"
)

var validStatuses = []InvoiceStatus{
	InvoiceStatusGathering,
	InvoiceStatusDraftCreated,
	InvoiceStatusDraftUpdating,
	InvoiceStatusDraftManualApprovalNeeded,
	InvoiceStatusDraftValidating,
	InvoiceStatusDraftInvalid,
	InvoiceStatusDraftSyncing,
	InvoiceStatusDraftSyncFailed,
	InvoiceStatusDraftWaitingAutoApproval,
	InvoiceStatusDraftReadyToIssue,

	InvoiceStatusDeleteInProgress,
	InvoiceStatusDeleteSyncing,
	InvoiceStatusDeleteFailed,
	InvoiceStatusDeleted,

	InvoiceStatusIssuing,
	InvoiceStatusIssuingSyncFailed,
	InvoiceStatusIssued,
}

func (s InvoiceStatus) Values() []string {
	return lo.Map(
		validStatuses,
		func(item InvoiceStatus, _ int) string {
			return string(item)
		},
	)
}

func (s InvoiceStatus) ShortStatus() string {
	parts := strings.SplitN(string(s), "_", 2)
	return parts[0]
}

var failedStatuses = []InvoiceStatus{
	InvoiceStatusDraftSyncFailed,
	InvoiceStatusIssuingSyncFailed,
	InvoiceStatusDeleteFailed,
}

func (s InvoiceStatus) IsFailed() bool {
	return lo.Contains(failedStatuses, s)
}

func (s InvoiceStatus) Validate() error {
	if !lo.Contains(validStatuses, s) {
		return fmt.Errorf("invalid invoice status: %s", s)
	}

	return nil
}

type InvoiceID models.NamespacedID

func (i InvoiceID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type InvoiceExpand struct {
	Preceding    bool
	WorkflowApps bool
	Lines        bool
	DeletedLines bool
}

var InvoiceExpandAll = InvoiceExpand{
	Preceding:    true,
	WorkflowApps: true,
	Lines:        true,
	DeletedLines: false,
}

func (e InvoiceExpand) Validate() error {
	return nil
}

func (e InvoiceExpand) SetLines(v bool) InvoiceExpand {
	e.Lines = v
	return e
}

func (e InvoiceExpand) SetDeletedLines(v bool) InvoiceExpand {
	e.DeletedLines = v
	return e
}

type InvoiceBase struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	Number      *string `json:"number,omitempty"`
	Description *string `json:"description,omitempty"`

	Type InvoiceType `json:"type"`

	Metadata map[string]string `json:"metadata"`

	Currency      currencyx.Code       `json:"currency,omitempty"`
	Timezone      timezone.Timezone    `json:"timezone,omitempty"`
	Status        InvoiceStatus        `json:"status"`
	StatusDetails InvoiceStatusDetails `json:"statusDetail,omitempty"`

	Period *Period `json:"period,omitempty"`

	DueAt *time.Time `json:"dueDate,omitempty"`

	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	VoidedAt   *time.Time `json:"voidedAt,omitempty"`
	DraftUntil *time.Time `json:"draftUntil,omitempty"`
	IssuedAt   *time.Time `json:"issuedAt,omitempty"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`

	// Customer is either a snapshot of the contact information of the customer at the time of invoice being sent
	// or the data from the customer entity (draft state)
	// This is required so that we are not modifying the invoice after it has been sent to the customer.
	Customer InvoiceCustomer  `json:"customer"`
	Supplier SupplierContact  `json:"supplier"`
	Workflow *InvoiceWorkflow `json:"workflow,omitempty"`

	ExternalIDs InvoiceExternalIDs `json:"externalIds,omitempty"`
}

type Invoice struct {
	InvoiceBase `json:",inline"`

	// Line items
	Lines            LineChildren     `json:"lines,omitempty"`
	ValidationIssues ValidationIssues `json:"validationIssues,omitempty"`
	Totals           Totals           `json:"totals"`

	// private fields required by the service
	ExpandedFields InvoiceExpand `json:"-"`
}

func (i Invoice) InvoiceID() InvoiceID {
	return InvoiceID{
		Namespace: i.Namespace,
		ID:        i.ID,
	}
}

func (i *Invoice) MergeValidationIssues(errIn error, reportingComponent ComponentName) error {
	i.ValidationIssues = lo.Filter(i.ValidationIssues, func(issue ValidationIssue, _ int) bool {
		return issue.Component != reportingComponent
	})

	// Regardless of the errors we need to add them to the invoice, in case the upstream service
	// decides to save the invoice.
	newIssues, finalErrs := ToValidationIssues(errIn)
	i.ValidationIssues = append(i.ValidationIssues, newIssues...)

	return finalErrs
}

func (i *Invoice) HasCriticalValidationIssues() bool {
	_, found := lo.Find(i.ValidationIssues, func(issue ValidationIssue) bool {
		return issue.Severity == ValidationIssueSeverityCritical
	})

	return found
}

// RemoveMetaForCompare returns a copy of the invoice without the fields that are not relevant for higher level
// tests that compare invoices. What gets removed:
// - Line's DB state
// - Line's dependencies are marked as resolved
// - Parent pointers are removed
func (i Invoice) RemoveMetaForCompare() Invoice {
	invoice := i
	invoice.Lines = i.Lines.Map(func(line *Line) *Line {
		return line.RemoveMetaForCompare()
	})

	return invoice
}

func (i *Invoice) FlattenLinesByID() map[string]*Line {
	out := make(map[string]*Line, len(i.Lines.OrEmpty()))

	for _, line := range i.Lines.OrEmpty() {
		out[line.ID] = line

		for _, child := range line.Children.OrEmpty() {
			out[child.ID] = child
		}
	}

	return out
}

func (i Invoice) Clone() Invoice {
	clone := i

	clone.Lines = i.Lines.Clone()
	clone.ValidationIssues = i.ValidationIssues.Clone()
	clone.Totals = i.Totals

	return clone
}

type InvoiceExternalIDs struct {
	Invoicing string `json:"invoicing,omitempty"`
	Payment   string `json:"payment,omitempty"`
}

type InvoiceAction string

const (
	InvoiceActionAdvance InvoiceAction = "advance"
	InvoiceActionApprove InvoiceAction = "approve"
	InvoiceActionDelete  InvoiceAction = "delete"
	InvoiceActionRetry   InvoiceAction = "retry"
	InvoiceActionVoid    InvoiceAction = "void"
)

type InvoiceStatusDetails struct {
	Immutable        bool            `json:"immutable"`
	Failed           bool            `json:"failed"`
	AvailableActions []InvoiceAction `json:"availableActions"`
}

const (
	CustomerUsageAttributionTypeVersion = "customer_usage_attribution.v1"
)

type (
	CustomerUsageAttribution          = customerentity.CustomerUsageAttribution
	VersionedCustomerUsageAttribution struct {
		CustomerUsageAttribution `json:",inline"`
		Type                     string `json:"type"`
	}
)

type InvoiceCustomer struct {
	CustomerID string `json:"customerId,omitempty"`

	Name             string                   `json:"name"`
	BillingAddress   *models.Address          `json:"billingAddress,omitempty"`
	Timezone         *timezone.Timezone       `json:"timezone,omitempty"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
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
