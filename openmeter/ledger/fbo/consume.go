package fbo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ConsumeDestination string

const (
	ConsumeDestinationAccrued    ConsumeDestination = "accrued"
	ConsumeDestinationReceivable ConsumeDestination = "receivable"
)

var _ models.Validator = ConsumeDestination("")

func (d ConsumeDestination) Validate() error {
	switch d {
	case ConsumeDestinationAccrued, ConsumeDestinationReceivable:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid consume destination: %q", d))
	}
}

type AdvancePolicy string

const (
	AdvancePolicyNever       AdvancePolicy = "never"
	AdvancePolicyOnShortfall AdvancePolicy = "on_shortfall"
)

var _ models.Validator = AdvancePolicy("")

func (p AdvancePolicy) Validate() error {
	switch p {
	case AdvancePolicyNever, AdvancePolicyOnShortfall:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid advance policy: %q", p))
	}
}

type ConsumeInput struct {
	breakage.PostingInput
	CustomerID    customer.CustomerID
	ChargeID      string
	BookedAt      time.Time
	Currency      currencies.CurrencyReference
	Filters       ledger.CreditFilters
	Amount        alpacadecimal.Decimal
	Destination   ConsumeDestination
	AdvancePolicy AdvancePolicy
	TaxCode       *string
	TaxBehavior   *ledger.TaxBehavior
}

var _ models.Validator = ConsumeInput{}

func (i ConsumeInput) Validate() error {
	var errs []error
	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := i.PostingInput.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("posting: %w", err))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	} else if i.Currency.IsCustom() && !i.Currency.IsResolved() {
		errs = append(errs, errors.New("custom currency must be resolved"))
	}

	if err := i.Filters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("filters: %w", err))
	}

	if i.Amount.IsNegative() {
		errs = append(errs, errors.New("amount cannot be negative"))
	} else if i.Amount.IsPositive() {
		if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
			errs = append(errs, fmt.Errorf("amount: %w", err))
		}
	}

	if err := i.Destination.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("destination: %w", err))
	}

	if err := i.AdvancePolicy.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("advance policy: %w", err))
	}

	if i.Destination == ConsumeDestinationReceivable {
		if !i.Currency.IsFiat() {
			errs = append(errs, errors.New("receivable coverage requires fiat currency"))
		}

		if i.AdvancePolicy != AdvancePolicyNever {
			errs = append(errs, errors.New("receivable coverage cannot create advance"))
		}

		if i.TaxCode != nil || i.TaxBehavior != nil {
			errs = append(errs, errors.New("receivable coverage cannot set accrued tax dimensions"))
		}
	}

	if i.TaxBehavior != nil {
		if err := i.TaxBehavior.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax behavior: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Consume selects credit and returns the resolved consumption, breakage release,
// and shortfall advance transactions. The caller must commit them under the supplied
// group ID in the same database transaction: breakage bookkeeping is written here,
// so any later failure requires rollback.
func (s *service) Consume(ctx context.Context, request ConsumeInput) ([]ledger.TransactionInput, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	if request.Amount.IsZero() {
		return nil, nil
	}

	selectedCredits, creditTransactions, err := s.selectCreditsAndBuildTransactions(ctx, request)
	if err != nil {
		return nil, err
	}

	// Each origin identifies one concrete FBO debit. Assign its IDs before
	// writing breakage records that reference the still-unposted consumption.
	sourceDebitsByOrigin, err := assignSourceDebitIDs(creditTransactions)
	if err != nil {
		return nil, err
	}

	breakageReleases, err := s.releaseConsumedBreakage(ctx, releaseConsumedBreakageInput{
		ReleasePosting:       request.PostingInput,
		SpendChargeID:        request.ChargeID,
		SelectedCredits:      selectedCredits,
		SourceDebitsByOrigin: sourceDebitsByOrigin,
	})
	if err != nil {
		return nil, err
	}

	postings := append(creditTransactions, breakageReleases...)
	shortfall := request.Amount
	for _, credit := range selectedCredits {
		shortfall = shortfall.Sub(credit.amount)
	}

	// Advances fund only the shortfall; they are issued and consumed together.
	if request.AdvancePolicy == AdvancePolicyOnShortfall && shortfall.IsPositive() {
		advancePostings, err := s.advance.PlanIssue(ctx, advance.IssueInput{
			CustomerID:  request.CustomerID,
			ChargeID:    request.ChargeID,
			At:          request.BookedAt,
			Amount:      shortfall,
			Currency:    request.Currency,
			Filters:     request.Filters,
			TaxCode:     request.TaxCode,
			TaxBehavior: request.TaxBehavior,
		})
		if err != nil {
			return nil, fmt.Errorf("plan FBO shortfall advance: %w", err)
		}

		postings = append(postings, advancePostings...)
	}

	for idx, posting := range postings {
		postings[idx] = transactions.WithAnnotations(posting, request.Annotations)
	}

	return postings, nil
}

// selectCreditsAndBuildTransactions applies source priority and builds
// transactions for the requested destination. Each selected
// slice gets its own collection origin, even when slices share an issuing charge.
func (s *service) selectCreditsAndBuildTransactions(ctx context.Context, request ConsumeInput) (selectionList, []ledger.TransactionInput, error) {
	sources, err := s.listCustomerFBOSources(ctx, SourceQuery{
		CustomerID: request.CustomerID,
		Currency:   request.Currency,
		Filters:    request.Filters,
		AsOf:       request.BookedAt,
	})
	if err != nil {
		return nil, nil, err
	}

	selectedCredits := selectFBOSources(sources, request.Amount)
	for idx := range selectedCredits {
		selectedCredits[idx].collectionOriginID = lo.ToPtr(ulid.Make().String())
	}

	sourceAmounts := selectionList(selectedCredits).postingAmounts(&request.ChargeID)
	if len(sourceAmounts) == 0 {
		return selectedCredits, nil, nil
	}

	var template transactions.TransactionTemplate
	switch request.Destination {
	case ConsumeDestinationAccrued:
		template = transactions.TransferCustomerFBOToAccruedTemplate{
			At:          request.BookedAt,
			Currency:    request.Currency,
			TaxCode:     request.TaxCode,
			TaxBehavior: request.TaxBehavior,
			Sources:     sourceAmounts,
		}
	case ConsumeDestinationReceivable:
		template = transactions.CoverCustomerReceivableTemplate{
			At:       request.BookedAt,
			Currency: request.Currency,
			Sources:  sourceAmounts,
		}
	}

	creditTransactions, err := transactions.ResolveTransactions(ctx, s.dependencies, transactions.ResolutionScope{
		CustomerID: request.CustomerID,
		Namespace:  request.CustomerID.Namespace,
	}, template)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve FBO consumption: %w", err)
	}

	return selectedCredits, creditTransactions, nil
}

