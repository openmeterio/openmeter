package breakage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	// PlanIssuance creates the future expiration entries for newly issued
	// expiring credit. ImmediateReleaseAmount handles credit that covers already
	// consumed advance: the issued credit has an expiry, but the covered slice is
	// already used, so its planned breakage is released in the same ledger group.
	PlanIssuance(ctx context.Context, input PlanIssuanceInput) ([]ledger.TransactionInput, []PendingRecord, error)

	// ReleasePlan creates a future-dated inverse entry that reduces a planned
	// breakage amount because the underlying expiring credit has been consumed or
	// otherwise removed before expiry.
	ReleasePlan(ctx context.Context, input ReleasePlanInput) (ledger.TransactionInput, PendingRecord, error)

	// ReopenRelease creates a future-dated entry that increases breakage again
	// because a correction made previously consumed expiring credit unused.
	ReopenRelease(ctx context.Context, input ReopenReleaseInput) (ledger.TransactionInput, PendingRecord, error)

	// ListPlans returns unreleased planned breakage in the same order the FBO
	// collector must consume expiring credit.
	ListPlans(ctx context.Context, input ListPlansInput) ([]Plan, error)

	// ListReleases returns usage releases keyed by the original FBO source entry
	// ids, with already-reopened amounts removed.
	ListReleases(ctx context.Context, input ListReleasesInput) ([]Release, error)

	// ListExpiredRecords returns raw breakage rows that have reached expiry.
	// Consumers must net plan/release/reopen rows before showing customer-facing
	// expired credit transactions.
	ListExpiredRecords(ctx context.Context, input ListExpiredRecordsInput) ([]Record, error)

	// ListExpiredBreakageImpacts returns customer-visible breakage impacts by
	// netting raw breakage rows that have reached expiry.
	ListExpiredBreakageImpacts(ctx context.Context, input ListExpiredBreakageImpactsInput) (ListExpiredBreakageImpactsResult, error)

	// PersistCommittedRecords turns pending record metadata into durable
	// rows after the corresponding breakage ledger transactions have committed.
	PersistCommittedRecords(ctx context.Context, pending []PendingRecord, group ledger.TransactionGroup) error
}

type Config struct {
	// Adapter stores durable record rows. The ledger entries themselves are
	// still committed through the caller's ledger transaction group.
	Adapter      Adapter
	Dependencies transactions.ResolverDependencies
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Dependencies.AccountService == nil {
		errs = append(errs, errors.New("account service is required"))
	}

	if c.Dependencies.AccountCatalog == nil {
		errs = append(errs, errors.New("account catalog is required"))
	}

	return errors.Join(errs...)
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter: config.Adapter,
		deps:    config.Dependencies,
	}, nil
}

type service struct {
	adapter Adapter
	deps    transactions.ResolverDependencies
}

// PlanIssuanceInput describes newly issued expiring credit and, optionally, the
// slice that immediately covers already-consumed advance.
type PlanIssuanceInput struct {
	CustomerID customer.CustomerID

	Amount                 alpacadecimal.Decimal
	ImmediateReleaseAmount alpacadecimal.Decimal
	Currency               currencyx.Code
	CostBasis              *alpacadecimal.Decimal
	CreditPriority         *int
	ExpiresAt              time.Time
}

func (i PlanIssuanceInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if i.ImmediateReleaseAmount.IsNegative() {
		errs = append(errs, errors.New("immediate release amount cannot be negative"))
	}

	if i.ImmediateReleaseAmount.GreaterThan(i.Amount) {
		errs = append(errs, errors.New("immediate release amount cannot exceed amount"))
	}

	if err := ledger.ValidateCurrency(i.Currency); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if i.CostBasis != nil {
		if err := ledger.ValidateCostBasis(*i.CostBasis); err != nil {
			errs = append(errs, fmt.Errorf("cost basis: %w", err))
		}
	}

	if i.CreditPriority != nil {
		if err := ledger.ValidateCreditPriority(*i.CreditPriority); err != nil {
			errs = append(errs, fmt.Errorf("credit priority: %w", err))
		}
	}

	if i.ExpiresAt.IsZero() {
		errs = append(errs, errors.New("expires at is required"))
	}

	return errors.Join(errs...)
}

// ReleasePlanInput describes how much of one open plan should be released and
// which business flow caused the release.
type ReleasePlanInput struct {
	Plan                   Plan
	Amount                 alpacadecimal.Decimal
	SourceKind             SourceKind
	SourceEntryIdentityKey string
}

