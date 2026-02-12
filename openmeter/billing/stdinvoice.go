package billing

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type StandardInvoiceStatusCategory string

const (
	StandardInvoiceStatusCategoryGathering         StandardInvoiceStatusCategory = "gathering"
	StandardInvoiceStatusCategoryDraft             StandardInvoiceStatusCategory = "draft"
	StandardInvoiceStatusCategoryDelete            StandardInvoiceStatusCategory = "delete"
	StandardInvoiceStatusCategoryDeleted           StandardInvoiceStatusCategory = "deleted"
	StandardInvoiceStatusCategoryIssuing           StandardInvoiceStatusCategory = "issuing"
	StandardInvoiceStatusCategoryIssued            StandardInvoiceStatusCategory = "issued"
	StandardInvoiceStatusCategoryPaymentProcessing StandardInvoiceStatusCategory = "payment_processing"
	StandardInvoiceStatusCategoryOverdue           StandardInvoiceStatusCategory = "overdue"
	StandardInvoiceStatusCategoryPaid              StandardInvoiceStatusCategory = "paid"
	StandardInvoiceStatusCategoryUncollectible     StandardInvoiceStatusCategory = "uncollectible"
	StandardInvoiceStatusCategoryVoided            StandardInvoiceStatusCategory = "voided"
)

func (s StandardInvoiceStatusCategory) MatchesInvoiceStatus(status StandardInvoiceStatus) bool {
	return status.ShortStatus() == string(s)
}

type StandardInvoiceStatus string

const (
	// StandardInvoiceStatusGathering is the status of an invoice that is gathering the items to be invoiced.
	StandardInvoiceStatusGathering StandardInvoiceStatus = "gathering"

	StandardInvoiceStatusDraftCreated StandardInvoiceStatus = "draft.created"
	// StandardInvoiceStatusDraftWaitingForCollection is the status of an invoice that is waiting for the collection to be possible (e.g. collection period has passed)
	StandardInvoiceStatusDraftWaitingForCollection StandardInvoiceStatus = "draft.waiting_for_collection"
	// StandardInvoiceStatusDraftCollecting is the status of an invoice that is collecting the items to be invoiced.
	StandardInvoiceStatusDraftCollecting           StandardInvoiceStatus = "draft.collecting"
	StandardInvoiceStatusDraftUpdating             StandardInvoiceStatus = "draft.updating"
	StandardInvoiceStatusDraftManualApprovalNeeded StandardInvoiceStatus = "draft.manual_approval_needed"
	StandardInvoiceStatusDraftValidating           StandardInvoiceStatus = "draft.validating"
	StandardInvoiceStatusDraftInvalid              StandardInvoiceStatus = "draft.invalid"
	StandardInvoiceStatusDraftSyncing              StandardInvoiceStatus = "draft.syncing"
	StandardInvoiceStatusDraftSyncFailed           StandardInvoiceStatus = "draft.sync_failed"
	StandardInvoiceStatusDraftWaitingAutoApproval  StandardInvoiceStatus = "draft.waiting_auto_approval"
	StandardInvoiceStatusDraftReadyToIssue         StandardInvoiceStatus = "draft.ready_to_issue"

	StandardInvoiceStatusDeleteInProgress StandardInvoiceStatus = "delete.in_progress"
	StandardInvoiceStatusDeleteSyncing    StandardInvoiceStatus = "delete.syncing"
	StandardInvoiceStatusDeleteFailed     StandardInvoiceStatus = "delete.failed"
	StandardInvoiceStatusDeleted          StandardInvoiceStatus = "deleted"

	StandardInvoiceStatusIssuingSyncing    StandardInvoiceStatus = "issuing.syncing"
	StandardInvoiceStatusIssuingSyncFailed StandardInvoiceStatus = "issuing.failed"

	StandardInvoiceStatusIssued StandardInvoiceStatus = "issued"

	StandardInvoiceStatusPaymentProcessingPending        StandardInvoiceStatus = "payment_processing.pending"
	StandardInvoiceStatusPaymentProcessingFailed         StandardInvoiceStatus = "payment_processing.failed"
	StandardInvoiceStatusPaymentProcessingActionRequired StandardInvoiceStatus = "payment_processing.action_required"

	// These are separate statuses to allow for more gradual filtering on the API without having to understand sub-statuses

	StandardInvoiceStatusOverdue StandardInvoiceStatus = "overdue"

	StandardInvoiceStatusPaid StandardInvoiceStatus = "paid"

	StandardInvoiceStatusUncollectible StandardInvoiceStatus = "uncollectible"

	StandardInvoiceStatusVoided StandardInvoiceStatus = "voided"
)

