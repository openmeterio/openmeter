package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) PlanLegacyCorrection(ctx context.Context, input advance.LegacyCorrectionInput) (advance.CorrectionPlan, error) {
	var out advance.CorrectionPlan

	if err := input.Validate(); err != nil {
		return out, err
	}

	if input.BackingGroupID != nil {
		group, err := s.ledger.GetTransactionGroup(ctx, models.NamespacedID{
			Namespace: input.CustomerID.Namespace,
			ID:        *input.BackingGroupID,
		})
		if err != nil {
			return out, fmt.Errorf("get backing transaction group: %w", err)
		}

		translation, err := advance.FindLegacyBackfillTransaction(advance.LegacyBackfillTransactionInput{
			Group:        group,
			Original:     input.Collection,
			AccountType:  ledger.AccountTypeCustomerAccrued,
			TemplateCode: transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}),
		})
		if err != nil {
			return out, err
		}

		if translation != nil {
			out.LegacyCorrections = append(out.LegacyCorrections, transactions.CorrectionInput{
				At:                  input.At,
				Amount:              input.Amount,
				OriginalTransaction: translation,
				OriginalGroup:       group,
			})
		}

		attribution, err := advance.FindLegacyBackfillTransaction(advance.LegacyBackfillTransactionInput{
			Group:        group,
			Original:     input.Issue,
			AccountType:  ledger.AccountTypeCustomerReceivable,
			TemplateCode: transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}),
		})
		if err != nil {
			return out, err
		}

		if attribution == nil {
			return out, fmt.Errorf("backing group %s has no receivable attribution matching the collected source", group.ID().ID)
		}

		out.LegacyCorrections = append(out.LegacyCorrections, transactions.CorrectionInput{
			At:                  input.At,
			Amount:              input.Amount,
			OriginalTransaction: attribution,
			OriginalGroup:       group,
		})

		out.Inputs, out.BreakagePending, err = s.resolveAdvanceBackfillBreakageReopenInputs(ctx, input, group, input.Amount)
		if err != nil {
			return out, err
		}

		reissued, err := s.reissueBackfilledCredit(ctx, input, group, input.Amount)
		if err != nil {
			return out, err
		}

		out.Inputs = append(out.Inputs, reissued...)
	}

	for _, original := range []ledger.Transaction{input.Collection, input.Issue} {
		out.LegacyCorrections = append(out.LegacyCorrections, transactions.CorrectionInput{
			At:                  input.At,
			Amount:              input.Amount,
			OriginalTransaction: original,
			OriginalGroup:       input.OriginalGroup,
		})
	}

	return out, nil
}

func (s *service) reissueBackfilledCredit(ctx context.Context, input advance.LegacyCorrectionInput, backingGroup ledger.TransactionGroup, amount alpacadecimal.Decimal) ([]ledger.TransactionInput, error) {
	// Re-issue into the same known-cost and priority bucket the backfill had used
	// so the released value becomes ordinary purchased credit again. It can be
	// consumed later, but we do not immediately redirect it onto some other
	// uncovered advance during this correction flow.
	route, err := s.backfilledCreditReissueRoute(backingGroup)
	if err != nil {
		return nil, err
	}

	resolved, err := transactions.ResolveTransactions(
		ctx,
		s.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: input.CustomerID,
			Namespace:  input.CustomerID.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:                input.At,
			Amount:            amount,
			Currency:          route.currency,
			CostBasisCurrency: route.costBasisCurrency,
			CostBasis:         route.costBasis,
			Filters:           route.filters,
			CreditPriority:    route.creditPriority,
			SourceChargeID:    route.sourceChargeID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve re-issued purchased credit: %w", err)
	}

	out := make([]ledger.TransactionInput, 0, len(resolved))

	for _, txInput := range resolved {
		out = append(out, transactions.WithAnnotations(txInput, ledger.TransactionAnnotations(
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
			ledger.TransactionDirectionCorrection,
		)))
	}

	return out, nil
}

