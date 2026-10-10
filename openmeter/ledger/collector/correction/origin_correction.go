package correction

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Balances determine the remaining amounts; history supplies immutable reversal references.
type originCorrectionSource struct {
	original *originPair
	history  originReferences
	balances []correctionPosition
}

func (c *Corrector) prepareOriginCorrection(ctx context.Context, input Input, correction creditrealization.CorrectionRequestItem, source collectedSource) (correctionPlan, error) {
	entries := slices.Clone(source.entries)

	for _, entry := range entries {
		if lo.FromPtr(entry.Provenance().SpendChargeID) != input.ChargeID {
			return correctionPlan{}, fmt.Errorf("collection origin belongs to a different spend charge")
		}

		if entry.Provenance().CollectionOriginID == nil {
			return correctionPlan{}, fmt.Errorf("collection mixes tracked and legacy source entries")
		}
	}

	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)

	byOrigin := make(map[string]originCorrectionSource)

	var origins []correctionPosition

	for idx, entry := range entries {
		id := *entry.Provenance().CollectionOriginID

		history, err := c.loadOriginReferences(ctx, input.Namespace, id)
		if err != nil {
			return correctionPlan{}, err
		}

		original, err := history.pairForTransaction(source.transaction.ID().ID)
		if err != nil {
			return correctionPlan{}, err
		}

		positions, err := c.readOriginPositions(ctx, input, id, history, original)
		if err != nil {
			return correctionPlan{}, err
		}

		position := correctionPosition{
			id:    id,
			order: idx,
		}

		for _, p := range positions {
			position.accrued = position.accrued.Add(p.accrued)
			position.earnings = position.earnings.Add(p.earnings)
			position.receivable = position.receivable.Add(p.receivable)
		}

		origins = append(origins, position)
		byOrigin[id] = originCorrectionSource{
			original: original,
			history:  history,
			balances: positions,
		}
	}

	selected, err := planCollectionCorrection(collectionCorrectionInput{
		amount:    correction.Amount.Abs(),
		positions: origins,
	})
	if err != nil {
		return correctionPlan{}, fmt.Errorf("exceeds remaining origin balance: %w", err)
	}

	var plan correctionPlan

	for _, selection := range selected {
		resolved, err := c.correctOrigin(ctx, input, source, byOrigin[selection.id], selection.amount)
		if err != nil {
			return correctionPlan{}, err
		}

		plan.append(resolved)
	}

	plan.realizations = append(plan.realizations, creditrealization.CreateCorrectionInput{
		Annotations:           models.Annotations{ledger.AnnotationOriginTracked: true},
		Amount:                correction.Amount,
		CorrectsRealizationID: correction.Allocation.ID,
	})

	return plan, nil
}

func (c *Corrector) correctOrigin(ctx context.Context, input Input, source collectedSource, origin originCorrectionSource, amount alpacadecimal.Decimal) (correctionPlan, error) {
	var out correctionPlan

	selections, err := planCollectionCorrection(collectionCorrectionInput{
		amount:    amount,
		positions: origin.balances,
	})
	if err != nil {
		return out, err
	}

	out.inputs, err = origin.history.reverseEarnings(input.AllocateAt, selections)
	if err != nil {
		return out, err
	}

	if source.advanceReceivableIssueTransaction != nil {
		backfills, err := origin.history.backfillCorrections(selections)
		if err != nil {
			return out, err
		}

		issue, err := origin.history.pairForTransaction(source.advanceReceivableIssueTransaction.ID().ID)
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
			Collection: origin.original.correctionSource(),
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

	reversal, err := origin.original.reverse(input.AllocateAt, amount)
	if err != nil {
		return out, err
	}

	out.inputs = append(out.inputs, reversal)

	// Restrict breakage reopening to this exact original source entry.
	plan := transactions.CorrectionInput{
		OriginalTransaction: origin.original.transaction,
		SourceEntryAmounts:  map[string]alpacadecimal.Decimal{origin.original.negativeEntry.ID().ID: amount},
		OriginalGroup:       source.group,
		Amount:              amount,
		At:                  input.AllocateAt,
	}
	inputs, pending, err := c.resolveBreakageReopenInputs(ctx, input, plan)
	if err != nil {
		return out, err
	}

	out.inputs = append(out.inputs, inputs...)
	out.breakagePending = append(out.breakagePending, pending...)

	return out, nil
}

func (h originReferences) reverseEarnings(at time.Time, selections []correctionSelection) ([]ledger.TransactionInput, error) {
	var inputs []ledger.TransactionInput

	for _, selection := range selections {
		remaining := selection.earnings

		for _, pair := range h.pairs {
			if pair.role != originRoleRecognition || lo.FromPtr(pair.positiveEntry.Provenance().SourceChargeID) != selection.id {
				continue
			}

			take := minDecimal(remaining, pair.remaining)
			if !take.IsPositive() {
				continue
			}

			reversal, err := pair.reverse(at, take)
			if err != nil {
				return nil, err
			}

			inputs = append(inputs, reversal)
			remaining = remaining.Sub(take)
		}

		if remaining.IsPositive() {
			return nil, fmt.Errorf("earnings position lacks original recognition references")
		}
	}

	return inputs, nil
}

func (h originReferences) backfillCorrections(selections []correctionSelection) ([]advance.BackfillCorrection, error) {
	var backfills []advance.BackfillCorrection

	for _, selection := range selections {
		if selection.id == unknownOriginSource {
			continue
		}

		remaining := selection.amount

		for _, pair := range h.pairs {
			if pair.role != originRoleBacking || lo.FromPtr(pair.positiveEntry.Provenance().SourceChargeID) != selection.id {
				continue
			}

			take := minDecimal(remaining, pair.remaining)
			if !take.IsPositive() {
				continue
			}

			attribution, err := h.attributionForBackfill(pair)
			if err != nil {
				return nil, err
			}

			backfills = append(backfills, advance.BackfillCorrection{
				Amount:     take,
				Accrued:    pair.correctionSource(),
				Receivable: attribution.correctionSource(),
			})
			remaining = remaining.Sub(take)
		}

		if remaining.IsPositive() {
			return nil, fmt.Errorf("funded position lacks original backing references")
		}
	}

	return backfills, nil
}
