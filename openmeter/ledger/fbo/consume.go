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
	ChargeID          string
	Annotations       models.Annotations
	BookedAt          time.Time
	SourceBalanceAsOf time.Time
	Currency          currencies.CurrencyReference
	Filters           ledger.CreditFilters
	Amount            alpacadecimal.Decimal
	Destination       ConsumeDestination
	AdvancePolicy     AdvancePolicy
	TaxCode           *string
	TaxBehavior       *ledger.TaxBehavior
}

var _ models.Validator = ConsumeInput{}

func (i ConsumeInput) Validate() error {
	var errs []error
	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}

	if i.SourceBalanceAsOf.IsZero() {
		errs = append(errs, errors.New("source balance as of is required"))
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

type ConsumedSource struct {
	Address            ledger.PostingAddress
	SourceChargeID     *string
	CollectionOriginID *string
	Amount             alpacadecimal.Decimal
	TransactionID      string
	EntryID            string
	BreakagePlanID     *string
}

// Consumption records one operation's amounts, selected sources, and accounting
// references. Its postings belong to the Plan and cannot be committed separately.
type Consumption struct {
	Sources         []ConsumedSource
	CoveredAmount   alpacadecimal.Decimal
	AdvanceAmount   alpacadecimal.Decimal
	UncoveredAmount alpacadecimal.Decimal
	TransactionIDs  []string
}

// Transactions resolves the accounting references after the plan's group is
// committed. Reference order preserves consumption order independently of the
// group's transaction enumeration, which billing uses for allocation ordering.
func (c Consumption) Transactions(group ledger.TransactionGroup) ([]ledger.Transaction, error) {
	if group == nil {
		return nil, errors.New("transaction group is required")
	}

	posted := group.Transactions()
	byID := make(map[string]ledger.Transaction, len(posted))
	for _, transaction := range posted {
		byID[transaction.ID().ID] = transaction
	}

	transactions := make([]ledger.Transaction, 0, len(c.TransactionIDs))
	for _, id := range c.TransactionIDs {
		transaction, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("consumption transaction %s is missing from group %s", id, group.ID().ID)
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (p *Plan) Consume(ctx context.Context, request ConsumeInput) (consumption Consumption, err error) {
	if err := p.validate(); err != nil {
		return consumption, err
	}

	// Breakage bookkeeping may already be written when a later step fails.
	// Such a plan cannot be continued or finalized; its caller must roll back.
	defer func() {
		if err != nil {
			p.failure = err
		}
	}()
	if err := request.Validate(); err != nil {
		return consumption, err
	}

	if request.Amount.IsZero() {
		return consumption, nil
	}

	selectedCredits, creditTransactions, err := p.selectCreditsAndBuildTransactions(ctx, request)
	if err != nil {
		return consumption, err
	}

	// Each origin identifies one concrete FBO debit. Assign its IDs before
	// writing breakage records that reference the still-unposted consumption.
	sourceDebitsByOrigin, err := assignSourceDebitIDs(creditTransactions)
	if err != nil {
		return consumption, err
	}

	consumption, breakageReleases, err := p.buildConsumptionAndReleaseBreakage(ctx, buildConsumptionAndReleaseBreakageInput{
		ReleasePosting: breakage.PostingInput{
			TransactionGroupID: p.groupID,
			Annotations:        request.Annotations,
		},
		SpendChargeID:        request.ChargeID,
		SelectedCredits:      selectedCredits,
		SourceDebitsByOrigin: sourceDebitsByOrigin,
	})
	if err != nil {
		return consumption, err
	}

	postings := append(creditTransactions, breakageReleases...)

	// Advances fund only the shortfall; they are issued and consumed together.
	consumption.UncoveredAmount = request.Amount.Sub(consumption.CoveredAmount)
	if request.AdvancePolicy == AdvancePolicyOnShortfall && consumption.UncoveredAmount.IsPositive() {
		advancePostings, err := p.owner.advance.PlanIssue(ctx, advance.IssueInput{
			CustomerID:  p.customerID,
			ChargeID:    request.ChargeID,
			At:          request.BookedAt,
			Amount:      consumption.UncoveredAmount,
			Currency:    request.Currency,
			Filters:     request.Filters,
			TaxCode:     request.TaxCode,
			TaxBehavior: request.TaxBehavior,
		})
		if err != nil {
			return consumption, fmt.Errorf("plan FBO shortfall advance: %w", err)
		}

		postings = append(postings, advancePostings...)
		consumption.AdvanceAmount = consumption.UncoveredAmount
		consumption.UncoveredAmount = alpacadecimal.Zero
	}

	// Keep references in planning order, including breakage and advance postings.
	// IDs already used by breakage bookkeeping must survive this final pass.
	for idx, posting := range postings {
		identified, err := ledger.PreassignIDs(posting)
		if err != nil {
			return consumption, err
		}

		postings[idx] = transactions.WithAnnotations(identified, request.Annotations)
		consumption.TransactionIDs = append(consumption.TransactionIDs, identified.AssignedID())
	}

	// Reserve credit only after the entire operation succeeds. Later Consume
	// calls must see it as used even though the group has not been posted yet.
	for _, credit := range selectedCredits {
		key := keyForSource(credit.source.address, credit.source.sourceChargeID)
		p.reserved[key] = p.reserved[key].Add(credit.amount)
	}

	p.inputs = append(p.inputs, postings...)

	return consumption, nil
}

// selectCreditsAndBuildTransactions applies source priority and existing plan
// reservations, then builds transactions for the requested destination. Each selected
// slice gets its own collection origin, even when slices share an issuing charge.
func (p *Plan) selectCreditsAndBuildTransactions(ctx context.Context, request ConsumeInput) (selectionList, []ledger.TransactionInput, error) {
	sources, err := p.owner.listCustomerFBOSources(ctx, SourceQuery{
		CustomerID: p.customerID,
		Currency:   request.Currency,
		Filters:    request.Filters,
		AsOf:       request.SourceBalanceAsOf,
	}, p)
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

	creditTransactions, err := transactions.ResolveTransactions(ctx, p.owner.dependencies, transactions.ResolutionScope{
		CustomerID: p.customerID,
		Namespace:  p.customerID.Namespace,
	}, template)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve FBO consumption: %w", err)
	}

	return selectedCredits, creditTransactions, nil
}

// assignSourceDebitIDs replaces transactions with identified copies and returns the
// exact FBO debit references used to link breakage releases to consumed credit.
func assignSourceDebitIDs(creditTransactions []ledger.TransactionInput) (map[string]ConsumedSource, error) {
	sourceDebitsByOrigin := make(map[string]ConsumedSource, len(creditTransactions))
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

			sourceDebitsByOrigin[originID] = ConsumedSource{
				Address:            entry.PostingAddress(),
				SourceChargeID:     entry.Provenance().SourceChargeID,
				CollectionOriginID: entry.Provenance().CollectionOriginID,
				Amount:             entry.Amount().Abs(),
				TransactionID:      identified.AssignedID(),
				EntryID:            entry.AssignedID(),
			}
		}
	}

	return sourceDebitsByOrigin, nil
}

