package creditvoid

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	VoidCreditPurchase(ctx context.Context, input VoidCreditPurchaseInput) (VoidCreditPurchaseResult, error)
	ListVoidedCreditImpacts(ctx context.Context, input ListVoidedCreditImpactsInput) (ListVoidedCreditImpactsResult, error)
}

type Config struct {
	Adapter            Adapter
	Ledger             ledger.Ledger
	Dependencies       transactions.ResolverDependencies
	Breakage           breakage.Service
	AccountLocker      ledger.AccountLocker
	TransactionManager transaction.Creator
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}
	if c.Ledger == nil {
		errs = append(errs, errors.New("ledger is required"))
	}
	if c.Dependencies.AccountService == nil {
		errs = append(errs, errors.New("account service is required"))
	}
	if c.Dependencies.AccountCatalog == nil {
		errs = append(errs, errors.New("account catalog is required"))
	}
	if c.Dependencies.BalanceQuerier == nil {
		errs = append(errs, errors.New("balance querier is required"))
	}
	if c.Breakage == nil {
		errs = append(errs, errors.New("breakage service is required"))
	}
	if c.AccountLocker == nil {
		errs = append(errs, errors.New("account locker is required"))
	}
	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
	}

	return errors.Join(errs...)
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:            config.Adapter,
		ledger:             config.Ledger,
		deps:               config.Dependencies,
		breakage:           config.Breakage,
		accountLocker:      config.AccountLocker,
		transactionManager: config.TransactionManager,
	}, nil
}

type service struct {
	adapter            Adapter
	ledger             ledger.Ledger
	deps               transactions.ResolverDependencies
	breakage           breakage.Service
	accountLocker      ledger.AccountLocker
	transactionManager transaction.Creator
}

type VoidCreditPurchaseInput struct {
	CustomerID  customer.CustomerID
	ChargeID    string
	Currency    currencyx.Code
	Annotations models.Annotations
}

func (i VoidCreditPurchaseInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	return errors.Join(errs...)
}

type VoidCreditPurchaseResult struct {
	VoidedAt           time.Time
	Amount             alpacadecimal.Decimal
	TransactionGroupID string
}

type Record struct {
	ID        models.NamespacedID
	Amount    alpacadecimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	CustomerID customer.CustomerID
	Currency   currencyx.Code
	VoidedAt   time.Time

	SourceChargeID string

	VoidTransactionGroupID string
	VoidTransactionID      string

	FBOSubAccountID        string
	ReceivableSubAccountID string

	Annotations models.Annotations
}

type PendingRecord struct {
	Record
}

type CreateRecordsInput struct {
	Records []Record
}

func (i CreateRecordsInput) Validate() error {
	for idx, record := range i.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("records[%d]: %w", idx, err)
		}
	}

	return nil
}

func (r Record) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("id: %w", err))
	}
	if !r.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if err := r.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}
	if err := r.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if r.VoidedAt.IsZero() {
		errs = append(errs, errors.New("voided at is required"))
	}
	if r.SourceChargeID == "" {
		errs = append(errs, errors.New("source charge id is required"))
	}
	if r.VoidTransactionGroupID == "" {
		errs = append(errs, errors.New("void transaction group id is required"))
	}
	if r.VoidTransactionID == "" {
		errs = append(errs, errors.New("void transaction id is required"))
	}
	if r.FBOSubAccountID == "" {
		errs = append(errs, errors.New("FBO sub-account id is required"))
	}
	if r.ReceivableSubAccountID == "" {
		errs = append(errs, errors.New("receivable sub-account id is required"))
	}

	return errors.Join(errs...)
}

type ListRecordsInput struct {
	CustomerID customer.CustomerID
	Currency   *currencyx.Code
	AsOf       time.Time
	Route      ledger.RouteFilter
}

type ListVoidedCreditImpactsInput struct {
	CustomerID customer.CustomerID
	Currency   *currencyx.Code
	AsOf       time.Time
	After      *ledger.TransactionCursor
	Before     *ledger.TransactionCursor
	Limit      int
	Route      ledger.RouteFilter
}

func (i ListVoidedCreditImpactsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}
	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}
	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}
	if i.After != nil {
		if err := i.After.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("after: %w", err))
		}
	}
	if i.Before != nil {
		if err := i.Before.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("before: %w", err))
		}
	}
	if i.After != nil && i.Before != nil {
		errs = append(errs, errors.New("after and before cannot be set together"))
	}
	if err := breakage.ValidateExpiredRouteFilter(i.Route); err != nil {
		errs = append(errs, fmt.Errorf("route: %w", err))
	}
	if i.Limit < 1 {
		errs = append(errs, errors.New("limit must be greater than 0"))
	}

	return errors.Join(errs...)
}

