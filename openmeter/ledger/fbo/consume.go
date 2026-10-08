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

type AdvancePolicy string

const (
	AdvancePolicyNever       AdvancePolicy = "never"
	AdvancePolicyOnShortfall AdvancePolicy = "on_shortfall"
)

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

	switch i.Destination {
	case ConsumeDestinationAccrued:
	case ConsumeDestinationReceivable:
		if !i.Currency.IsFiat() {
			errs = append(errs, errors.New("receivable coverage requires fiat currency"))
		}

		if i.AdvancePolicy != AdvancePolicyNever {
			errs = append(errs, errors.New("receivable coverage cannot create advance"))
		}

		if i.TaxCode != nil || i.TaxBehavior != nil {
			errs = append(errs, errors.New("receivable coverage cannot set accrued tax dimensions"))
		}
	default:
		errs = append(errs, errors.New("consume destination is invalid"))
	}

	if i.AdvancePolicy != AdvancePolicyNever && i.AdvancePolicy != AdvancePolicyOnShortfall {
		errs = append(errs, errors.New("advance policy is invalid"))
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

type ConsumePlan struct {
	Sources         []ConsumedSource
	CoveredAmount   alpacadecimal.Decimal
	AdvanceAmount   alpacadecimal.Decimal
	UncoveredAmount alpacadecimal.Decimal
	Inputs          []ledger.TransactionInput
}

func (s *service) PlanConsume(ctx context.Context, scope *Scope, input ConsumeInput) (plan ConsumePlan, err error) {
	if err := s.validateScope(ctx, scope); err != nil {
		return plan, err
	}

	defer func() {
		if err != nil {
			scope.failure = err
		}
	}()
	if err := input.Validate(); err != nil {
		return plan, err
	}

	if input.Amount.IsZero() {
		return plan, nil
	}

	sources, err := s.listCustomerFBOSources(ctx, SourceQuery{
		CustomerID: scope.customerID,
		Currency:   input.Currency,
		Filters:    input.Filters,
		AsOf:       input.SourceBalanceAsOf,
	}, scope)
	if err != nil {
		return plan, err
	}

	selected := selectFBOSources(sources, input.Amount)
	for idx := range selected {
		selected[idx].collectionOriginID = lo.ToPtr(ulid.Make().String())
	}

	amounts := selectionList(selected).postingAmounts(&input.ChargeID)
	if len(amounts) > 0 {
		var template transactions.TransactionTemplate
		switch input.Destination {
		case ConsumeDestinationAccrued:
			template = transactions.TransferCustomerFBOToAccruedTemplate{
				At:          input.BookedAt,
				Currency:    input.Currency,
				TaxCode:     input.TaxCode,
				TaxBehavior: input.TaxBehavior,
				Sources:     amounts,
			}
		case ConsumeDestinationReceivable:
			template = transactions.CoverCustomerReceivableTemplate{
				At:       input.BookedAt,
				Currency: input.Currency,
				Sources:  amounts,
			}
		}

		plan.Inputs, err = transactions.ResolveTransactions(ctx, s.dependencies, transactions.ResolutionScope{CustomerID: scope.customerID, Namespace: scope.customerID.Namespace}, template)
		if err != nil {
			return plan, fmt.Errorf("resolve FBO consumption: %w", err)
		}
	}

	// Each origin identifies one concrete FBO debit. Assign its IDs before
	// writing breakage records that reference the still-unposted consumption.
	sourcesByOrigin := make(map[string]ConsumedSource, len(selected))
	for idx, posting := range plan.Inputs {
		identified, err := ledger.PreassignIDs(posting)
		if err != nil {
			return plan, err
		}

		plan.Inputs[idx] = identified
		for _, entry := range identified.EntryInputs() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO || !entry.Amount().IsNegative() {
				continue
			}

			originID := lo.FromPtr(entry.Provenance().CollectionOriginID)
			if _, exists := sourcesByOrigin[originID]; exists {
				return plan, fmt.Errorf("collection origin %s has multiple FBO source entries", originID)
			}

			sourcesByOrigin[originID] = ConsumedSource{
				Address:            entry.PostingAddress(),
				SourceChargeID:     entry.Provenance().SourceChargeID,
				CollectionOriginID: entry.Provenance().CollectionOriginID,
				Amount:             entry.Amount().Abs(),
				TransactionID:      identified.AssignedID(),
				EntryID:            entry.AssignedID(),
			}
		}
	}

	remainingByPlan := make(map[string]alpacadecimal.Decimal)
	for _, selection := range selected {
		source, ok := sourcesByOrigin[lo.FromPtr(selection.collectionOriginID)]
		if !ok {
			return plan, errors.New("selected credit has no resolved FBO source posting")
		}

		if selection.source.breakagePlan != nil {
			expiry := *selection.source.breakagePlan
			remaining, ok := remainingByPlan[expiry.ID.ID]
			if !ok {
				remaining = expiry.OpenAmount
			}

			// Legacy source-less plans can span several selected source slices;
			// cap their combined release before writing more bookkeeping.
			if selection.amount.GreaterThan(remaining) {
				return plan, fmt.Errorf("release amount exceeds remaining breakage plan amount for %s", expiry.ID.ID)
			}

			remainingByPlan[expiry.ID.ID] = remaining.Sub(selection.amount)
			source.BreakagePlanID = &expiry.ID.ID
			release, err := s.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
				PostingInput: breakage.PostingInput{
					TransactionGroupID: scope.groupID,
					Annotations:        input.Annotations,
				},
				Plan:                expiry,
				Amount:              selection.amount,
				SourceKind:          breakage.SourceKindUsage,
				SourceChargeID:      source.SourceChargeID,
				SpendChargeID:       &input.ChargeID,
				CollectionOriginID:  source.CollectionOriginID,
				SourceTransactionID: &source.TransactionID,
				SourceEntryID:       &source.EntryID,
			})
			if err != nil {
				return plan, fmt.Errorf("release consumed credit expiry: %w", err)
			}

			plan.Inputs = append(plan.Inputs, release)
		}

		plan.Sources = append(plan.Sources, source)
		plan.CoveredAmount = plan.CoveredAmount.Add(selection.amount)
	}

	plan.UncoveredAmount = input.Amount.Sub(plan.CoveredAmount)
	if input.AdvancePolicy == AdvancePolicyOnShortfall && plan.UncoveredAmount.IsPositive() {
		advanceInputs, err := s.advance.PlanIssue(ctx, advance.IssueInput{
			CustomerID:  scope.customerID,
			ChargeID:    input.ChargeID,
			At:          input.BookedAt,
			Amount:      plan.UncoveredAmount,
			Currency:    input.Currency,
			Filters:     input.Filters,
			TaxCode:     input.TaxCode,
			TaxBehavior: input.TaxBehavior,
		})
		if err != nil {
			return plan, fmt.Errorf("plan FBO shortfall advance: %w", err)
		}

		plan.Inputs = append(plan.Inputs, advanceInputs...)
		plan.AdvanceAmount = plan.UncoveredAmount
		plan.UncoveredAmount = alpacadecimal.Zero
	}

	for idx, posting := range plan.Inputs {
		identified, err := ledger.PreassignIDs(posting)
		if err != nil {
			return plan, err
		}

		plan.Inputs[idx] = transactions.WithAnnotations(identified, input.Annotations)
	}

	for _, selected := range selected {
		key := keyForSource(selected.source.address, selected.source.sourceChargeID)
		scope.reserved[key] = scope.reserved[key].Add(selected.amount)
	}

	scope.inputs = append(scope.inputs, plan.Inputs...)

	return plan, nil
}