func (i ReleasePlanInput) Validate() error {
	var errs []error

	if err := i.Plan.Record.ValidateForReference(); err != nil {
		errs = append(errs, fmt.Errorf("plan: %w", err))
	}

	if i.Plan.Kind != ledger.BreakageKindPlan {
		errs = append(errs, errors.New("plan record must have kind plan"))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if i.Amount.GreaterThan(i.Plan.OpenAmount) {
		errs = append(errs, errors.New("release amount cannot exceed open plan amount"))
	}

	if i.Plan.FBOAddress == nil {
		errs = append(errs, errors.New("plan FBO address is required"))
	}

	if i.Plan.BreakageAddress == nil {
		errs = append(errs, errors.New("plan breakage address is required"))
	}

	switch i.SourceKind {
	case SourceKindUsage, SourceKindUsageCorrection, SourceKindCreditPurchaseCorrection, SourceKindAdvanceBackfill:
	default:
		errs = append(errs, fmt.Errorf("invalid release source kind: %s", i.SourceKind))
	}

	return errors.Join(errs...)
}

// ReopenReleaseInput describes how much of one released plan should be reopened
// and which correction flow caused it.
type ReopenReleaseInput struct {
	Release    Release
	Amount     alpacadecimal.Decimal
	SourceKind SourceKind
}

func (i ReopenReleaseInput) Validate() error {
	var errs []error

	if err := i.Release.Record.ValidateForReference(); err != nil {
		errs = append(errs, fmt.Errorf("release: %w", err))
	}

	if i.Release.Kind != ledger.BreakageKindRelease {
		errs = append(errs, errors.New("release record must have kind release"))
	}

	if i.Release.PlanID == nil || *i.Release.PlanID == "" {
		errs = append(errs, errors.New("release plan id is required"))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if i.Amount.GreaterThan(i.Release.OpenAmount) {
		errs = append(errs, errors.New("reopen amount cannot exceed open release amount"))
	}

	if i.Release.FBOAddress == nil {
		errs = append(errs, errors.New("release FBO address is required"))
	}

	if i.Release.BreakageAddress == nil {
		errs = append(errs, errors.New("release breakage address is required"))
	}

	switch i.SourceKind {
	case SourceKindUsageCorrection, SourceKindCreditPurchaseCorrection:
	default:
		errs = append(errs, fmt.Errorf("invalid reopen source kind: %s", i.SourceKind))
	}

	return errors.Join(errs...)
}

func (c Record) ValidateForReference() error {
	var errs []error

	if err := c.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("id: %w", err))
	}

	if err := c.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if err := c.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if c.ExpiresAt.IsZero() {
		errs = append(errs, errors.New("expires at is required"))
	}

	if c.FBOSubAccountID == "" {
		errs = append(errs, errors.New("FBO sub-account id is required"))
	}

	if c.BreakageSubAccountID == "" {
		errs = append(errs, errors.New("breakage sub-account id is required"))
	}

	return errors.Join(errs...)
}