type ListVoidedCreditImpactsResult struct {
	Items   []VoidImpact
	HasMore bool
}

type VoidImpact struct {
	ID          models.NamespacedID
	CreatedAt   time.Time
	VoidedAt    time.Time
	CustomerID  customer.CustomerID
	Currency    currencyx.Code
	Amount      alpacadecimal.Decimal
	Annotations models.Annotations
}

func (i VoidImpact) Cursor() ledger.TransactionCursor {
	return ledger.TransactionCursor{
		BookedAt:  i.VoidedAt,
		CreatedAt: i.CreatedAt,
		ID:        i.ID,
	}
}

type Adapter interface {
	CreateRecords(ctx context.Context, input CreateRecordsInput) error
	ListRecords(ctx context.Context, input ListRecordsInput) ([]Record, error)
}

// voidPlan is the read-only outcome of the planning step: the concrete
// remaining balance slices to forfeit, each with the open expiry breakage plan
// it must release.
type voidPlan struct {
	voidedAt time.Time
	slices   []voidSlice
}

type voidSlice struct {
	amount     alpacadecimal.Decimal
	fboAccount string
	fboAddress ledger.PostingAddress
	// expiryPlan is nil for non-expiring charges.
	expiryPlan *breakage.Plan
}

func (s *service) VoidCreditPurchase(ctx context.Context, input VoidCreditPurchaseInput) (VoidCreditPurchaseResult, error) {
	if err := input.Validate(); err != nil {
		return VoidCreditPurchaseResult{}, err
	}

	return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (VoidCreditPurchaseResult, error) {
		plan, err := s.planVoid(ctx, input)
		if err != nil {
			return VoidCreditPurchaseResult{}, err
		}

		return s.executeVoid(ctx, input, plan)
	})
}

