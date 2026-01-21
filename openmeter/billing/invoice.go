package billing

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
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

type InvoiceStatusCategory string

const (
	InvoiceStatusCategoryGathering         InvoiceStatusCategory = "gathering"
	InvoiceStatusCategoryDraft             InvoiceStatusCategory = "draft"
	InvoiceStatusCategoryDelete            InvoiceStatusCategory = "delete"
	InvoiceStatusCategoryDeleted           InvoiceStatusCategory = "deleted"
	InvoiceStatusCategoryIssuing           InvoiceStatusCategory = "issuing"
	InvoiceStatusCategoryIssued            InvoiceStatusCategory = "issued"
	InvoiceStatusCategoryPaymentProcessing InvoiceStatusCategory = "payment_processing"
	InvoiceStatusCategoryOverdue           InvoiceStatusCategory = "overdue"
	InvoiceStatusCategoryPaid              InvoiceStatusCategory = "paid"
	InvoiceStatusCategoryUncollectible     InvoiceStatusCategory = "uncollectible"
	InvoiceStatusCategoryVoided            InvoiceStatusCategory = "voided"
)

func (s InvoiceStatusCategory) MatchesInvoiceStatus(status InvoiceStatus) bool {
	return status.ShortStatus() == string(s)
}

type InvoiceStatus string

const (
	// InvoiceStatusGathering is the status of an invoice that is gathering the items to be invoiced.
	InvoiceStatusGathering InvoiceStatus = "gathering"

	InvoiceStatusDraftCreated InvoiceStatus = "draft.created"
	// InvoiceStatusDraftWaitingForCollection is the status of an invoice that is waiting for the collection to be possible (e.g. collection period has passed)
	InvoiceStatusDraftWaitingForCollection InvoiceStatus = "draft.waiting_for_collection"
	// InvoiceStatusDraftCollecting is the status of an invoice that is collecting the items to be invoiced.
	InvoiceStatusDraftCollecting           InvoiceStatus = "draft.collecting"
	InvoiceStatusDraftUpdating             InvoiceStatus = "draft.updating"
	InvoiceStatusDraftManualApprovalNeeded InvoiceStatus = "draft.manual_approval_needed"
	InvoiceStatusDraftValidating           InvoiceStatus = "draft.validating"
	InvoiceStatusDraftInvalid              InvoiceStatus = "draft.invalid"
	InvoiceStatusDraftSyncing              InvoiceStatus = "draft.syncing"
	InvoiceStatusDraftSyncFailed           InvoiceStatus = "draft.sync_failed"
	InvoiceStatusDraftWaitingAutoApproval  InvoiceStatus = "draft.waiting_auto_approval"
	InvoiceStatusDraftReadyToIssue         InvoiceStatus = "draft.ready_to_issue"

	InvoiceStatusDeleteInProgress InvoiceStatus = "delete.in_progress"
	InvoiceStatusDeleteSyncing    InvoiceStatus = "delete.syncing"
	InvoiceStatusDeleteFailed     InvoiceStatus = "delete.failed"
	InvoiceStatusDeleted          InvoiceStatus = "deleted"

	InvoiceStatusIssuingSyncing    InvoiceStatus = "issuing.syncing"
	InvoiceStatusIssuingSyncFailed InvoiceStatus = "issuing.failed"

	InvoiceStatusIssued InvoiceStatus = "issued"

	InvoiceStatusPaymentProcessingPending        InvoiceStatus = "payment_processing.pending"
	InvoiceStatusPaymentProcessingFailed         InvoiceStatus = "payment_processing.failed"
	InvoiceStatusPaymentProcessingActionRequired InvoiceStatus = "payment_processing.action_required"

	// These are separate statuses to allow for more gradual filtering on the API without having to understand sub-statuses

	InvoiceStatusOverdue InvoiceStatus = "overdue"

	InvoiceStatusPaid InvoiceStatus = "paid"

	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"

	InvoiceStatusVoided InvoiceStatus = "voided"
)