type buildConsumptionAndReleaseBreakageInput struct {
	ReleasePosting       breakage.PostingInput
	SpendChargeID        string
	SelectedCredits      selectionList
	SourceDebitsByOrigin map[string]ConsumedSource
}

var _ models.Validator = buildConsumptionAndReleaseBreakageInput{}

func (i buildConsumptionAndReleaseBreakageInput) Validate() error {
	var errs []error
	if err := i.ReleasePosting.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.SpendChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// buildConsumptionAndReleaseBreakage builds the consumption receipt in selection
// order and releases scheduled breakage for consumed slices. Release records are
// written now; their transactions join the same group as the source debits.
// On failure, the receipt contains only sources whose processing succeeded.
func (p *Plan) buildConsumptionAndReleaseBreakage(ctx context.Context, input buildConsumptionAndReleaseBreakageInput) (consumption Consumption, breakageReleases []ledger.TransactionInput, err error) {
	if err := input.Validate(); err != nil {
		return consumption, nil, err
	}

	remainingBreakageByPlanID := make(map[string]alpacadecimal.Decimal)
	for _, selection := range input.SelectedCredits {
		consumedSource, ok := input.SourceDebitsByOrigin[lo.FromPtr(selection.collectionOriginID)]
		if !ok {
			return consumption, nil, errors.New("selected credit has no resolved FBO source posting")
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
				return consumption, nil, fmt.Errorf("release amount exceeds remaining breakage plan amount for %s", breakagePlan.ID.ID)
			}

			remainingBreakageByPlanID[breakagePlan.ID.ID] = remaining.Sub(selection.amount)
			consumedSource.BreakagePlanID = &breakagePlan.ID.ID
			breakageRelease, err := p.owner.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
				PostingInput:        input.ReleasePosting,
				Plan:                breakagePlan,
				Amount:              selection.amount,
				SourceKind:          breakage.SourceKindUsage,
				SourceChargeID:      consumedSource.SourceChargeID,
				SpendChargeID:       &input.SpendChargeID,
				CollectionOriginID:  consumedSource.CollectionOriginID,
				SourceTransactionID: &consumedSource.TransactionID,
				SourceEntryID:       &consumedSource.EntryID,
			})
			if err != nil {
				return consumption, nil, fmt.Errorf("release consumed credit breakage: %w", err)
			}

			breakageReleases = append(breakageReleases, breakageRelease)
		}

		consumption.Sources = append(consumption.Sources, consumedSource)
		consumption.CoveredAmount = consumption.CoveredAmount.Add(selection.amount)
	}

	return consumption, breakageReleases, nil
}
