package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func (s *service) PlanCorrection(ctx context.Context, input advance.CorrectionInput) (advance.CorrectionPlan, error) {
	var out advance.CorrectionPlan

	if err := input.Validate(); err != nil {
		return out, err
	}

	for _, backfill := range input.Backfills {
		planned, err := s.correctBackfill(ctx, input, backfill)
		if err != nil {
			return out, err
		}

		out.Inputs = append(out.Inputs, planned.Inputs...)
		out.BreakagePending = append(out.BreakagePending, planned.BreakagePending...)
	}

	for _, source := range []advance.CorrectionSource{input.Collection, input.Issue} {
		reversal, err := source.Reverse(input.At, input.Amount)
		if err != nil {
			return out, err
		}

		out.Inputs = append(out.Inputs, reversal)
	}

	return out, nil
}

func (s *service) correctBackfill(ctx context.Context, input advance.CorrectionInput, backfill advance.BackfillCorrection) (advance.CorrectionPlan, error) {
	var out advance.CorrectionPlan

	for _, pair := range []advance.CorrectionSource{backfill.Accrued, backfill.Receivable} {
		reversal, err := pair.Reverse(input.At, backfill.Amount)
		if err != nil {
			return out, err
		}

		out.Inputs = append(out.Inputs, reversal)
	}

	group, err := s.ledger.GetTransactionGroup(ctx, backfill.Accrued.Transaction.GroupID())
	if err != nil {
		return out, err
	}

	releases, err := s.breakage.ListReleases(ctx, breakage.ListReleasesInput{
		CustomerID:               input.CustomerID,
		SourceTransactionGroupID: []string{group.ID().ID},
		ReleaseSourceKind:        []breakage.SourceKind{breakage.SourceKindAdvanceBackfill},
	})
	if err != nil {
		return out, err
	}

	matching := make(map[string]bool)

	for _, tx := range group.Transactions() {
		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO &&
				lo.FromPtr(entry.Provenance().CollectionOriginID) == lo.FromPtr(backfill.Accrued.PositiveEntry.Provenance().CollectionOriginID) &&
				lo.FromPtr(entry.Provenance().SourceChargeID) == lo.FromPtr(backfill.Accrued.PositiveEntry.Provenance().SourceChargeID) {
				matching[tx.ID().ID] = true
			}
		}
	}

	remaining := backfill.Amount

	for _, release := range releases {
		if !matching[release.BreakageTransactionID] || !remaining.IsPositive() {
			continue
		}

		take := minDecimal(remaining, release.OpenAmount)
		if !take.IsPositive() {
			continue
		}

		reopened, pending, err := s.breakage.ReopenRelease(ctx, breakage.ReopenReleaseInput{
			Release:            release,
			Amount:             take,
			SourceKind:         breakage.SourceKindUsageCorrection,
			SourceChargeID:     backfill.Accrued.PositiveEntry.Provenance().SourceChargeID,
			SpendChargeID:      backfill.Accrued.PositiveEntry.Provenance().SpendChargeID,
			CollectionOriginID: backfill.Accrued.PositiveEntry.Provenance().CollectionOriginID,
		})
		if err != nil {
			return out, err
		}

		out.Inputs = append(out.Inputs, reopened)
		out.BreakagePending = append(out.BreakagePending, pending)
		remaining = remaining.Sub(take)
	}

	// The attributed receivable retains the purchase's feature restrictions.
	// Priority is preserved explicitly even when a purchase was fully backfilled
	// and therefore never wrote an ordinary FBO issuance entry.
	route := backfill.Receivable.NegativeEntry.PostingAddress().Route().Route()

	priority, ok := group.Annotations().GetInt(ledger.AnnotationBackfillCreditPriority)
	if !ok {
		return out, fmt.Errorf("origin backfill is missing purchased credit priority")
	}

	reissued, err := transactions.ResolveTransactions(ctx, s.resolverDependencies(), transactions.ResolutionScope{
		CustomerID: input.CustomerID,
		Namespace:  input.CustomerID.Namespace,
	}, transactions.IssueCustomerReceivableTemplate{
		At:                input.At,
		Amount:            backfill.Amount,
		Currency:          route.Currency,
		CostBasisCurrency: route.CostBasisCurrency,
		CostBasis:         route.CostBasis,
		Filters:           route.Filters,
		CreditPriority:    &priority,
		SourceChargeID:    backfill.Receivable.NegativeEntry.Provenance().SourceChargeID,
	})
	if err != nil {
		return out, err
	}

	for _, tx := range reissued {
		out.Inputs = append(out.Inputs, transactions.WithAnnotations(tx, ledger.TransactionAnnotations(
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}), ledger.TransactionDirectionCorrection)))
	}

	return out, nil
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