var validStatuses = []StandardInvoiceStatus{
	StandardInvoiceStatusGathering,
	StandardInvoiceStatusDraftCreated,
	StandardInvoiceStatusDraftWaitingForCollection,
	StandardInvoiceStatusDraftCollecting,
	StandardInvoiceStatusDraftUpdating,
	StandardInvoiceStatusDraftManualApprovalNeeded,
	StandardInvoiceStatusDraftValidating,
	StandardInvoiceStatusDraftInvalid,
	StandardInvoiceStatusDraftSyncing,
	StandardInvoiceStatusDraftSyncFailed,
	StandardInvoiceStatusDraftWaitingAutoApproval,
	StandardInvoiceStatusDraftReadyToIssue,

	StandardInvoiceStatusDeleteInProgress,
	StandardInvoiceStatusDeleteSyncing,
	StandardInvoiceStatusDeleteFailed,
	StandardInvoiceStatusDeleted,

	StandardInvoiceStatusIssuingSyncing,
	StandardInvoiceStatusIssuingSyncFailed,

	StandardInvoiceStatusIssued,

	StandardInvoiceStatusPaymentProcessingPending,
	StandardInvoiceStatusPaymentProcessingFailed,
	StandardInvoiceStatusPaymentProcessingActionRequired,

	StandardInvoiceStatusOverdue,

	StandardInvoiceStatusPaid,

	StandardInvoiceStatusUncollectible,

	StandardInvoiceStatusVoided,
}

func (s StandardInvoiceStatus) Values() []string {
	return lo.Map(
		validStatuses,
		func(item StandardInvoiceStatus, _ int) string {
			return string(item)
		},
	)
}

func (s StandardInvoiceStatus) ShortStatus() string {
	parts := strings.SplitN(string(s), ".", 2)
	return parts[0]
}

type StandardInvoiceStatusMatcher interface {
	MatchesInvoiceStatus(StandardInvoiceStatus) bool
}

func (s StandardInvoiceStatus) Matches(statuses ...StandardInvoiceStatusMatcher) bool {
	for _, matcher := range statuses {
		if matcher.MatchesInvoiceStatus(s) {
			return true
		}
	}

	return false
}

func (s StandardInvoiceStatus) MatchesInvoiceStatus(status StandardInvoiceStatus) bool {
	return s == status
}

var failedStatuses = []StandardInvoiceStatus{
	StandardInvoiceStatusDraftSyncFailed,
	StandardInvoiceStatusIssuingSyncFailed,
	StandardInvoiceStatusDeleteFailed,
	StandardInvoiceStatusPaymentProcessingFailed,
}

func (s StandardInvoiceStatus) IsFailed() bool {
	return lo.Contains(failedStatuses, s)
}

var finalStatuses = []StandardInvoiceStatus{
	StandardInvoiceStatusDeleted,
	StandardInvoiceStatusPaid,
	StandardInvoiceStatusUncollectible,
	StandardInvoiceStatusVoided,
}

func (s StandardInvoiceStatus) IsFinal() bool {
	return lo.Contains(finalStatuses, s)
}

func (s StandardInvoiceStatus) Validate() error {
	if !lo.Contains(validStatuses, s) {
		return fmt.Errorf("invalid invoice status: %s", s)
	}

	return nil
}

type StandardInvoiceBase struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	Number      string  `json:"number"`
	Description *string `json:"description,omitempty"`

	Type InvoiceType `json:"type"`

	Metadata map[string]string `json:"metadata"`

	Currency      currencyx.Code               `json:"currency,omitempty"`
	Status        StandardInvoiceStatus        `json:"status"`
	StatusDetails StandardInvoiceStatusDetails `json:"statusDetail,omitempty"`

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

func (i StandardInvoiceBase) Validate() error {
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

func (i StandardInvoiceBase) DefaultCollectionAtForStandardInvoice() time.Time {
	if i.CollectionAt == nil {
		return i.CreatedAt
	}

	return lo.FromPtr(i.CollectionAt)
}

func (i StandardInvoiceBase) GetDeletedAt() *time.Time {
	return i.DeletedAt
}

func (i StandardInvoiceBase) GetID() string {
	return i.ID
}

func (i StandardInvoiceBase) GetInvoiceID() InvoiceID {
	return InvoiceID{
		Namespace: i.Namespace,
		ID:        i.ID,
	}
}

func (i StandardInvoiceBase) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: i.Namespace,
		ID:        i.Customer.CustomerID,
	}
}