// planVoid locks the customer FBO account, reads the charge's remaining
// balance slices, and matches each to the open expiry breakage plan it must
// release. The lock stays held until the surrounding transaction commits, so
// the plan cannot go stale against concurrent collection or voiding.
func (s *service) planVoid(ctx context.Context, input VoidCreditPurchaseInput) (voidPlan, error) {
	// Truncate to Postgres timestamp precision so the returned void time
	// matches the persisted rows byte-for-byte.
	voidedAt := clock.Now().UTC().Truncate(time.Microsecond)

	customerAccounts, err := s.deps.AccountService.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return voidPlan{}, fmt.Errorf("get customer accounts: %w", err)
	}

	if err := s.accountLocker.LockAccountsForPosting(ctx, []ledger.Account{customerAccounts.FBOAccount}); err != nil {
		return voidPlan{}, fmt.Errorf("lock customer FBO account: %w", err)
	}

	fboAccountID := customerAccounts.FBOAccount.ID()
	// The as-of read naturally excludes the charge's own future-dated expiry
	// breakage entries.
	buckets, err := s.deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: input.CustomerID.Namespace,
		Filters: ledger.Filters{
			AccountID:      &fboAccountID.ID,
			SourceChargeID: mo.Some(&input.ChargeID),
			AsOf:           &voidedAt,
			Route: ledger.RouteFilter{
				Currency: input.Currency,
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	if err != nil {
		return voidPlan{}, fmt.Errorf("get FBO balance buckets: %w", err)
	}

	openPlansBySubAccount, err := s.openExpiryPlansBySubAccount(ctx, input, voidedAt)
	if err != nil {
		return voidPlan{}, err
	}

	plan := voidPlan{voidedAt: voidedAt}
	for _, bucket := range buckets {
		if !bucket.SettledAmount.IsPositive() {
			continue
		}

		slice := voidSlice{
			amount:     bucket.SettledAmount,
			fboAccount: fboAccountID.ID,
			fboAddress: bucket.Address,
		}

		// Expiring grants should have matching open expiry plans; non-expiring
		// grants and already-released plans are absent from this lookup.
		expiryPlan, ok := openPlansBySubAccount[bucket.Address.SubAccountID()]
		if ok {
			if expiryPlan.OpenAmount.LessThan(slice.amount) {
				return voidPlan{}, fmt.Errorf("open expiry breakage plan %s amount %s is less than voided amount %s", expiryPlan.ID.ID, expiryPlan.OpenAmount, slice.amount)
			}

			slice.expiryPlan = &expiryPlan
		}

		plan.slices = append(plan.slices, slice)
	}

	if len(plan.slices) == 0 {
		return voidPlan{}, models.NewGenericConflictError(
			fmt.Errorf("credit purchase %s has no remaining value to void", input.ChargeID),
		)
	}

	return plan, nil
}

func (s *service) openExpiryPlansBySubAccount(ctx context.Context, input VoidCreditPurchaseInput, voidedAt time.Time) (map[string]breakage.Plan, error) {
	openPlans, err := s.breakage.ListPlans(ctx, breakage.ListPlansInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       voidedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("list open breakage plans: %w", err)
	}

	out := make(map[string]breakage.Plan)
	for _, plan := range openPlans {
		if plan.SourceChargeID == nil || *plan.SourceChargeID != input.ChargeID {
			continue
		}

		out[plan.FBOSubAccountID] = plan
	}

	return out, nil
}

func (s *service) executeVoid(ctx context.Context, input VoidCreditPurchaseInput, plan voidPlan) (VoidCreditPurchaseResult, error) {
	var (
		inputs             []ledger.TransactionInput
		pendingBreakage    []breakage.PendingRecord
		pendingVoidRecords []PendingRecord
	)
	amount := alpacadecimal.Zero

	for _, slice := range plan.slices {
		voidTx, voidRecord, err := s.resolveVoidSlice(ctx, input, plan.voidedAt, slice)
		if err != nil {
			return VoidCreditPurchaseResult{}, err
		}

		inputs = append(inputs, voidTx)
		pendingVoidRecords = append(pendingVoidRecords, voidRecord)
		amount = amount.Add(slice.amount)

		if slice.expiryPlan == nil {
			continue
		}

		releaseTx, releaseRecord, err := s.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
			Plan:           *slice.expiryPlan,
			Amount:         slice.amount,
			SourceKind:     breakage.SourceKindCreditPurchase,
			SourceChargeID: &input.ChargeID,
		})
		if err != nil {
			return VoidCreditPurchaseResult{}, fmt.Errorf("resolve expiry breakage release: %w", err)
		}

		inputs = append(inputs, releaseTx)
		pendingBreakage = append(pendingBreakage, releaseRecord)
	}

	for i, txInput := range inputs {
		if txInput != nil {
			inputs[i] = transactions.WithAnnotations(txInput, input.Annotations)
		}
	}

	transactionGroup, err := s.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.CustomerID.Namespace,
		input.Annotations,
		inputs...,
	))
	if err != nil {
		return VoidCreditPurchaseResult{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	if err := s.breakage.PersistCommittedRecords(ctx, pendingBreakage, transactionGroup); err != nil {
		return VoidCreditPurchaseResult{}, fmt.Errorf("persist breakage records: %w", err)
	}

	if err := s.persistCommittedVoidRecords(ctx, pendingVoidRecords, transactionGroup); err != nil {
		return VoidCreditPurchaseResult{}, fmt.Errorf("persist void records: %w", err)
	}

	return VoidCreditPurchaseResult{
		VoidedAt:           plan.voidedAt,
		Amount:             amount,
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

func (s *service) resolveVoidSlice(ctx context.Context, input VoidCreditPurchaseInput, voidedAt time.Time, slice voidSlice) (ledger.TransactionInput, PendingRecord, error) {
	recordID := newRecordID(input.CustomerID.Namespace)
	route := slice.fboAddress.Route().Route()

	issueTx, err := s.originalIssueTransaction(ctx, input, voidedAt, slice)
	if err != nil {
		return nil, PendingRecord{}, err
	}

	inputs, err := transactions.CorrectTransaction(ctx, s.deps, transactions.CorrectionInput{
		At:                  voidedAt,
		Amount:              slice.amount,
		OriginalTransaction: issueTx,
	})
	if err != nil {
		return nil, PendingRecord{}, fmt.Errorf("resolve issue correction: %w", err)
	}
	if len(inputs) != 1 {
		return nil, PendingRecord{}, fmt.Errorf("expected one issue correction transaction input, got %d", len(inputs))
	}

	correctedFBO, correctedReceivable, err := correctionEntrySubAccounts(inputs[0])
	if err != nil {
		return nil, PendingRecord{}, err
	}
	if correctedFBO != slice.fboAddress.SubAccountID() {
		return nil, PendingRecord{}, fmt.Errorf("issue correction FBO sub-account %s does not match voided FBO sub-account %s", correctedFBO, slice.fboAddress.SubAccountID())
	}

	record := PendingRecord{Record: Record{
		ID:                     recordID,
		Amount:                 slice.amount,
		CustomerID:             input.CustomerID,
		Currency:               route.Currency,
		VoidedAt:               voidedAt,
		SourceChargeID:         input.ChargeID,
		FBOSubAccountID:        correctedFBO,
		ReceivableSubAccountID: correctedReceivable,
	}}

	return transactions.WithAnnotations(inputs[0], creditVoidRecordAnnotations(recordID.ID)), record, nil
}

func (s *service) originalIssueTransaction(ctx context.Context, input VoidCreditPurchaseInput, voidedAt time.Time, slice voidSlice) (ledger.Transaction, error) {
	cursor := (*ledger.TransactionCursor)(nil)

	for {
		page, err := s.ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
			Namespace:      input.CustomerID.Namespace,
			Cursor:         cursor,
			Limit:          100,
			AccountIDs:     []string{slice.fboAccount},
			Currency:       &input.Currency,
			AsOf:           &voidedAt,
			CreditMovement: ledger.ListTransactionsCreditMovementPositive,
			AnnotationFilters: map[string]string{
				ledger.AnnotationTransactionTemplateCode: string(transactions.TemplateCodeIssueCustomerReceivable),
				ledger.AnnotationTransactionDirection:    string(ledger.TransactionDirectionForward),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("list issue transactions: %w", err)
		}

		for _, tx := range page.Items {
			if !transactionIssuedFBOForSource(tx, input.ChargeID, slice.fboAddress.SubAccountID()) {
				continue
			}

			fullTx, err := s.transactionByID(ctx, tx.ID())
			if err != nil {
				return nil, err
			}

			return fullTx, nil
		}

		if page.NextCursor == nil {
			return nil, fmt.Errorf("issue transaction for charge %s FBO sub-account %s not found", input.ChargeID, slice.fboAddress.SubAccountID())
		}

		cursor = page.NextCursor
	}
}

func (s *service) transactionByID(ctx context.Context, id models.NamespacedID) (ledger.Transaction, error) {
	page, err := s.ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace:     id.Namespace,
		Limit:         1,
		TransactionID: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get transaction %s: %w", id.ID, err)
	}
	if len(page.Items) != 1 {
		return nil, fmt.Errorf("transaction %s not found", id.ID)
	}

	return page.Items[0], nil
}

func transactionIssuedFBOForSource(tx ledger.Transaction, sourceChargeID string, fboSubAccountID string) bool {
	for _, entry := range tx.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO {
			continue
		}
		if entry.PostingAddress().SubAccountID() != fboSubAccountID {
			continue
		}
		if !entry.Amount().IsPositive() {
			continue
		}
		if entry.SourceChargeID() == nil || *entry.SourceChargeID() != sourceChargeID {
			continue
		}

		return true
	}

	return false
}

func correctionEntrySubAccounts(input ledger.TransactionInput) (string, string, error) {
	var fboSubAccountID string
	var receivableSubAccountID string

	for _, entry := range input.EntryInputs() {
		switch entry.PostingAddress().AccountType() {
		case ledger.AccountTypeCustomerFBO:
			if !entry.Amount().IsNegative() {
				continue
			}
			fboSubAccountID = entry.PostingAddress().SubAccountID()
		case ledger.AccountTypeCustomerReceivable:
			if !entry.Amount().IsPositive() {
				continue
			}
			receivableSubAccountID = entry.PostingAddress().SubAccountID()
		}
	}

	if fboSubAccountID == "" {
		return "", "", errors.New("issue correction FBO entry is required")
	}
	if receivableSubAccountID == "" {
		return "", "", errors.New("issue correction receivable entry is required")
	}

	return fboSubAccountID, receivableSubAccountID, nil
}

func (s *service) persistCommittedVoidRecords(ctx context.Context, pending []PendingRecord, group ledger.TransactionGroup) error {
	if len(pending) == 0 {
		return nil
	}
	if group == nil {
		return errors.New("transaction group is required")
	}

	pendingByID := make(map[string]PendingRecord, len(pending))
	for _, item := range pending {
		pendingByID[item.ID.ID] = item
	}

	records := make([]Record, 0, len(pending))
	groupID := group.ID().ID
	for _, tx := range group.Transactions() {
		recordID, ok := creditVoidRecordID(tx.Annotations())
		if !ok {
			continue
		}

		pendingRecord, ok := pendingByID[recordID]
		if !ok {
			return fmt.Errorf("committed void transaction %s has unknown record id %s", tx.ID().ID, recordID)
		}

		record := pendingRecord.Record
		record.VoidTransactionGroupID = groupID
		record.VoidTransactionID = tx.ID().ID
		record.Annotations = tx.Annotations()

		records = append(records, record)
		delete(pendingByID, recordID)
	}

	if len(pendingByID) > 0 {
		return fmt.Errorf("missing committed void transactions for %d pending records", len(pendingByID))
	}

	return s.adapter.CreateRecords(ctx, CreateRecordsInput{Records: records})
}

func (s *service) ListVoidedCreditImpacts(ctx context.Context, input ListVoidedCreditImpactsInput) (ListVoidedCreditImpactsResult, error) {
	if err := input.Validate(); err != nil {
		return ListVoidedCreditImpactsResult{}, err
	}

	records, err := s.adapter.ListRecords(ctx, ListRecordsInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       input.AsOf,
		Route:      input.Route,
	})
	if err != nil {
		return ListVoidedCreditImpactsResult{}, fmt.Errorf("list void records: %w", err)
	}
	if len(records) == 0 {
		return ListVoidedCreditImpactsResult{
			Items: []VoidImpact{},
		}, nil
	}

	groups := make(map[voidImpactGroupKey]*voidImpactGroup)
	for _, record := range records {
		key := voidImpactGroupKey{
			voidedAt:           record.VoidedAt,
			currency:           record.Currency,
			sourceChargeID:     record.SourceChargeID,
			transactionGroupID: record.VoidTransactionGroupID,
		}

		group := groups[key]
		if group == nil {
			group = &voidImpactGroup{
				id:          record.ID,
				createdAt:   record.CreatedAt,
				voidedAt:    record.VoidedAt,
				currency:    record.Currency,
				annotations: models.Annotations{},
			}
			groups[key] = group
		}

		group.amount = group.amount.Add(record.Amount)
		if group.id.ID == "" || record.ID.ID < group.id.ID {
			group.id = record.ID
		}
		if group.createdAt.IsZero() || record.CreatedAt.Before(group.createdAt) {
			group.createdAt = record.CreatedAt
		}
		for k, v := range record.Annotations {
			group.annotations[k] = v
		}
		group.annotations[ledger.AnnotationChargeID] = record.SourceChargeID
	}

	items := make([]VoidImpact, 0, len(groups))
	for _, group := range groups {
		if group.amount.IsZero() {
			continue
		}
		if group.amount.IsNegative() {
			return ListVoidedCreditImpactsResult{}, fmt.Errorf("void amount is negative for %s %s", group.voidedAt, group.currency)
		}

		item := VoidImpact{
			ID:          group.id,
			CreatedAt:   group.createdAt,
			VoidedAt:    group.voidedAt,
			CustomerID:  input.CustomerID,
			Currency:    group.currency,
			Amount:      group.amount.Neg(),
			Annotations: group.annotations,
		}
		if !voidImpactMatchesCursorWindow(item, input.After, input.Before) {
			continue
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b VoidImpact) int {
		return -a.Cursor().Compare(b.Cursor())
	})

	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}

	return ListVoidedCreditImpactsResult{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func voidImpactMatchesCursorWindow(item VoidImpact, after, before *ledger.TransactionCursor) bool {
	cursor := item.Cursor()

	if after != nil && cursor.Compare(*after) >= 0 {
		return false
	}
	if before != nil && cursor.Compare(*before) <= 0 {
		return false
	}

	return true
}

type voidImpactGroupKey struct {
	voidedAt           time.Time
	currency           currencyx.Code
	sourceChargeID     string
	transactionGroupID string
}

type voidImpactGroup struct {
	id          models.NamespacedID
	createdAt   time.Time
	voidedAt    time.Time
	currency    currencyx.Code
	amount      alpacadecimal.Decimal
	annotations models.Annotations
}

const AnnotationCreditVoidRecordID = "ledger.credit_void.record_id"

func creditVoidRecordAnnotations(recordID string) models.Annotations {
	return models.Annotations{
		AnnotationCreditVoidRecordID: recordID,
	}
}

func creditVoidRecordID(annotations models.Annotations) (string, bool) {
	raw, ok := annotations[AnnotationCreditVoidRecordID]
	if !ok {
		return "", false
	}

	value, ok := raw.(string)
	if !ok || value == "" {
		return "", false
	}

	return value, true
}

func newRecordID(namespace string) models.NamespacedID {
	return models.NamespacedID{
		Namespace: namespace,
		ID:        ulid.Make().String(),
	}
}