var validStatuses = []InvoiceStatus{
	InvoiceStatusGathering,
	InvoiceStatusDraftCreated,
	InvoiceStatusDraftWaitingForCollection,
	InvoiceStatusDraftCollecting,
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

	InvoiceStatusIssuingSyncing,
	InvoiceStatusIssuingSyncFailed,

	InvoiceStatusIssued,

	InvoiceStatusPaymentProcessingPending,
	InvoiceStatusPaymentProcessingFailed,
	InvoiceStatusPaymentProcessingActionRequired,

	InvoiceStatusOverdue,

	InvoiceStatusPaid,

	InvoiceStatusUncollectible,

	InvoiceStatusVoided,
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
	parts := strings.SplitN(string(s), ".", 2)
	return parts[0]
}

type InvoiceStatusMatcher interface {
	MatchesInvoiceStatus(InvoiceStatus) bool
}

func (s InvoiceStatus) Matches(statuses ...InvoiceStatusMatcher) bool {
	for _, matcher := range statuses {
		if matcher.MatchesInvoiceStatus(s) {
			return true
		}
	}

	return false
}

func (s InvoiceStatus) MatchesInvoiceStatus(status InvoiceStatus) bool {
	return s == status
}

var failedStatuses = []InvoiceStatus{
	InvoiceStatusDraftSyncFailed,
	InvoiceStatusIssuingSyncFailed,
	InvoiceStatusDeleteFailed,
	InvoiceStatusPaymentProcessingFailed,
}

func (s InvoiceStatus) IsFailed() bool {
	return lo.Contains(failedStatuses, s)
}

var finalStatuses = []InvoiceStatus{
	InvoiceStatusDeleted,
	InvoiceStatusPaid,
	InvoiceStatusUncollectible,
	InvoiceStatusVoided,
}

func (s InvoiceStatus) IsFinal() bool {
	return lo.Contains(finalStatuses, s)
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
	Preceding bool

	Lines        bool
	DeletedLines bool

	// RecalculateGatheringInvoice is used to calculate the totals and status details of the invoice when gathering,
	// this is temporary until we implement the full progressive billing stack, including gathering invoice recalculations.
	RecalculateGatheringInvoice bool
}

var InvoiceExpandAll = InvoiceExpand{
	Preceding:    true,
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

func (e InvoiceExpand) SetRecalculateGatheringInvoice(v bool) InvoiceExpand {
	e.RecalculateGatheringInvoice = v
	return e
}

type InvoiceBase struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	Number      string  `json:"number"`
	Description *string `json:"description,omitempty"`

	Type InvoiceType `json:"type"`

	Metadata map[string]string `json:"metadata"`

	Currency      currencyx.Code       `json:"currency,omitempty"`
	Status        InvoiceStatus        `json:"status"`
	StatusDetails InvoiceStatusDetails `json:"statusDetail,omitempty"`

	Period *Period `json:"period,omitempty"`

	DueAt *time.Time `json:"dueDate,omitempty"`

	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	VoidedAt             *time.Time `json:"voidedAt,omitempty"`
	DraftUntil           *time.Time `json:"draftUntil,omitempty"`
	IssuedAt             *time.Time `json:"issuedAt,omitempty"`
	DeletedAt            *time.Time `json:"deletedAt,omitempty"`
	SentToCustomerAt     *time.Time `json:"sentToCustomerAt,omitempty"`
	QuantitySnapshotedAt *time.Time `json:"quantitySnapshotedAt,omitempty"`

	CollectionAt *time.Time `json:"collectionAt,omitempty"`
	// PaymentProcessingEnteredAt stores when the invoice first entered payment processing
	PaymentProcessingEnteredAt *time.Time `json:"paymentProcessingEnteredAt,omitempty"`

	// Customer is either a snapshot of the contact information of the customer at the time of invoice being sent
	// or the data from the customer entity (draft state)
	// This is required so that we are not modifying the invoice after it has been sent to the customer.
	Customer InvoiceCustomer `json:"customer"`
	Supplier SupplierContact `json:"supplier"`
	Workflow InvoiceWorkflow `json:"workflow,omitempty"`

	ExternalIDs InvoiceExternalIDs `json:"externalIds,omitempty"`

	SchemaLevel int `json:"schemaLevel"`

	// TODO[later]: Let's also include the totals here, as that's part of the invoice db table
}