type sourcePostingIDs struct {
	TransactionID string
	EntryID       string
}

// assignSourceDebitIDs replaces transactions with identified copies and returns the
// exact FBO debit references used to link breakage releases to consumed credit.
func assignSourceDebitIDs(creditTransactions []ledger.TransactionInput) (map[string]sourcePostingIDs, error) {
	sourceDebitsByOrigin := make(map[string]sourcePostingIDs, len(creditTransactions))
	for idx, posting := range creditTransactions {
		identified, err := ledger.PreassignIDs(posting)
		if err != nil {
			return nil, err
		}

		creditTransactions[idx] = identified
		for _, entry := range identified.EntryInputs() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO || !entry.Amount().IsNegative() {
				continue
			}

			originID := lo.FromPtr(entry.Provenance().CollectionOriginID)
			if _, exists := sourceDebitsByOrigin[originID]; exists {
				return nil, fmt.Errorf("collection origin %s has multiple FBO source entries", originID)
			}

			sourceDebitsByOrigin[originID] = sourcePostingIDs{
				TransactionID: identified.AssignedID(),
				EntryID:       entry.AssignedID(),
			}
		}
	}

	return sourceDebitsByOrigin, nil
}

type releaseConsumedBreakageInput struct {
	ReleasePosting       breakage.PostingInput
	SpendChargeID        string
	SelectedCredits      selectionList
	SourceDebitsByOrigin map[string]sourcePostingIDs
}

var _ models.Validator = releaseConsumedBreakageInput{}

func (i releaseConsumedBreakageInput) Validate() error {
	var errs []error
	if err := i.ReleasePosting.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.SpendChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// releaseConsumedBreakage writes release records for selected expiring credit.
// Their transactions join the same group as the source debits they reference.
func (s *service) releaseConsumedBreakage(ctx context.Context, input releaseConsumedBreakageInput) ([]ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	var breakageReleases []ledger.TransactionInput
	remainingBreakageByPlanID := make(map[string]alpacadecimal.Decimal)
	for _, selection := range input.SelectedCredits {
		consumedSource, ok := input.SourceDebitsByOrigin[lo.FromPtr(selection.collectionOriginID)]
		if !ok {
			return nil, errors.New("selected credit has no resolved FBO source posting")
		}

		if selection.source.breakagePlan != nil {
			breakagePlan := *selection.source.breakagePlan
			remaining, ok := remainingBreakageByPlanID[breakagePlan.ID.ID]
			if !ok {
				remaining = breakagePlan.OpenAmount
			}

			// Legacy source-less plans can span several selected source slices;
			// cap their combined release before writing more bookkeeping.
			if selection.amount.GreaterThan(remaining) {
				return nil, fmt.Errorf("release amount exceeds remaining breakage plan amount for %s", breakagePlan.ID.ID)
			}

			remainingBreakageByPlanID[breakagePlan.ID.ID] = remaining.Sub(selection.amount)
			breakageRelease, err := s.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
				PostingInput:        input.ReleasePosting,
				Plan:                breakagePlan,
				Amount:              selection.amount,
				SourceKind:          breakage.SourceKindUsage,
				SourceChargeID:      selection.source.sourceChargeID,
				SpendChargeID:       &input.SpendChargeID,
				CollectionOriginID:  selection.collectionOriginID,
				SourceTransactionID: &consumedSource.TransactionID,
				SourceEntryID:       &consumedSource.EntryID,
			})
			if err != nil {
				return nil, fmt.Errorf("release consumed credit breakage: %w", err)
			}

			breakageReleases = append(breakageReleases, breakageRelease)
		}
	}

	return breakageReleases, nil
}
