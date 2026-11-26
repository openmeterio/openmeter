package customerbalance

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// JournalCheckpoint captures the ledger inputs to live allocation, before any
// charge has consumed them. Sources stay in the order used by collection.
type JournalCheckpoint struct {
	Scope          journalScope          `json:"scope"`
	EntryCount     int                   `json:"entryCount"`
	EffectiveCount int                   `json:"effectiveCount"`
	AsOf           time.Time             `json:"asOf"`
	NextBooking    *time.Time            `json:"nextBooking,omitempty"`
	BookedFBO      alpacadecimal.Decimal `json:"bookedFBO"`
	SpendableFBO   alpacadecimal.Decimal `json:"spendableFBO"`
	Advance        alpacadecimal.Decimal `json:"advance"`
	Sources        []journalSource       `json:"sources"`
}

type journalSource struct {
	SubAccountID string                `json:"subAccountId"`
	Priority     *int                  `json:"priority,omitempty"`
	Filters      ledger.CreditFilters  `json:"filters"`
	Amount       alpacadecimal.Decimal `json:"amount"`
}

type journalScope struct {
	FBOAccount        string `json:"fboAccount"`
	ReceivableAccount string `json:"receivableAccount"`
	Currency          string `json:"currency"`
	AllFeatures       bool   `json:"allFeatures"`
	Feature           string `json:"feature,omitempty"`
}

func (i GetIncrementalBalanceInput) journalScope(accounts ledger.CustomerAccounts) (journalScope, error) {
	// A hydrated custom currency matches its exact persisted precision. Its
	// unhydrated reference is a prefix filter, even for the same currency ID.
	serialize := i.Currency.MarshalText
	if !i.Currency.IsResolved() {
		serialize = i.Currency.MarshalTextPrefix
	}

	currency, err := serialize()
	if err != nil {
		return journalScope{}, err
	}

	scope := journalScope{
		FBOAccount:        accounts.FBOAccount.ID().ID,
		ReceivableAccount: accounts.ReceivableAccount.ID().ID,
		Currency:          string(currency),
		AllFeatures:       i.FeatureFilter.IsAbsent(),
	}
	if features := i.FeatureFilter.OrEmpty(); len(features) > 0 {
		scope.Feature = features[0]
	}

	return scope, nil
}

func (cp JournalCheckpoint) matches(scope journalScope, at time.Time) bool {
	if cp.Scope != scope || cp.Sources == nil || cp.EffectiveCount < 0 || cp.EffectiveCount > cp.EntryCount ||
		cp.AsOf.IsZero() || at.Before(cp.AsOf) {
		return false
	}

	// A missing next booking can only mean all entries are already effective.
	if (cp.NextBooking == nil) != (cp.EffectiveCount == cp.EntryCount) ||
		(cp.NextBooking != nil && !cp.NextBooking.After(cp.AsOf)) {
		return false
	}

	return !slices.ContainsFunc(cp.Sources, func(source journalSource) bool {
		return source.SubAccountID == "" || !source.Amount.IsPositive()
	})
}

func (cp JournalCheckpoint) projection() ledgerProjection {
	sources := make([]liveBalanceSource, len(cp.Sources))
	for i, source := range cp.Sources {
		sources[i] = liveBalanceSource{
			cursor: source.SubAccountID,
			route:  ledger.Route{Filters: source.Filters, CreditPriority: source.Priority},
			amount: source.Amount,
		}
	}

	return ledgerProjection{bookedFBO: cp.BookedFBO, spendableFBO: cp.SpendableFBO, advance: cp.Advance, sources: sources}
}

type GetIncrementalBalanceInput struct {
	GetBalanceServiceInput
	// StartedAt sets the opening boundary for this current calculation. Charges
	// and live usage still use their normal current-read semantics.
	StartedAt   time.Time
	Checkpoints []JournalCheckpoint
}

func (i GetIncrementalBalanceInput) Validate() error {
	if i.BalanceQuery.AsOf != nil || i.BalanceQuery.After != nil {
		return models.NewGenericValidationError(errors.New("journal checkpoints require a current balance query"))
	}

	return i.GetBalanceServiceInput.Validate()
}