func (i InvoiceBase) Validate() error {
	var outErr error

	if err := i.Type.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("type", err))
	}

	if err := i.Currency.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("currency", err))
	}

	if err := i.Status.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("status", err))
	}

	if err := i.Customer.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("customer", err))
	}

	if err := i.Supplier.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("supplier", err))
	}

	if i.Period != nil {
		if err := i.Period.Validate(); err != nil {
			outErr = errors.Join(outErr, ValidationWithFieldPrefix("period", err))
		}
	}

	return outErr
}

func (i InvoiceBase) DefaultCollectionAtForStandardInvoice() time.Time {
	if i.CollectionAt == nil {
		return i.CreatedAt
	}

	return lo.FromPtr(i.CollectionAt)
}

type Invoice struct {
	InvoiceBase `json:",inline"`

	// Entities external to the invoice itself
	Lines            InvoiceLines     `json:"lines,omitempty"`
	ValidationIssues ValidationIssues `json:"validationIssues,omitempty"`

	Totals Totals `json:"totals"`

	// private fields required by the service
	ExpandedFields InvoiceExpand `json:"-"`
}

func (i Invoice) Validate() error {
	var outErr error

	if err := i.InvoiceBase.Validate(); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if err := i.Lines.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("lines", err))
	}

	return outErr
}

func (i Invoice) InvoiceID() InvoiceID {
	return InvoiceID{
		Namespace: i.Namespace,
		ID:        i.ID,
	}
}

func (i Invoice) CustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: i.Namespace,
		ID:        i.Customer.CustomerID,
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

// getLeafLines returns the leaf lines
func (i *Invoice) getLeafLines() DetailedLines {
	out := []DetailedLine{}

	for _, line := range i.Lines.OrEmpty() {
		// Skip non leaf nodes

		out = append(out, line.DetailedLines...)
	}

	return out
}

// GetLeafLinesWithConsolidatedTaxBehavior returns the leaf lines with the tax behavior set to the invoice's tax behavior
// unless the line already has a tax behavior set.
func (i *Invoice) GetLeafLinesWithConsolidatedTaxBehavior() DetailedLines {
	leafLines := i.getLeafLines()
	if i.Workflow.Config.Invoicing.DefaultTaxConfig == nil {
		return leafLines
	}

	return lo.Map(leafLines, func(line DetailedLine, _ int) DetailedLine {
		line.TaxConfig = productcatalog.MergeTaxConfigs(i.Workflow.Config.Invoicing.DefaultTaxConfig, line.TaxConfig)
		return line
	})
}

func (i Invoice) Clone() Invoice {
	clone := i

	clone.Lines = i.Lines.Clone()
	clone.ValidationIssues = i.ValidationIssues.Clone()
	clone.Totals = i.Totals

	return clone
}

func (i Invoice) RemoveCircularReferences() Invoice {
	clone := i.Clone()

	clone.Lines = clone.Lines.Map(func(line *Line) *Line {
		return line.RemoveCircularReferences()
	})

	return clone
}

func (i *Invoice) SortLines() {
	if !i.Lines.IsPresent() {
		return
	}

	i.Lines.Sort()
}

type InvoiceLines struct {
	mo.Option[Lines]
}

func NewInvoiceLines(children []*Line) InvoiceLines {
	// Note: this helps with test equality checks
	if len(children) == 0 {
		children = nil
	}

	return InvoiceLines{mo.Some(Lines(children))}
}

func (i InvoiceLines) Validate() error {
	return errors.Join(lo.Map(i.OrEmpty(), func(line *Line, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), line.Validate())
	})...)
}

func (c InvoiceLines) Map(fn func(*Line) *Line) InvoiceLines {
	if !c.IsPresent() {
		return c
	}

	return InvoiceLines{
		mo.Some(
			c.OrEmpty().Map(fn),
		),
	}
}

func (c InvoiceLines) Clone() InvoiceLines {
	return c.Map(func(l *Line) *Line {
		return l.Clone()
	})
}

func (c InvoiceLines) GetByID(id string) *Line {
	return lo.FindOrElse(c.Option.OrEmpty(), nil, func(line *Line) bool {
		return line.ID == id
	})
}

