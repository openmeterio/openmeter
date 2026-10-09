package breakage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service writes bookkeeping in the caller's transaction and returns identified
// postings for its group. A successful operation cannot be discarded independently.
type Service interface {
	// PlanIssuance creates the future expiration entries for newly issued
	// expiring credit. ImmediateReleases handles credit that covers already
	// consumed advance: the issued credit has an expiry, but the covered slice is
	// already used, so its planned breakage is released in the same ledger group.
	PlanIssuance(ctx context.Context, input PlanIssuanceInput) ([]ledger.TransactionInput, error)

	// ReleasePlan creates a future-dated inverse entry that reduces a planned
	// breakage amount because the underlying expiring credit has been consumed or
	// otherwise removed before expiry.
	ReleasePlan(ctx context.Context, input ReleasePlanInput) (ledger.TransactionInput, error)

	// ReopenRelease creates a future-dated entry that increases breakage again
	// because a correction made previously consumed expiring credit unused.
	ReopenRelease(ctx context.Context, input ReopenReleaseInput) (ledger.TransactionInput, error)

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
	PostingInput

	CustomerID customer.CustomerID

	Amount            alpacadecimal.Decimal
	ImmediateReleases []PlanIssuanceImmediateRelease
	Currency          currencies.CurrencyReference
	CostBasisCurrency *currencyx.Code
	TaxCode           *string
	TaxBehavior       *ledger.TaxBehavior
	CostBasis         *alpacadecimal.Decimal
	CreditPriority    *int
	Filters           ledger.CreditFilters
	ExpiresAt         time.Time
	SourceChargeID    *string
}

type PlanIssuanceImmediateRelease struct {
	Amount             alpacadecimal.Decimal
	SpendChargeID      *string
	CollectionOriginID *string
}