// PlanIssuance returns ledger inputs instead of committing them. The caller owns
// the surrounding ledger transaction group so normal credit movement and
// breakage movement stay atomic.
func (s *service) PlanIssuance(ctx context.Context, input PlanIssuanceInput) ([]ledger.TransactionInput, []PendingRecord, error) {
	if err := input.Validate(); err != nil {
		return nil, nil, err
	}

	fboAddress, breakageAddress, err := s.resolvePlanAddresses(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	priority := resolveCreditPriority(input.CreditPriority)
	planID := newRecordID(input.CustomerID.Namespace)

	planRecord := PendingRecord{Record: Record{
		ID:                   planID,
		Kind:                 ledger.BreakageKindPlan,
		Amount:               input.Amount,
		CustomerID:           input.CustomerID,
		Currency:             input.Currency,
		CreditPriority:       priority,
		ExpiresAt:            input.ExpiresAt,
		SourceKind:           SourceKindCreditPurchase,
		FBOSubAccountID:      fboAddress.SubAccountID(),
		BreakageSubAccountID: breakageAddress.SubAccountID(),
	}}

	planTx, err := s.resolveBreakageTemplate(ctx, input.CustomerID, planID.ID, nil, transactions.PlanCustomerFBOBreakageTemplate{
		At:              input.ExpiresAt,
		Amount:          input.Amount,
		FBOAddress:      fboAddress,
		BreakageAddress: breakageAddress,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolve planned breakage: %w", err)
	}

	inputs := []ledger.TransactionInput{planTx}
	pending := []PendingRecord{planRecord}

	if input.ImmediateReleaseAmount.IsPositive() {
		releaseTx, releaseRecord, err := s.ReleasePlan(ctx, ReleasePlanInput{
			Plan: Plan{
				Record:          planRecord.Record,
				OpenAmount:      input.Amount,
				FBOAddress:      fboAddress,
				BreakageAddress: breakageAddress,
			},
			Amount:     input.ImmediateReleaseAmount,
			SourceKind: SourceKindAdvanceBackfill,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("resolve immediate breakage release: %w", err)
		}

		inputs = append(inputs, releaseTx)
		pending = append(pending, releaseRecord)
	}

	return inputs, pending, nil
}

func (s *service) ReleasePlan(ctx context.Context, input ReleasePlanInput) (ledger.TransactionInput, PendingRecord, error) {
	if err := input.Validate(); err != nil {
		return nil, PendingRecord{}, err
	}

	releaseID := newRecordID(input.Plan.ID.Namespace)
	planID := input.Plan.ID.ID

	record := PendingRecord{
		Record: Record{
			ID:                   releaseID,
			Kind:                 ledger.BreakageKindRelease,
			Amount:               input.Amount,
			CustomerID:           input.Plan.CustomerID,
			Currency:             input.Plan.Currency,
			CreditPriority:       input.Plan.CreditPriority,
			ExpiresAt:            input.Plan.ExpiresAt,
			SourceKind:           input.SourceKind,
			FBOSubAccountID:      input.Plan.FBOSubAccountID,
			BreakageSubAccountID: input.Plan.BreakageSubAccountID,
			PlanID:               &planID,
		},
		SourceEntryIdentityKey: input.SourceEntryIdentityKey,
	}

	tx, err := s.resolveBreakageTemplate(ctx, input.Plan.CustomerID, releaseID.ID, &planID, transactions.ReleaseCustomerFBOBreakageTemplate{
		At:              input.Plan.ExpiresAt,
		Amount:          input.Amount,
		FBOAddress:      input.Plan.FBOAddress,
		BreakageAddress: input.Plan.BreakageAddress,
	})
	if err != nil {
		return nil, PendingRecord{}, fmt.Errorf("resolve breakage release: %w", err)
	}

	return tx, record, nil
}

func (s *service) ReopenRelease(ctx context.Context, input ReopenReleaseInput) (ledger.TransactionInput, PendingRecord, error) {
	if err := input.Validate(); err != nil {
		return nil, PendingRecord{}, err
	}

	reopenID := newRecordID(input.Release.ID.Namespace)
	planID := *input.Release.PlanID
	releaseID := input.Release.ID.ID

	record := PendingRecord{Record: Record{
		ID:                   reopenID,
		Kind:                 ledger.BreakageKindReopen,
		Amount:               input.Amount,
		CustomerID:           input.Release.CustomerID,
		Currency:             input.Release.Currency,
		CreditPriority:       input.Release.CreditPriority,
		ExpiresAt:            input.Release.ExpiresAt,
		SourceKind:           input.SourceKind,
		FBOSubAccountID:      input.Release.FBOSubAccountID,
		BreakageSubAccountID: input.Release.BreakageSubAccountID,
		PlanID:               &planID,
		ReleaseID:            &releaseID,
	}}

	tx, err := s.resolveBreakageTemplate(ctx, input.Release.CustomerID, reopenID.ID, &planID, transactions.ReopenCustomerFBOBreakageTemplate{
		At:              input.Release.ExpiresAt,
		Amount:          input.Amount,
		FBOAddress:      input.Release.FBOAddress,
		BreakageAddress: input.Release.BreakageAddress,
	})
	if err != nil {
		return nil, PendingRecord{}, fmt.Errorf("resolve breakage reopen: %w", err)
	}

	return tx, record, nil
}

func (s *service) ListPlans(ctx context.Context, input ListPlansInput) ([]Plan, error) {
	records, err := s.adapter.ListCandidateRecords(ctx, input)
	if err != nil {
		return nil, err
	}

	plansByID := make(map[string]*Plan, len(records))
	planOrder := make([]string, 0, len(records))

	for _, record := range records {
		if record.Kind != ledger.BreakageKindPlan {
			continue
		}

		plan := &Plan{
			Record:     record,
			OpenAmount: record.Amount,
		}
		plansByID[record.ID.ID] = plan
		planOrder = append(planOrder, record.ID.ID)
	}

	for _, record := range records {
		if record.Kind == ledger.BreakageKindPlan || record.PlanID == nil {
			continue
		}

		plan := plansByID[*record.PlanID]
		if plan == nil {
			continue
		}

		switch record.Kind {
		case ledger.BreakageKindRelease:
			plan.OpenAmount = plan.OpenAmount.Sub(record.Amount)
		case ledger.BreakageKindReopen:
			plan.OpenAmount = plan.OpenAmount.Add(record.Amount)
		}
	}

	out := make([]Plan, 0, len(planOrder))
	for _, planID := range planOrder {
		plan := plansByID[planID]
		if plan == nil || !plan.OpenAmount.IsPositive() {
			continue
		}

		if err := s.hydratePlanAddresses(ctx, plan); err != nil {
			return nil, err
		}

		out = append(out, *plan)
	}

	return out, nil
}

func (s *service) ListReleases(ctx context.Context, input ListReleasesInput) ([]Release, error) {
	records, err := s.adapter.ListReleaseRecords(ctx, input)
	if err != nil {
		return nil, err
	}

	releasesByID := make(map[string]*Release, len(records))
	releaseOrder := make([]string, 0, len(records))

	for _, record := range records {
		if record.Kind != ledger.BreakageKindRelease {
			continue
		}

		release := &Release{
			Record:     record,
			OpenAmount: record.Amount,
		}
		releasesByID[record.ID.ID] = release
		releaseOrder = append(releaseOrder, record.ID.ID)
	}

	for _, record := range records {
		if record.Kind != ledger.BreakageKindReopen || record.ReleaseID == nil {
			continue
		}

		release := releasesByID[*record.ReleaseID]
		if release == nil {
			continue
		}

		release.OpenAmount = release.OpenAmount.Sub(record.Amount)
	}

	out := make([]Release, 0, len(releaseOrder))
	for _, releaseID := range releaseOrder {
		release := releasesByID[releaseID]
		if release == nil || !release.OpenAmount.IsPositive() {
			continue
		}

		if err := s.hydrateReleaseAddresses(ctx, release); err != nil {
			return nil, err
		}

		out = append(out, *release)
	}

	return out, nil
}

func (s *service) ListExpiredRecords(ctx context.Context, input ListExpiredRecordsInput) ([]Record, error) {
	if err := input.CustomerID.Validate(); err != nil {
		return nil, fmt.Errorf("customer id: %w", err)
	}

	if input.Currency != nil {
		if err := input.Currency.Validate(); err != nil {
			return nil, fmt.Errorf("currency: %w", err)
		}
	}

	if input.AsOf.IsZero() {
		return nil, errors.New("as of is required")
	}

	return s.adapter.ListExpiredRecords(ctx, input)
}

func (s *service) PersistCommittedRecords(ctx context.Context, pending []PendingRecord, group ledger.TransactionGroup) error {
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

	sourceEntriesByIdentity := committedSourceEntriesByIdentity(group)

	records := make([]Record, 0, len(pending))
	groupID := group.ID().ID
	for _, tx := range group.Transactions() {
		recordID, ok := breakageRecordID(tx.Annotations())
		if !ok {
			continue
		}

		pendingRecord, ok := pendingByID[recordID]
		if !ok {
			return fmt.Errorf("committed breakage transaction %s has unknown record id %s", tx.ID().ID, recordID)
		}

		record := pendingRecord.Record
		record.BreakageTransactionGroupID = groupID
		record.BreakageTransactionID = tx.ID().ID
		record.Annotations = tx.Annotations()
		if record.SourceTransactionGroupID == nil {
			record.SourceTransactionGroupID = &groupID
		}
		if pendingRecord.SourceEntryIdentityKey != "" {
			sourceEntry, ok := sourceEntriesByIdentity[pendingRecord.SourceEntryIdentityKey]
			if !ok {
				return fmt.Errorf("source entry with identity key %s not found for breakage record %s", pendingRecord.SourceEntryIdentityKey, recordID)
			}

			sourceTransactionID := sourceEntry.TransactionID().ID
			sourceEntryID := sourceEntry.ID().ID
			record.SourceTransactionID = &sourceTransactionID
			record.SourceEntryID = &sourceEntryID
		}

		records = append(records, record)
		delete(pendingByID, recordID)
	}

	if len(pendingByID) > 0 {
		return fmt.Errorf("missing committed breakage transactions for %d pending records", len(pendingByID))
	}

	return s.adapter.CreateRecords(ctx, CreateRecordsInput{Records: records})
}

func committedSourceEntriesByIdentity(group ledger.TransactionGroup) map[string]ledger.Entry {
	out := make(map[string]ledger.Entry)

	for _, tx := range group.Transactions() {
		if _, ok := breakageRecordID(tx.Annotations()); ok {
			continue
		}

		for _, entry := range tx.Entries() {
			if entry.IdentityKey() == "" {
				continue
			}

			out[entry.IdentityKey()] = entry
		}
	}

	return out
}

func (s *service) resolvePlanAddresses(ctx context.Context, input PlanIssuanceInput) (ledger.PostingAddress, ledger.PostingAddress, error) {
	customerAccounts, err := s.deps.AccountService.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return nil, nil, fmt.Errorf("get customer accounts: %w", err)
	}

	businessAccounts, err := s.deps.AccountService.GetBusinessAccounts(ctx, input.CustomerID.Namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("get business accounts: %w", err)
	}

	fboSubAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       input.Currency,
		CostBasis:      input.CostBasis,
		CreditPriority: resolveCreditPriority(input.CreditPriority),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get FBO sub-account: %w", err)
	}

	breakageSubAccount, err := businessAccounts.BreakageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:  input.Currency,
		CostBasis: input.CostBasis,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get breakage sub-account: %w", err)
	}

	return fboSubAccount.Address(), breakageSubAccount.Address(), nil
}