type GetIncrementalBalanceResult struct {
	Balance    Balance
	Checkpoint *JournalCheckpoint
	Reused     bool
}

// GetIncrementalBalance reuses a verified ledger projection and always reads
// the current charge lifecycle, pending grants and usage through GetBalance's
// allocation path.
func (s *service) GetIncrementalBalance(ctx context.Context, input GetIncrementalBalanceInput) (GetIncrementalBalanceResult, error) {
	if err := input.Validate(); err != nil {
		return GetIncrementalBalanceResult{}, err
	}

	var journal ledger.JournalReader
	if provider, ok := s.BalanceQuerier.(interface{ BalanceJournal() ledger.JournalReader }); ok {
		journal = provider.BalanceJournal()
	}

	if journal == nil {
		value, err := s.GetBalance(ctx, input.GetBalanceServiceInput)
		return GetIncrementalBalanceResult{Balance: value}, err
	}

	at := input.StartedAt
	if at.IsZero() {
		at = clock.Now()
	}

	accounts, err := s.AccountResolver.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return GetIncrementalBalanceResult{}, fmt.Errorf("get customer accounts: %w", err)
	}

	scope, err := input.journalScope(accounts)
	if err != nil {
		return GetIncrementalBalanceResult{}, fmt.Errorf("get journal scope: %w", err)
	}

	entries := ledger.AccountEntryScope{
		Namespace:  input.CustomerID.Namespace,
		AccountIDs: []string{scope.FBOAccount, scope.ReceivableAccount},
	}
	coverageInput := ledger.AccountBookingCoverageInput{AccountEntryScope: entries, AsOf: at}
	candidates := make([]JournalCheckpoint, 0, len(input.Checkpoints))
	var entryCount *int
	for _, cp := range input.Checkpoints {
		if !cp.matches(scope, at) {
			continue
		}

		if cp.NextBooking == nil || at.Before(*cp.NextBooking) {
			if entryCount == nil {
				count, err := journal.CountAccountEntries(ctx, entries)
				if err != nil {
					return GetIncrementalBalanceResult{}, fmt.Errorf("count account entries: %w", err)
				}

				entryCount = &count
			}

			if cp.EntryCount == *entryCount {
				value, err := s.applyChargeBalances(ctx, input.GetBalanceServiceInput, cp.projection())
				return GetIncrementalBalanceResult{Balance: value, Reused: true}, err
			}
		}

		candidates = append(candidates, cp)
		coverageInput.Earlier = append(coverageInput.Earlier, cp.AsOf)
	}

	coverage, err := journal.GetAccountBookingCoverage(ctx, coverageInput)
	if err != nil {
		return GetIncrementalBalanceResult{}, fmt.Errorf("verify journal prefixes: %w", err)
	}

	var previous *JournalCheckpoint
	for i := range candidates {
		// Postings are append-only. An unchanged count behind this boundary
		// proves every posting that makes up the opening amount is still there.
		if candidates[i].EffectiveCount == coverage.EffectiveEarlier[i] {
			previous = &candidates[i]
			break
		}
	}

	var projection ledgerProjection
	if previous == nil {
		projection, err = s.getBalanceSources(ctx, input.GetBalanceServiceInput, accounts, ledger.BalanceQuery{AsOf: &at})
	} else {
		projection, err = s.advanceJournal(ctx, input.GetBalanceServiceInput, journal, *previous, at)
	}

	if err != nil {
		return GetIncrementalBalanceResult{}, err
	}

	// A backdated posting can commit after prefix verification, outside the
	// delta we just read. Retry that case once through the unconditional path.
	after, err := journal.CountAccountEntries(ctx, entries)
	if err != nil {
		return GetIncrementalBalanceResult{}, fmt.Errorf("recheck account entries: %w", err)
	}

	if after != coverage.EntryCount {
		if previous != nil {
			value, err := s.GetBalance(ctx, input.GetBalanceServiceInput)
			return GetIncrementalBalanceResult{Balance: value}, err
		}

		value, err := s.applyChargeBalances(ctx, input.GetBalanceServiceInput, projection)

		return GetIncrementalBalanceResult{Balance: value}, err
	}

	checkpoint := &JournalCheckpoint{
		Scope: scope, AsOf: at, NextBooking: coverage.NextBooking,
		EntryCount: coverage.EntryCount, EffectiveCount: coverage.EffectiveAtAsOf,
		BookedFBO: projection.bookedFBO, SpendableFBO: projection.spendableFBO, Advance: projection.advance,
		Sources: make([]journalSource, len(projection.sources)),
	}
	for i, source := range projection.sources {
		checkpoint.Sources[i] = journalSource{
			SubAccountID: source.cursor, Priority: source.route.CreditPriority, Filters: source.route.Filters, Amount: source.amount,
		}
	}

	value, err := s.applyChargeBalances(ctx, input.GetBalanceServiceInput, projection)

	return GetIncrementalBalanceResult{Balance: value, Checkpoint: checkpoint, Reused: previous != nil}, err
}