func (c *InvoiceLines) ReplaceByID(id string, newLine *Line) bool {
	if c.IsAbsent() {
		return false
	}

	lines := c.OrEmpty()

	for i, line := range lines {
		if line.ID == id {
			// Let's preserve the DB state of the original line (as we are only replacing the current state)
			originalDBState := line.DBState

			lines[i] = newLine
			lines[i].DBState = originalDBState
			return true
		}
	}

	return false
}

func (c *InvoiceLines) Sort() {
	if c.IsAbsent() {
		return
	}

	lines := c.OrEmpty()
	lines.Sort()
	c.Option = mo.Some(lines)
}

// NonDeletedLineCount returns the number of lines that are not deleted and have a valid status (e.g. we are ignoring split lines)
func (c InvoiceLines) NonDeletedLineCount() int {
	return lo.CountBy(c.OrEmpty(), func(l *Line) bool {
		return l.DeletedAt == nil
	})
}

func (c *InvoiceLines) Append(l ...*Line) {
	c.Option = mo.Some(append(c.OrEmpty(), l...))
}

func (c *InvoiceLines) RemoveByID(id string) bool {
	toBeRemoved := c.GetByID(id)
	if toBeRemoved == nil {
		return false
	}

	c.Option = mo.Some(
		lo.Filter(c.Option.OrEmpty(), func(l *Line, _ int) bool {
			return l.ID != id
		}),
	)

	return true
}

type InvoiceExternalIDs struct {
	Invoicing string `json:"invoicing,omitempty"`
	Payment   string `json:"payment,omitempty"`
}

func (i *InvoiceExternalIDs) GetInvoicingOrEmpty() string {
	if i == nil {
		return ""
	}
	return i.Invoicing
}

type InvoiceAvailableActions struct {
	Advance            *InvoiceAvailableActionDetails `json:"advance,omitempty"`
	Approve            *InvoiceAvailableActionDetails `json:"approve,omitempty"`
	Delete             *InvoiceAvailableActionDetails `json:"delete,omitempty"`
	Retry              *InvoiceAvailableActionDetails `json:"retry,omitempty"`
	Void               *InvoiceAvailableActionDetails `json:"void,omitempty"`
	SnapshotQuantities *InvoiceAvailableActionDetails `json:"snapshotQuantities,omitempty"`

	Invoice *InvoiceAvailableActionInvoiceDetails `json:"invoice,omitempty"`
}

type InvoiceAvailableActionDetails struct {
	ResultingState InvoiceStatus `json:"resultingState"`
}

type InvoiceAvailableActionInvoiceDetails struct{}

type InvoiceStatusDetails struct {
	Immutable        bool                    `json:"immutable"`
	Failed           bool                    `json:"failed"`
	AvailableActions InvoiceAvailableActions `json:"availableActions"`
}

const (
	CustomerUsageAttributionTypeVersionV1 = "customer_usage_attribution.v1"
	CustomerUsageAttributionTypeVersionV2 = "customer_usage_attribution.v2"
)

type (
	VersionedCustomerUsageAttribution struct {
		streaming.CustomerUsageAttribution `json:",inline"`
		Type                               string `json:"type"`
	}
)

// InvoiceCustomer implements the streaming.CustomerUsageAttribution interface
// This is used to query the usage of a customer in a meter query
var _ streaming.Customer = &InvoiceCustomer{}

// NewInvoiceCustomer creates a new InvoiceCustomer from a customer.Customer
func NewInvoiceCustomer(cust customer.Customer) InvoiceCustomer {
	ic := InvoiceCustomer{
		Key:            cust.Key,
		CustomerID:     cust.ID,
		Name:           cust.Name,
		BillingAddress: cust.BillingAddress,
	}

	// If the customer has a usage attribution, we add it to the invoice customer
	// We use the validator but this is not an error, we allow non usage based invoices without usage attribution.
	if err := cust.GetUsageAttribution().Validate(); err == nil {
		ic.UsageAttribution = lo.ToPtr(cust.GetUsageAttribution())
	}

	return ic
}