func (s *service) hydratePlanAddresses(ctx context.Context, plan *Plan) error {
	fboSubAccount, err := s.deps.AccountCatalog.GetSubAccountByID(ctx, models.NamespacedID{
		Namespace: plan.ID.Namespace,
		ID:        plan.FBOSubAccountID,
	})
	if err != nil {
		return fmt.Errorf("get FBO sub-account %s: %w", plan.FBOSubAccountID, err)
	}

	breakageSubAccount, err := s.deps.AccountCatalog.GetSubAccountByID(ctx, models.NamespacedID{
		Namespace: plan.ID.Namespace,
		ID:        plan.BreakageSubAccountID,
	})
	if err != nil {
		return fmt.Errorf("get breakage sub-account %s: %w", plan.BreakageSubAccountID, err)
	}

	plan.FBOAddress = fboSubAccount.Address()
	plan.BreakageAddress = breakageSubAccount.Address()

	return nil
}

func (s *service) hydrateReleaseAddresses(ctx context.Context, release *Release) error {
	fboSubAccount, err := s.deps.AccountCatalog.GetSubAccountByID(ctx, models.NamespacedID{
		Namespace: release.ID.Namespace,
		ID:        release.FBOSubAccountID,
	})
	if err != nil {
		return fmt.Errorf("get FBO sub-account %s: %w", release.FBOSubAccountID, err)
	}

	breakageSubAccount, err := s.deps.AccountCatalog.GetSubAccountByID(ctx, models.NamespacedID{
		Namespace: release.ID.Namespace,
		ID:        release.BreakageSubAccountID,
	})
	if err != nil {
		return fmt.Errorf("get breakage sub-account %s: %w", release.BreakageSubAccountID, err)
	}

	release.FBOAddress = fboSubAccount.Address()
	release.BreakageAddress = breakageSubAccount.Address()

	return nil
}