var _ GenericInvoice = (*StandardInvoice)(nil)

type StandardInvoice struct {
	StandardInvoiceBase `json:",inline"`

	// Entities external to the invoice itself
	Lines            StandardInvoiceLines `json:"lines,omitempty"`
	ValidationIssues ValidationIssues     `json:"validationIssues,omitempty"`

	Totals Totals `json:"totals"`

	// private fields required by the service
	ExpandedFields InvoiceExpand `json:"-"`
}

func (i StandardInvoice) Validate() error {
	var outErr error

	if err := i.StandardInvoiceBase.Validate(); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if err := i.Lines.Validate(); err != nil {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("lines", err))
	}

	if i.Lines.IsPresent() {
		for _, line := range i.Lines.OrEmpty() {
			if line.Currency != i.Currency {
				outErr = errors.Join(outErr, fmt.Errorf("line[%s]: currency[%s] is not equal to invoice currency[%s]", line.ID, line.Currency, i.Currency))
			}
		}
	}

	return outErr
}

// InvoiceID returns the invoice ID.
// Deprecated: use GetInvoiceID instead
// TODO[later]: remove this method
func (i StandardInvoice) InvoiceID() InvoiceID {
	return i.GetInvoiceID()
}

func (i StandardInvoice) CustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: i.Namespace,
		ID:        i.Customer.CustomerID,
	}
}

func (i StandardInvoice) AsInvoice() Invoice {
	return Invoice{
		t:               InvoiceTypeStandard,
		standardInvoice: &i,
	}
}

func (i StandardInvoice) GetGenericLines() mo.Option[[]GenericInvoiceLine] {
	if !i.Lines.IsPresent() {
		return mo.None[[]GenericInvoiceLine]()
	}

	return mo.Some(lo.Map(i.Lines.OrEmpty(), func(l *StandardLine, _ int) GenericInvoiceLine {
		return &standardInvoiceLineGenericWrapper{StandardLine: l}
	}))
}

func (i *StandardInvoice) SetLines(lines []GenericInvoiceLine) error {
	mappedLines, err := slicesx.MapWithErr(lines, func(l GenericInvoiceLine) (*StandardLine, error) {
		line, err := l.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return nil, err
		}

		return &line, nil
	})
	if err != nil {
		return fmt.Errorf("mapping lines: %w", err)
	}

	i.Lines = NewStandardInvoiceLines(mappedLines)
	return nil
}

func (i *StandardInvoice) MergeValidationIssues(errIn error, reportingComponent ComponentName) error {
	i.ValidationIssues = lo.Filter(i.ValidationIssues, func(issue ValidationIssue, _ int) bool {
		return issue.Component != reportingComponent
	})

	// Regardless of the errors we need to add them to the invoice, in case the upstream service
	// decides to save the invoice.
	newIssues, finalErrs := ToValidationIssues(errIn)
	i.ValidationIssues = append(i.ValidationIssues, newIssues...)

	return finalErrs
}

