package collector

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func (c *accrualCorrector) planOriginCorrection(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, amount alpacadecimal.Decimal) ([]plannedAction, error) {
	entries := make([]ledger.Entry, 0)

	for _, entry := range source.entries {
		if lo.FromPtr(entry.Provenance().SpendChargeID) != input.ChargeID {
			return nil, fmt.Errorf("collection origin belongs to a different spend charge")
		}

		if entry.Provenance().CollectionOriginID == nil {
			return nil, fmt.Errorf("collection mixes tracked and legacy source entries")
		}

		entries = append(entries, entry)
	}

	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)

	histories := make(map[string]originReferences)
	positionsByOrigin := make(map[string][]correctionPosition)
	originals := make(map[string]*originPair)

	var origins []correctionPosition

	for idx, entry := range entries {
		id := *entry.Provenance().CollectionOriginID

		history, err := c.loadOriginReferences(ctx, input.Namespace, id)
		if err != nil {
			return nil, err
		}

		original, err := history.pairForTransaction(source.transaction.ID().ID)
		if err != nil {
			return nil, err
		}

		positions, err := c.readOriginPositions(ctx, input, id, history, original)
		if err != nil {
			return nil, err
		}

		position := correctionPosition{
			id:    id,
			order: idx,
		}

		for _, p := range positions {
			position.accrued = position.accrued.Add(p.amount())
		}

		origins = append(origins, position)
		histories[id], positionsByOrigin[id], originals[id] = history, positions, original
	}

	selected, err := planCollectionCorrection(collectionCorrectionInput{
		amount:    amount,
		positions: origins,
	})
	if err != nil {
		return nil, fmt.Errorf("exceeds remaining origin balance: %w", err)
	}

	var actions []plannedAction

	for _, selection := range selected {
		resolved, err := c.unwindOrigin(ctx, input, source, histories[selection.id], originals[selection.id], positionsByOrigin[selection.id], selection.amount)
		if err != nil {
			return nil, err
		}

		actions = append(actions, plannedDirectInputs(resolved))
	}

	return actions, nil
}

func (c *accrualCorrector) unwindOrigin(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, history originReferences, original *originPair, positions []correctionPosition, amount alpacadecimal.Decimal) (resolvedCorrectionInputs, error) {
	var out resolvedCorrectionInputs

	selections, err := planCollectionCorrection(collectionCorrectionInput{
		amount:    amount,
		positions: positions,
	})
	if err != nil {
		return out, err
	}

	var backfills []advance.BackfillCorrection

	for _, selection := range selections {
		remainingRecognition := selection.earnings

		for _, pair := range history.pairs {
			if pair.role != originRoleRecognition || lo.FromPtr(pair.positiveEntry.Provenance().SourceChargeID) != selection.id {
				continue
			}

			take := minDecimal(remainingRecognition, pair.remaining)
			if !take.IsPositive() {
				continue
			}

			reversal, err := reverseOriginPair(input, pair, take)
			if err != nil {
				return out, err
			}

			out.inputs = append(out.inputs, reversal)
			remainingRecognition = remainingRecognition.Sub(take)
		}

		if remainingRecognition.IsPositive() {
			return out, fmt.Errorf("earnings position lacks original recognition references")
		}

		if source.advanceReceivableIssueTransaction == nil || selection.id == unknownOriginSource {
			continue
		}

		remainingBacking := selection.amount

		for _, pair := range history.pairs {
			if pair.role != originRoleBacking || lo.FromPtr(pair.positiveEntry.Provenance().SourceChargeID) != selection.id {
				continue
			}

			take := minDecimal(remainingBacking, pair.remaining)
			if !take.IsPositive() {
				continue
			}

			attribution, err := history.attributionForBackfill(pair)
			if err != nil {
				return out, err
			}

			backfills = append(backfills, advance.BackfillCorrection{
				Amount:     take,
				Accrued:    pair.correctionSource(),
				Receivable: attribution.correctionSource(),
			})
			remainingBacking = remainingBacking.Sub(take)
		}

		if remainingBacking.IsPositive() {
			return out, fmt.Errorf("funded position lacks original backing references")
		}
	}

	if source.advanceReceivableIssueTransaction != nil {
		issue, err := history.pairForTransaction(source.advanceReceivableIssueTransaction.ID().ID)
		if err != nil {
			return out, err
		}

		plan, err := c.advance.PlanCorrection(ctx, advance.CorrectionInput{
			CustomerID: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
			At:         input.AllocateAt,
			Amount:     amount,
			Collection: original.correctionSource(),
			Issue:      issue.correctionSource(),
			Backfills:  backfills,
		})
		if err != nil {
			return out, err
		}

		out.inputs = append(out.inputs, plan.Inputs...)
		out.breakagePending = append(out.breakagePending, plan.BreakagePending...)
		return out, nil
	}

	reversal, err := reverseOriginPair(input, original, amount)
	if err != nil {
		return out, err
	}

	out.inputs = append(out.inputs, reversal)

	// Restrict breakage reopening to this exact original source entry.
	plan := transactionCorrectionPlan{
		transaction: originTransactionView{
			Transaction: original.transaction,
			entries:     []ledger.Entry{original.negativeEntry, original.positiveEntry},
		},
		group:  source.group,
		amount: amount,
	}
	inputs, pending, err := c.resolveBreakageReopenInputs(ctx, input, plan)
	if err != nil {
		return out, err
	}

	out.inputs = append(out.inputs, inputs...)
	out.breakagePending = append(out.breakagePending, pending...)

	return out, nil
}

func reverseOriginPair(input CorrectCollectedAccruedInput, pair *originPair, amount alpacadecimal.Decimal) (ledger.TransactionInput, error) {
	if amount.GreaterThan(pair.remaining) {
		return nil, fmt.Errorf("reversal exceeds remaining original entry amount")
	}

	return transactions.ReverseOriginEntryPair(transactions.ReverseOriginEntryPairInput{
		At:            input.AllocateAt,
		Amount:        amount,
		Transaction:   pair.transaction,
		NegativeEntry: pair.negativeEntry,
		PositiveEntry: pair.positiveEntry,
	})
}

type originTransactionView struct {
	ledger.Transaction
	entries []ledger.Entry
}

func (v originTransactionView) Entries() []ledger.Entry {
	return v.entries
}