func (i PlanIssuanceInput) Validate() error {
	var errs []error

	if err := i.PostingInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	immediateReleaseAmount := alpacadecimal.Zero
	for idx, release := range i.ImmediateReleases {
		if release.Amount.IsNegative() {
			errs = append(errs, fmt.Errorf("immediate releases[%d]: amount cannot be negative", idx))
		}

		immediateReleaseAmount = immediateReleaseAmount.Add(release.Amount)
	}

	if immediateReleaseAmount.GreaterThan(i.Amount) {
		errs = append(errs, errors.New("immediate release amount cannot exceed amount"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	} else if i.Currency.IsCustom() && !i.Currency.IsResolved() {
		errs = append(errs, errors.New("custom currency must be resolved"))
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

	if i.TaxBehavior != nil {
		if err := i.TaxBehavior.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax behavior: %w", err))
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
	PostingInput

	Plan                Plan
	Amount              alpacadecimal.Decimal
	SourceKind          SourceKind
	SourceTransactionID *string
	SourceEntryID       *string
	SourceChargeID      *string
	SpendChargeID       *string
	CollectionOriginID  *string
}

func (i ReleasePlanInput) Validate() error {
	var errs []error

	if err := i.PostingInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.SourceEntryID != nil && i.SourceTransactionID == nil {
		errs = append(errs, errors.New("source entry requires a source transaction"))
	}

	for field, id := range map[string]*string{
		"source transaction id": i.SourceTransactionID,
		"source entry id":       i.SourceEntryID,
	} {
		if id == nil {
			continue
		}

		if *id == "" {
			errs = append(errs, fmt.Errorf("%s cannot be empty", field))
		} else if err := ledger.ValidateAssignedID(*id); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", field, err))
		}
	}

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
	case SourceKindCreditPurchase, SourceKindUsage, SourceKindUsageCorrection, SourceKindCreditPurchaseCorrection, SourceKindAdvanceBackfill:
	default:
		errs = append(errs, fmt.Errorf("invalid release source kind: %s", i.SourceKind))
	}

	return errors.Join(errs...)
}

// ReopenReleaseInput describes how much of one released plan should be reopened
// and which correction flow caused it.
type ReopenReleaseInput struct {
	PostingInput

	Release            Release
	Amount             alpacadecimal.Decimal
	SourceKind         SourceKind
	SourceChargeID     *string
	SpendChargeID      *string
	CollectionOriginID *string
}

func (i ReopenReleaseInput) Validate() error {
	var errs []error

	if err := i.PostingInput.Validate(); err != nil {
		errs = append(errs, err)
	}

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
func (s *service) PlanIssuance(ctx context.Context, input PlanIssuanceInput) ([]ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	fboAddress, breakageAddress, err := s.resolvePlanAddresses(ctx, input)
	if err != nil {
		return nil, err
	}

	priority := resolveCreditPriority(input.CreditPriority)
	planID := newRecordID(input.CustomerID.Namespace)

	planRecord := Record{
		ID:                   planID,
		Kind:                 ledger.BreakageKindPlan,
		Amount:               input.Amount,
		CustomerID:           input.CustomerID,
		Currency:             input.Currency.Code,
		CreditPriority:       priority,
		ExpiresAt:            input.ExpiresAt,
		SourceKind:           SourceKindCreditPurchase,
		SourceChargeID:       input.SourceChargeID,
		FBOSubAccountID:      fboAddress.SubAccountID(),
		BreakageSubAccountID: breakageAddress.SubAccountID(),
	}

	planTx, err := s.resolveBreakageTemplate(ctx, input.CustomerID, planID.ID, nil, transactions.PlanCustomerFBOBreakageTemplate{
		At:              input.ExpiresAt,
		Amount:          input.Amount,
		FBOAddress:      fboAddress,
		BreakageAddress: breakageAddress,
		FBOIdentity: ledger.EntryIdentityParts{
			Provenance: ledger.Provenance{
				SourceChargeID: input.SourceChargeID,
			},
		},
		BreakageIdentity: ledger.EntryIdentityParts{
			Provenance: ledger.Provenance{
				SourceChargeID: input.SourceChargeID,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("resolve planned breakage: %w", err)
	}

	planTx, err = s.recordPosting(ctx, input.PostingInput, planRecord, planTx)
	if err != nil {
		return nil, err
	}

	inputs := []ledger.TransactionInput{planTx}

	for _, immediateRelease := range input.ImmediateReleases {
		if !immediateRelease.Amount.IsPositive() {
			continue
		}

		releaseTx, err := s.ReleasePlan(ctx, ReleasePlanInput{
			PostingInput: input.PostingInput,
			Plan: Plan{
				Record:          planRecord,
				OpenAmount:      input.Amount,
				FBOAddress:      fboAddress,
				BreakageAddress: breakageAddress,
			},
			Amount:             immediateRelease.Amount,
			SourceKind:         SourceKindAdvanceBackfill,
			SourceChargeID:     input.SourceChargeID,
			SpendChargeID:      immediateRelease.SpendChargeID,
			CollectionOriginID: immediateRelease.CollectionOriginID,
		})
		if err != nil {
			return nil, fmt.Errorf("resolve immediate breakage release: %w", err)
		}

		inputs = append(inputs, releaseTx)
	}

	return inputs, nil
}

func (s *service) ReleasePlan(ctx context.Context, input ReleasePlanInput) (ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	releaseID := newRecordID(input.Plan.ID.Namespace)
	planID := input.Plan.ID.ID

	record := Record{
		ID:                   releaseID,
		Kind:                 ledger.BreakageKindRelease,
		Amount:               input.Amount,
		CustomerID:           input.Plan.CustomerID,
		Currency:             input.Plan.Currency,
		CreditPriority:       input.Plan.CreditPriority,
		ExpiresAt:            input.Plan.ExpiresAt,
		SourceKind:           input.SourceKind,
		SourceChargeID:       input.SourceChargeID,
		FBOSubAccountID:      input.Plan.FBOSubAccountID,
		BreakageSubAccountID: input.Plan.BreakageSubAccountID,
		PlanID:               &planID,
		SourceTransactionID:  input.SourceTransactionID,
		SourceEntryID:        input.SourceEntryID,
	}

	tx, err := s.resolveBreakageTemplate(ctx, input.Plan.CustomerID, releaseID.ID, &planID, transactions.ReleaseCustomerFBOBreakageTemplate{
		At:              input.Plan.ExpiresAt,
		Amount:          input.Amount,
		FBOAddress:      input.Plan.FBOAddress,
		BreakageAddress: input.Plan.BreakageAddress,
		FBOIdentity: ledger.EntryIdentityParts{
			Provenance: ledger.Provenance{
				SourceChargeID:     input.SourceChargeID,
				SpendChargeID:      input.SpendChargeID,
				CollectionOriginID: input.CollectionOriginID,
			},
		},
		BreakageIdentity: releaseBreakageIdentity(input.SourceChargeID, input.SpendChargeID, input.CollectionOriginID),
	})
	if err != nil {
		return nil, fmt.Errorf("resolve breakage release: %w", err)
	}

	return s.recordPosting(ctx, input.PostingInput, record, tx)
}

func (s *service) ReopenRelease(ctx context.Context, input ReopenReleaseInput) (ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	reopenID := newRecordID(input.Release.ID.Namespace)
	planID := *input.Release.PlanID
	releaseID := input.Release.ID.ID

	record := Record{
		ID:                   reopenID,
		Kind:                 ledger.BreakageKindReopen,
		Amount:               input.Amount,
		CustomerID:           input.Release.CustomerID,
		Currency:             input.Release.Currency,
		CreditPriority:       input.Release.CreditPriority,
		ExpiresAt:            input.Release.ExpiresAt,
		SourceKind:           input.SourceKind,
		SourceChargeID:       input.SourceChargeID,
		FBOSubAccountID:      input.Release.FBOSubAccountID,
		BreakageSubAccountID: input.Release.BreakageSubAccountID,
		PlanID:               &planID,
		ReleaseID:            &releaseID,
	}

	tx, err := s.resolveBreakageTemplate(ctx, input.Release.CustomerID, reopenID.ID, &planID, transactions.ReopenCustomerFBOBreakageTemplate{
		At:              input.Release.ExpiresAt,
		Amount:          input.Amount,
		FBOAddress:      input.Release.FBOAddress,
		BreakageAddress: input.Release.BreakageAddress,
		FBOIdentity: ledger.EntryIdentityParts{
			Provenance: ledger.Provenance{
				SourceChargeID:     input.SourceChargeID,
				SpendChargeID:      input.SpendChargeID,
				CollectionOriginID: input.CollectionOriginID,
			},
		},
		BreakageIdentity: releaseBreakageIdentity(input.SourceChargeID, input.SpendChargeID, input.CollectionOriginID),
	})
	if err != nil {
		return nil, fmt.Errorf("resolve breakage reopen: %w", err)
	}

	return s.recordPosting(ctx, input.PostingInput, record, tx)
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

	if err := ValidateExpiredRouteFilter(input.Route); err != nil {
		return nil, fmt.Errorf("route: %w", err)
	}

	return s.adapter.ListExpiredRecords(ctx, input)
}

// recordPosting writes complete bookkeeping before the journal group is posted.
// Deferred ledger FKs reject discarded postings at the enclosing transaction's commit.
func (s *service) recordPosting(ctx context.Context, input PostingInput, record Record, posting ledger.TransactionInput) (ledger.TransactionInput, error) {
	if _, err := transaction.GetDriverFromContext(ctx); err != nil {
		return nil, fmt.Errorf("breakage requires the caller's transaction: %w", err)
	}

	posting, err := ledger.PreassignIDs(posting)
	if err != nil {
		return nil, fmt.Errorf("assign breakage posting IDs: %w", err)
	}

	posting = transactions.WithAnnotations(posting, input.Annotations)

	record.BreakageTransactionGroupID = input.TransactionGroupID
	record.BreakageTransactionID = posting.AssignedID()
	record.SourceTransactionGroupID = &input.TransactionGroupID
	record.Annotations = posting.Annotations()

	if err := s.adapter.CreateRecords(ctx, CreateRecordsInput{Records: []Record{record}}); err != nil {
		return nil, fmt.Errorf("persist breakage record: %w", err)
	}

	return posting, nil
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
		Currency:          input.Currency,
		CostBasisCurrency: input.CostBasisCurrency,
		CostBasis:         input.CostBasis,
		CreditPriority:    resolveCreditPriority(input.CreditPriority),
		Filters:           input.Filters,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get FBO sub-account: %w", err)
	}

	breakageSubAccount, err := businessAccounts.BreakageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:          input.Currency,
		CostBasisCurrency: input.CostBasisCurrency,
		CostBasis:         input.CostBasis,
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

	kind, err := breakageKindForTemplate(template)
	if err != nil {
		return nil, err
	}

	return transactions.WithAnnotations(inputs[0], ledger.BreakageAnnotations(kind, recordID, planID)), nil
}

func breakageKindForTemplate(template transactions.TransactionTemplate) (ledger.BreakageKind, error) {
	switch template.(type) {
	case transactions.PlanCustomerFBOBreakageTemplate:
		return ledger.BreakageKindPlan, nil
	case transactions.ReleaseCustomerFBOBreakageTemplate:
		return ledger.BreakageKindRelease, nil
	case transactions.ReopenCustomerFBOBreakageTemplate:
		return ledger.BreakageKindReopen, nil
	default:
		return "", fmt.Errorf("unsupported breakage template %T", template)
	}
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

// Legacy releases used source-only breakage provenance. Origin-tracked releases
// carry the same origin and spend on both legs so the origin balances independently.
func releaseBreakageIdentity(source, spend, origin *string) ledger.EntryIdentityParts {
	identity := ledger.EntryIdentityParts{Provenance: ledger.Provenance{
		SourceChargeID:     source,
		CollectionOriginID: origin,
	}}
	if origin != nil {
		identity.SpendChargeID = spend
	}

	return identity
}