func (s *service) resolveAdvanceBackfillBreakageReopenInputs(ctx context.Context, input advance.LegacyCorrectionInput, backingGroup ledger.TransactionGroup, amount alpacadecimal.Decimal) ([]ledger.TransactionInput, []breakage.PendingRecord, error) {
	releases, err := s.breakage.ListReleases(ctx, breakage.ListReleasesInput{
		CustomerID:               input.CustomerID,
		SourceTransactionGroupID: []string{backingGroup.ID().ID},
		ReleaseSourceKind:        []breakage.SourceKind{breakage.SourceKindAdvanceBackfill},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list advance-backfill breakage releases: %w", err)
	}

	inputs := make([]ledger.TransactionInput, 0, len(releases))
	pending := make([]breakage.PendingRecord, 0, len(releases))
	releaseFactsByTransactionID := breakageReleaseFactsByTransactionID(backingGroup)
	remaining := amount
	for _, release := range releases {
		if !remaining.IsPositive() {
			break
		}

		reopenAmount := minDecimal(release.OpenAmount, remaining)
		if !reopenAmount.IsPositive() {
			continue
		}

		releaseFacts := releaseFactsByTransactionID[release.BreakageTransactionID]
		if releaseFacts.SpendChargeID != nil && *releaseFacts.SpendChargeID != input.ChargeID {
			continue
		}
		reopenInput, reopenRecord, err := s.breakage.ReopenRelease(ctx, breakage.ReopenReleaseInput{
			Release:        release,
			Amount:         reopenAmount,
			SourceKind:     breakage.SourceKindUsageCorrection,
			SourceChargeID: releaseFacts.SourceChargeID,
			SpendChargeID:  releaseFacts.SpendChargeID,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("resolve advance-backfill breakage reopen: %w", err)
		}

		inputs = append(inputs, reopenInput)
		pending = append(pending, reopenRecord)
		remaining = remaining.Sub(reopenAmount)
	}

	return inputs, pending, nil
}

func breakageReleaseFactsByTransactionID(group ledger.TransactionGroup) map[string]ledger.EntryIdentityParts {
	out := make(map[string]ledger.EntryIdentityParts)

	for _, tx := range group.Transactions() {
		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO {
				continue
			}

			out[tx.ID().ID] = ledger.EntryIdentityParts{
				Provenance: ledger.Provenance{
					SourceChargeID: entry.Provenance().SourceChargeID,
					SpendChargeID:  entry.Provenance().SpendChargeID,
				},
			}
			break
		}
	}

	return out
}

type backfilledCreditReissueRouteResult struct {
	currency          currencies.CurrencyReference
	costBasisCurrency *currencyx.Code
	costBasis         *alpacadecimal.Decimal
	creditPriority    *int
	filters           ledger.CreditFilters
	sourceChargeID    *string
}

func (s *service) backfilledCreditReissueRoute(group ledger.TransactionGroup) (backfilledCreditReissueRouteResult, error) {
	// A correction of backfilled advance turns already-covered value back into
	// ordinary FBO credit. Use the backing group's known-cost route for that
	// re-issue. If the group has an FBO route, prefer it because it also carries
	// the customer credit collection priority. Fully backfilled purchases may
	// only have the known cost basis on receivable/accrued attribution entries.
	var fallbackCurrency currencies.CurrencyReference
	var fallbackCostBasisCurrency *currencyx.Code
	var fallbackCostBasis *alpacadecimal.Decimal
	var fallbackFilters ledger.CreditFilters
	var sourceChargeID *string

	for _, transaction := range group.Transactions() {
		for _, entry := range transaction.Entries() {
			if sourceChargeID == nil && entry.Provenance().SourceChargeID != nil {
				sourceChargeID = entry.Provenance().SourceChargeID
			}

			route := entry.PostingAddress().Route().Route()
			if route.CostBasis == nil {
				continue
			}

			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				return backfilledCreditReissueRouteResult{
					currency:          route.Currency,
					costBasisCurrency: route.CostBasisCurrency,
					costBasis:         route.CostBasis,
					creditPriority:    route.CreditPriority,
					filters:           route.Filters,
					sourceChargeID:    sourceChargeID,
				}, nil
			}

			if fallbackCostBasis == nil {
				fallbackCurrency = route.Currency
				fallbackCostBasisCurrency = route.CostBasisCurrency
				fallbackCostBasis = route.CostBasis
				fallbackFilters = route.Filters
			}
		}
	}

	if fallbackCostBasis != nil {
		return backfilledCreditReissueRouteResult{
			currency:          fallbackCurrency,
			costBasisCurrency: fallbackCostBasisCurrency,
			costBasis:         fallbackCostBasis,
			filters:           fallbackFilters,
			sourceChargeID:    sourceChargeID,
		}, nil
	}

	return backfilledCreditReissueRouteResult{}, fmt.Errorf("backing transaction group %s does not contain a known cost basis route", group.ID().ID)
}