func (s *service) resolveBreakageTemplate(
	ctx context.Context,
	customerID customer.CustomerID,
	recordID string,
	planID *string,
	template transactions.TransactionTemplate,
) (ledger.TransactionInput, error) {
	inputs, err := transactions.ResolveTransactions(
		ctx,
		s.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  customerID.Namespace,
		},
		template,
	)
	if err != nil {
		return nil, err
	}

	if len(inputs) != 1 {
		return nil, fmt.Errorf("expected one breakage transaction input, got %d", len(inputs))
	}

	return transactions.WithAnnotations(inputs[0], ledger.BreakageAnnotations(breakageKindForTemplate(template), recordID, planID)), nil
}

func breakageKindForTemplate(template transactions.TransactionTemplate) ledger.BreakageKind {
	switch template.(type) {
	case transactions.PlanCustomerFBOBreakageTemplate:
		return ledger.BreakageKindPlan
	case transactions.ReleaseCustomerFBOBreakageTemplate:
		return ledger.BreakageKindRelease
	case transactions.ReopenCustomerFBOBreakageTemplate:
		return ledger.BreakageKindReopen
	default:
		panic(fmt.Sprintf("unsupported breakage template %T", template))
	}
}

func breakageRecordID(annotations models.Annotations) (string, bool) {
	raw, ok := annotations[ledger.AnnotationBreakageRecordID]
	if !ok {
		return "", false
	}

	value, ok := raw.(string)
	if !ok || value == "" {
		return "", false
	}

	return value, true
}

func resolveCreditPriority(priority *int) int {
	if priority == nil {
		return ledger.DefaultCustomerFBOPriority
	}

	return *priority
}

func newRecordID(namespace string) models.NamespacedID {
	return models.NamespacedID{
		Namespace: namespace,
		ID:        ulid.Make().String(),
	}
}
