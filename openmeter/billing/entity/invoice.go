package billingentity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/invopop/gobl/bill"
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

type InvoiceStatus string

const (
	// InvoiceStatusGathering is the status of an invoice that is gathering the items to be invoiced.
	InvoiceStatusGathering InvoiceStatus = "gathering"

	InvoiceStatusDraftCreated              InvoiceStatus = "draft_created"
	InvoiceStatusDraftManualApprovalNeeded InvoiceStatus = "draft_manual_approval_needed"
	InvoiceStatusDraftValidating           InvoiceStatus = "draft_validating"
	InvoiceStatusDraftInvalid              InvoiceStatus = "draft_invalid"
	InvoiceStatusDraftSyncing              InvoiceStatus = "draft_syncing"
	InvoiceStatusDraftSyncFailed           InvoiceStatus = "draft_sync_failed"
	InvoiceStatusDraftWaitingAutoApproval  InvoiceStatus = "draft_waiting_auto_approval"
	InvoiceStatusDraftReadyToIssue         InvoiceStatus = "draft_ready_to_issue"

	InvoiceStatusIssuing           InvoiceStatus = "issuing_syncing"
	InvoiceStatusIssuingSyncFailed InvoiceStatus = "issuing_sync_failed"

	// InvoiceStatusIssued is the status of an invoice that has been issued.
	InvoiceStatusIssued InvoiceStatus = "issued"
)

var validStatuses = []InvoiceStatus{
	InvoiceStatusGathering,
	InvoiceStatusDraftCreated,
	InvoiceStatusDraftManualApprovalNeeded,
	InvoiceStatusDraftValidating,
	InvoiceStatusDraftInvalid,
	InvoiceStatusDraftSyncing,
	InvoiceStatusDraftSyncFailed,
	InvoiceStatusDraftWaitingAutoApproval,
	InvoiceStatusDraftReadyToIssue,
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

var immutableStatuses = []InvoiceStatus{InvoiceStatusIssued}

func (s InvoiceStatus) IsMutable() bool {
	return !lo.Contains(immutableStatuses, s)
}

var failedStatuses = []InvoiceStatus{
	InvoiceStatusDraftSyncFailed,
	InvoiceStatusIssuingSyncFailed,
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
	Lines        bool
	Preceding    bool
	Workflow     bool
	WorkflowApps bool
}

var InvoiceExpandAll = InvoiceExpand{
	Lines:        true,
	Preceding:    true,
	Workflow:     true,
	WorkflowApps: true,
}

func (e InvoiceExpand) Validate() error {
	if !e.Workflow && e.WorkflowApps {
		return errors.New("workflow.apps can only be expanded when workflow is expanded")
	}

	return nil
}

type Invoice struct {
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

	// Line items
	Lines []Line `json:"lines,omitempty"`

	// private fields required by the service
	Changed        bool          `json:"-"`
	ExpandedFields InvoiceExpand `json:"-"`
}

func (i *Invoice) Calculate() error {
	for _, calc := range InvoiceCalculations {
		changed, err := calc(i)
		if err != nil {
			return err
		}

		if changed {
			i.Changed = true
		}
	}

	return nil
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