// InvoiceCustomer represents a customer that is used in an invoice
// We use a specific model as we snapshot the customer at the time of invoice creation,
// and we don't want to modify the customer entity after it has been sent to the customer.
type InvoiceCustomer struct {
	Key              *string                             `json:"key,omitempty"`
	CustomerID       string                              `json:"customerId,omitempty"`
	Name             string                              `json:"name"`
	BillingAddress   *models.Address                     `json:"billingAddress,omitempty"`
	UsageAttribution *streaming.CustomerUsageAttribution `json:"usageAttribution,omitempty"`
}

// GetUsageAttribution returns the customer usage attribution
// implementing the streaming.CustomerUsageAttribution interface
func (c InvoiceCustomer) GetUsageAttribution() streaming.CustomerUsageAttribution {
	subjectKeys := []string{}
	if c.UsageAttribution != nil {
		subjectKeys = c.UsageAttribution.SubjectKeys
	}

	return streaming.NewCustomerUsageAttribution(c.CustomerID, c.Key, subjectKeys)
}

// Validate validates the invoice customer
func (i *InvoiceCustomer) Validate() error {
	if i.CustomerID == "" {
		return fmt.Errorf("customerID is required")
	}

	if i.Key != nil && *i.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type CustomerMetadata struct {
	Name string `json:"name"`
}

type GetInvoiceByIdInput struct {
	Invoice InvoiceID
	Expand  InvoiceExpand
}

func (i GetInvoiceByIdInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if err := i.Expand.Validate(); err != nil {
		return fmt.Errorf("expand: %w", err)
	}

	return nil
}

type genericMultiInvoiceInput struct {
	Namespace  string
	InvoiceIDs []string
}

func (i genericMultiInvoiceInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if len(i.InvoiceIDs) == 0 {
		return errors.New("invoice IDs are required")
	}

	return nil
}

type (
	DeleteGatheringInvoicesInput     = genericMultiInvoiceInput
	LockInvoicesForUpdateInput       = genericMultiInvoiceInput
	AssociatedLineCountsAdapterInput = genericMultiInvoiceInput
)

type ExternalIDType string

const (
	InvoicingExternalIDType ExternalIDType = "invoicing"
	PaymentExternalIDType   ExternalIDType = "payment"
	TaxExternalIDType       ExternalIDType = "tax"
)

func (t ExternalIDType) Validate() error {
	if !slices.Contains([]ExternalIDType{
		InvoicingExternalIDType,
		PaymentExternalIDType,
		TaxExternalIDType,
	}, t) {
		return fmt.Errorf("invalid external ID type: %s", t)
	}

	return nil
}

type ListInvoicesExternalIDFilter struct {
	Type ExternalIDType
	IDs  []string
}

func (f ListInvoicesExternalIDFilter) Validate() error {
	if err := f.Type.Validate(); err != nil {
		return err
	}

	if len(f.IDs) == 0 {
		return errors.New("IDs are required")
	}

	return nil
}

type InvoiceAvailableActionsFilter string

const (
	InvoiceAvailableActionsFilterAdvance InvoiceAvailableActionsFilter = "advance"
	InvoiceAvailableActionsFilterApprove InvoiceAvailableActionsFilter = "approve"
)

func (f InvoiceAvailableActionsFilter) Values() []InvoiceAvailableActionsFilter {
	return []InvoiceAvailableActionsFilter{
		InvoiceAvailableActionsFilterAdvance,
		InvoiceAvailableActionsFilterApprove,
	}
}

func (f InvoiceAvailableActionsFilter) Validate() error {
	if !slices.Contains(f.Values(), f) {
		return fmt.Errorf("invalid available action filter: %s", f)
	}

	return nil
}

type ListInvoicesInput struct {
	pagination.Page

	Namespaces []string
	IDs        []string
	Customers  []string
	// Statuses searches by short InvoiceStatus (e.g. draft, issued)
	Statuses []string

	HasAvailableAction []InvoiceAvailableActionsFilter

	// ExtendedStatuses searches by exact InvoiceStatus
	ExtendedStatuses []InvoiceStatus
	Currencies       []currencyx.Code

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	PeriodStartAfter  *time.Time
	PeriodStartBefore *time.Time

	// Filter by invoice creation time
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	IncludeDeleted bool

	// DraftUtil allows to filter invoices which have their draft state expired based on the provided time.
	// Invoice is expired if the time defined by Invoice.DraftUntil is in the past compared to ListInvoicesInput.DraftUntil.
	DraftUntil *time.Time

	// CollectionAt allows to filter invoices which have their collection_at attribute is in the past compared
	// to the time provided in CollectionAt parameter.
	CollectionAt *time.Time

	Expand InvoiceExpand

	ExternalIDs *ListInvoicesExternalIDFilter

	OrderBy api.InvoiceOrderBy
	Order   sortx.Order
}

func (i ListInvoicesInput) Validate() error {
	var outErr []error

	if i.IssuedAfter != nil && i.IssuedBefore != nil && i.IssuedAfter.After(*i.IssuedBefore) {
		outErr = append(outErr, errors.New("issuedAfter must be before issuedBefore"))
	}

	if i.CreatedAfter != nil && i.CreatedBefore != nil && i.CreatedAfter.After(*i.CreatedBefore) {
		outErr = append(outErr, errors.New("createdAfter must be before createdBefore"))
	}

	if i.PeriodStartAfter != nil && i.PeriodStartBefore != nil && i.PeriodStartAfter.After(*i.PeriodStartBefore) {
		outErr = append(outErr, errors.New("periodStartAfter must be before periodStartBefore"))
	}

	if err := i.Expand.Validate(); err != nil {
		outErr = append(outErr, fmt.Errorf("expand: %w", err))
	}

	if i.ExternalIDs != nil {
		if err := i.ExternalIDs.Validate(); err != nil {
			outErr = append(outErr, fmt.Errorf("external IDs: %w", err))
		}
	}

	if len(i.HasAvailableAction) > 0 {
		errs := errors.Join(
			lo.Map(i.HasAvailableAction, func(action InvoiceAvailableActionsFilter, _ int) error {
				return action.Validate()
			})...,
		)
		if errs != nil {
			outErr = append(outErr, errs)
		}
	}

	return errors.Join(outErr...)
}

type ListInvoicesResponse = pagination.Result[Invoice]

type CreateInvoiceAdapterInput struct {
	Namespace string
	Customer  customer.Customer
	Profile   Profile
	Number    string
	Currency  currencyx.Code
	Status    InvoiceStatus
	Metadata  map[string]string
	IssuedAt  time.Time

	Type         InvoiceType
	Description  *string
	DueAt        *time.Time
	CollectionAt *time.Time

	Totals Totals
}

func (c CreateInvoiceAdapterInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := c.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if err := c.Profile.Validate(); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	if c.Profile.Apps == nil {
		return errors.New("profile: apps must be expanded")
	}

	if err := c.Currency.Validate(); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if err := c.Status.Validate(); err != nil {
		return fmt.Errorf("status: %w", err)
	}

	if err := c.Type.Validate(); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	if err := c.Totals.Validate(); err != nil {
		return fmt.Errorf("totals: %w", err)
	}

	if c.Number == "" {
		return errors.New("invoice number is required")
	}

	if c.CollectionAt != nil && c.Status != InvoiceStatusGathering {
		return errors.New("setting collectionAt is only allowed when creating gathering invoices")
	}

	return nil
}

type CreateInvoiceAdapterRespone = Invoice

type InvoicePendingLinesInput struct {
	Customer customer.CustomerID

	IncludePendingLines mo.Option[[]string]
	AsOf                *time.Time

	// ProgressiveBillingOverride allows to override the progressive billing setting of the customer.
	// This is used to make sure that system collection does not use progressive billing.
	ProgressiveBillingOverride *bool
}

func (i InvoicePendingLinesInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.AsOf != nil && i.AsOf.After(clock.Now()) {
		return errors.New("asOf must be in the past")
	}

	if i.IncludePendingLines.IsPresent() {
		if len(i.IncludePendingLines.OrEmpty()) == 0 {
			return errors.New("includePendingLines must contain at least one line ID")
		}
	}

	return nil
}

type AssociatedLineCountsAdapterResponse struct {
	Counts map[InvoiceID]int64
}

type (
	AdvanceInvoiceInput     = InvoiceID
	ApproveInvoiceInput     = InvoiceID
	RetryInvoiceInput       = InvoiceID
	SnapshotQuantitiesInput = InvoiceID
)

type UpdateInvoiceAdapterInput = Invoice

type GetInvoiceOwnershipAdapterInput = InvoiceID

type GetOwnershipAdapterResponse struct {
	Namespace  string
	InvoiceID  string
	CustomerID string
}

type DeleteInvoiceInput = InvoiceID

type UpdateInvoiceLinesInternalInput struct {
	Namespace  string
	CustomerID string
	Lines      []*Line
}

func (i UpdateInvoiceLinesInternalInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.CustomerID == "" {
		return errors.New("customer ID is required")
	}

	return nil
}

type UpdateInvoiceInput struct {
	Invoice InvoiceID
	EditFn  func(*Invoice) error
	// IncludeDeletedLines signals the update to populate the deleted lines into the lines field, for the edit function
	IncludeDeletedLines bool
}

func (i UpdateInvoiceInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if i.EditFn == nil {
		return errors.New("edit function is required")
	}

	return nil
}

type SimulateInvoiceInput struct {
	Namespace  string
	CustomerID *string
	Customer   *customer.Customer

	Number   *string
	Currency currencyx.Code
	Lines    InvoiceLines
}

func (i SimulateInvoiceInput) Validate() error {
	if i.CustomerID != nil {
		if *i.CustomerID == "" {
			return errors.New("customer ID is required")
		}
	}

	if i.Customer != nil {
		if err := i.Customer.Validate(); err != nil {
			return fmt.Errorf("customer: %w", err)
		}
	}

	if i.CustomerID == nil && i.Customer == nil {
		return errors.New("either customer ID or customer is required")
	}

	if i.CustomerID != nil && i.Customer != nil {
		return errors.New("only one of customer ID or customer can be specified")
	}

	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Currency == "" {
		return errors.New("currency is required")
	}

	if len(i.Lines.OrEmpty()) == 0 {
		return errors.New("lines are required")
	}

	return nil
}

type UpsertValidationIssuesInput struct {
	Invoice InvoiceID
	Issues  ValidationIssues
}

func (i UpsertValidationIssuesInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if len(i.Issues) == 0 {
		return errors.New("issues are required")
	}

	return nil
}

type InvoiceTriggerValidationInput struct {
	// Operation specifies the operation that yielded the validation errors
	// previous validation errors from this operation will be replaced by this one
	Operation InvoiceOperation
	Errors    []error
}

func (i InvoiceTriggerValidationInput) Validate() error {
	if err := i.Operation.Validate(); err != nil {
		return fmt.Errorf("operation: %w", err)
	}

	if len(i.Errors) == 0 {
		return errors.New("validation errors are required")
	}

	return nil
}

type InvoiceTriggerInput struct {
	Invoice InvoiceID
	// Trigger specifies the trigger that caused the invoice to be changed, only triggerPaid and triggerPayment* are allowed
	Trigger InvoiceTrigger

	ValidationErrors *InvoiceTriggerValidationInput
}

func (i InvoiceTriggerInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if i.Trigger == "" {
		return errors.New("trigger is required")
	}

	if i.ValidationErrors != nil {
		if err := i.ValidationErrors.Validate(); err != nil {
			return fmt.Errorf("validation errors: %w", err)
		}
	}

	return nil
}

type InvoiceTriggerServiceInput struct {
	InvoiceTriggerInput

	// AppType is the type of the app that triggered the invoice
	AppType app.AppType
	// Capability is the capability of the app that was processing this trigger
	Capability app.CapabilityType
}

func (i InvoiceTriggerServiceInput) Validate() error {
	if err := i.InvoiceTriggerInput.Validate(); err != nil {
		return fmt.Errorf("trigger: %w", err)
	}

	if i.AppType == "" {
		return errors.New("app type is required")
	}

	if i.Capability == "" {
		return errors.New("capability is required")
	}

	return nil
}

type UpdateInvoiceFieldsInput struct {
	Invoice          InvoiceID
	SentToCustomerAt mo.Option[*time.Time]
}

func (i UpdateInvoiceFieldsInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	return nil
}

type RecalculateGatheringInvoicesInput = customer.CustomerID