func (i *StandardInvoice) HasCriticalValidationIssues() bool {
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
func (i StandardInvoice) RemoveMetaForCompare() (StandardInvoice, error) {
	invoice := i
	newLines, err := i.Lines.MapWithErr(func(line *StandardLine) (*StandardLine, error) {
		return line.RemoveMetaForCompare()
	})
	if err != nil {
		return StandardInvoice{}, err
	}

	invoice.Lines = newLines

	return invoice, nil
}

// getLeafLines returns the leaf lines
func (i *StandardInvoice) getLeafLines() DetailedLines {
	out := []DetailedLine{}

	for _, line := range i.Lines.OrEmpty() {
		// Skip non leaf nodes

		out = append(out, line.DetailedLines...)
	}

	return out
}

// GetLeafLinesWithConsolidatedTaxBehavior returns the leaf lines with the tax behavior set to the invoice's tax behavior
// unless the line already has a tax behavior set.
func (i *StandardInvoice) GetLeafLinesWithConsolidatedTaxBehavior() DetailedLines {
	leafLines := i.getLeafLines()
	if i.Workflow.Config.Invoicing.DefaultTaxConfig == nil {
		return leafLines
	}

	return lo.Map(leafLines, func(line DetailedLine, _ int) DetailedLine {
		line.TaxConfig = productcatalog.MergeTaxConfigs(i.Workflow.Config.Invoicing.DefaultTaxConfig, line.TaxConfig)
		return line
	})
}

func (i StandardInvoice) Clone() (StandardInvoice, error) {
	clone := i

	clonedLines, err := i.Lines.Clone()
	if err != nil {
		return StandardInvoice{}, err
	}

	clone.Lines = clonedLines
	clone.ValidationIssues = i.ValidationIssues.Clone()
	clone.Totals = i.Totals

	return clone, nil
}

func (i StandardInvoice) RemoveCircularReferences() (StandardInvoice, error) {
	clone, err := i.Clone()
	if err != nil {
		return StandardInvoice{}, err
	}

	clone.Lines, err = clone.Lines.MapWithErr(func(line *StandardLine) (*StandardLine, error) {
		return line.RemoveCircularReferences()
	})
	if err != nil {
		return StandardInvoice{}, err
	}

	return clone, nil
}

func (i *StandardInvoice) SortLines() {
	if !i.Lines.IsPresent() {
		return
	}

	i.Lines.Sort()
}

type StandardInvoiceLines struct {
	mo.Option[StandardLines]
}

func NewStandardInvoiceLines(children []*StandardLine) StandardInvoiceLines {
	// Note: this helps with test equality checks
	if len(children) == 0 {
		children = nil
	}

	return StandardInvoiceLines{mo.Some(StandardLines(children))}
}

func (i StandardInvoiceLines) Validate() error {
	return errors.Join(lo.Map(i.OrEmpty(), func(line *StandardLine, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), line.Validate())
	})...)
}

func (c StandardInvoiceLines) Map(fn func(*StandardLine) *StandardLine) StandardInvoiceLines {
	if !c.IsPresent() {
		return c
	}

	return StandardInvoiceLines{
		mo.Some(
			c.OrEmpty().Map(fn),
		),
	}
}

func (c StandardInvoiceLines) MapWithErr(fn func(*StandardLine) (*StandardLine, error)) (StandardInvoiceLines, error) {
	if !c.IsPresent() {
		return c, nil
	}

	res, err := slicesx.MapWithErr(c.OrEmpty(), fn)
	if err != nil {
		return StandardInvoiceLines{}, err
	}

	return StandardInvoiceLines{mo.Some(StandardLines(res))}, nil
}

func (c StandardInvoiceLines) WithNormalizedValues() (StandardInvoiceLines, error) {
	return c.MapWithErr(func(line *StandardLine) (*StandardLine, error) {
		return line.WithNormalizedValues()
	})
}

func (c StandardInvoiceLines) Clone() (StandardInvoiceLines, error) {
	return c.MapWithErr(func(l *StandardLine) (*StandardLine, error) {
		return l.Clone()
	})
}

func (c StandardInvoiceLines) GetByID(id string) *StandardLine {
	return lo.FindOrElse(c.Option.OrEmpty(), nil, func(line *StandardLine) bool {
		return line.ID == id
	})
}