func (s *service) advanceJournal(ctx context.Context, input GetBalanceServiceInput, journal ledger.JournalReader, previous JournalCheckpoint, at time.Time) (ledgerProjection, error) {
	spendable, sources, err := s.advanceSpendable(ctx, input, journal, previous, at)
	if err != nil {
		return ledgerProjection{}, err
	}

	fbo, err := journal.SumAccountMovement(ctx, ledger.AccountMovementInput{
		AccountID: models.NamespacedID{Namespace: input.CustomerID.Namespace, ID: previous.Scope.FBOAccount},
		Route:     input.bookedRoute(), After: previous.AsOf, Through: at,
	})
	if err != nil {
		return ledgerProjection{}, fmt.Errorf("get FBO movements: %w", err)
	}

	advance, err := journal.SumAccountMovement(ctx, ledger.AccountMovementInput{
		AccountID: models.NamespacedID{Namespace: input.CustomerID.Namespace, ID: previous.Scope.ReceivableAccount},
		Route:     input.advanceRoute(), After: previous.AsOf, Through: at,
	})
	if err != nil {
		return ledgerProjection{}, fmt.Errorf("get advance movements: %w", err)
	}

	return ledgerProjection{
		bookedFBO: previous.BookedFBO.Add(fbo), spendableFBO: spendable,
		advance: previous.Advance.Add(advance), sources: sources,
	}, nil
}

func (s *service) advanceSpendable(ctx context.Context, input GetBalanceServiceInput, journal ledger.JournalReader, previous JournalCheckpoint, at time.Time) (alpacadecimal.Decimal, []liveBalanceSource, error) {
	// Include future bookings. A new reservation can change the spendable
	// minimum immediately, even though it has no booked impact yet.
	const maxSubAccounts = 256
	changed, err := journal.ListSubAccountsBookedAfter(ctx, ledger.SubAccountsBookedAfterInput{
		AccountEntryScope: ledger.AccountEntryScope{Namespace: input.CustomerID.Namespace, AccountIDs: []string{previous.Scope.FBOAccount}},
		After:             previous.AsOf, Limit: maxSubAccounts + 1,
	})
	if err != nil {
		return alpacadecimal.Zero, nil, fmt.Errorf("find changed FBO subaccounts: %w", err)
	}

	if len(changed) > maxSubAccounts {
		return s.getSpendableSources(ctx, input, previous.Scope.FBOAccount, &at, nil)
	}

	projection := previous.projection()
	if len(changed) == 0 {
		return projection.spendableFBO, projection.sources, nil
	}

	spendable, sources, err := s.getSpendableSources(ctx, input, previous.Scope.FBOAccount, &at, changed)
	if err != nil {
		return alpacadecimal.Zero, nil, err
	}

	// The prefix has already been verified. Subaccounts with no later postings
	// have exactly the same per-grant minima as at the checkpoint.
	changedIDs := make(map[string]struct{}, len(changed))
	for _, id := range changed {
		changedIDs[id] = struct{}{}
	}

	for _, source := range projection.sources {
		if _, changed := changedIDs[source.cursor]; changed {
			continue
		}

		sources = append(sources, source)
		spendable = spendable.Add(source.amount)
	}

	slices.SortStableFunc(sources, cmpx.Compare[liveBalanceSource])

	return spendable, sources, nil
}