func (c *StandardInvoiceLines) ReplaceByID(id string, newLine *StandardLine) bool {
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

func (c *StandardInvoiceLines) Sort() {
	if c.IsAbsent() {
		return
	}

	lines := c.OrEmpty()
	lines.Sort()
	c.Option = mo.Some(lines)
}

// NonDeletedLineCount returns the number of lines that are not deleted and have a valid status (e.g. we are ignoring split lines)
func (c StandardInvoiceLines) NonDeletedLineCount() int {
	return lo.CountBy(c.OrEmpty(), func(l *StandardLine) bool {
		return l.DeletedAt == nil
	})
}

func (c *StandardInvoiceLines) Append(l ...*StandardLine) {
	c.Option = mo.Some(append(c.OrEmpty(), l...))
}

func (c *StandardInvoiceLines) RemoveByID(id string) bool {
	toBeRemoved := c.GetByID(id)
	if toBeRemoved == nil {
		return false
	}

	c.Option = mo.Some(
		lo.Filter(c.Option.OrEmpty(), func(l *StandardLine, _ int) bool {
			return l.ID != id
		}),
	)

	return true
}

func (c StandardInvoiceLines) GetReferencedFeatureKeys() ([]string, error) {
	if c.IsAbsent() {
		return nil, nil
	}

	return c.OrEmpty().GetReferencedFeatureKeys()
}

type StandardInvoiceAvailableActions struct {
	Advance            *StandardInvoiceAvailableActionDetails `json:"advance,omitempty"`
	Approve            *StandardInvoiceAvailableActionDetails `json:"approve,omitempty"`
	Delete             *StandardInvoiceAvailableActionDetails `json:"delete,omitempty"`
	Retry              *StandardInvoiceAvailableActionDetails `json:"retry,omitempty"`
	Void               *StandardInvoiceAvailableActionDetails `json:"void,omitempty"`
	SnapshotQuantities *StandardInvoiceAvailableActionDetails `json:"snapshotQuantities,omitempty"`

	Invoice *StandardInvoiceAvailableActionInvoiceDetails `json:"invoice,omitempty"`
}

type StandardInvoiceAvailableActionDetails struct {
	ResultingState StandardInvoiceStatus `json:"resultingState"`
}

type StandardInvoiceAvailableActionInvoiceDetails struct{}

type StandardInvoiceStatusDetails struct {
	Immutable        bool                            `json:"immutable"`
	Failed           bool                            `json:"failed"`
	AvailableActions StandardInvoiceAvailableActions `json:"availableActions"`
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

type CreateInvoiceAdapterInput struct {
	Namespace string
	Customer  customer.Customer
	Profile   Profile
	Number    string
	Currency  currencyx.Code
	Status    StandardInvoiceStatus
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

	if c.CollectionAt != nil && c.Status != StandardInvoiceStatusGathering {
		return errors.New("setting collectionAt is only allowed when creating gathering invoices")
	}

	return nil
}

type CreateInvoiceAdapterRespone = StandardInvoice

type AssociatedLineCountsAdapterResponse struct {
	Counts map[InvoiceID]int64
}

type (
	AdvanceInvoiceInput     = InvoiceID
	ApproveInvoiceInput     = InvoiceID
	RetryInvoiceInput       = InvoiceID
	SnapshotQuantitiesInput = InvoiceID
)

type UpdateInvoiceAdapterInput = StandardInvoice

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
	Lines      []*StandardLine
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

type UpdateStandardInvoiceInput struct {
	Invoice InvoiceID
	EditFn  func(*StandardInvoice) error
	// IncludeDeletedLines signals the update to populate the deleted lines into the lines field, for the edit function
	IncludeDeletedLines bool
}

func (i UpdateStandardInvoiceInput) Validate() error {
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
	Lines    StandardInvoiceLines
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
	Operation StandardInvoiceOperation
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

type ListStandardInvoicesInput struct {
	pagination.Page

	Namespaces []string
	IDs        []string
	// DraftUtil allows to filter invoices which have their draft state expired based on the provided time.
	// Invoice is expired if the time defined by Invoice.DraftUntil is in the past compared to ListInvoicesInput.DraftUntil.
	DraftUntil *time.Time

	// ExtendedStatuses searches by exact InvoiceStatus
	ExtendedStatuses   []StandardInvoiceStatus
	HasAvailableAction []InvoiceAvailableActionsFilter
	ExternalIDs        *ListInvoicesExternalIDFilter

	/// DELETE everything below this line

	Customers []string
	// Statuses searches by short InvoiceStatus (e.g. draft, issued)
	Statuses []string

	Currencies []currencyx.Code

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	PeriodStartAfter  *time.Time
	PeriodStartBefore *time.Time

	// Filter by invoice creation time
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	IncludeDeleted bool

	// CollectionAt allows to filter invoices which have their collection_at attribute is in the past compared
	// to the time provided in CollectionAt parameter.
	CollectionAt *time.Time

	Expand InvoiceExpand

	OrderBy api.InvoiceOrderBy
	Order   sortx.Order
}

func (i ListStandardInvoicesInput) Validate() error {
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

type ListStandardInvoicesResponse = pagination.Result[StandardInvoice]

type GetStandardInvoiceByIdInput struct {
	Invoice InvoiceID
	Expand  InvoiceExpand
}

func (i GetStandardInvoiceByIdInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if err := i.Expand.Validate(); err != nil {
		return fmt.Errorf("expand: %w", err)
	}

	return nil
}
