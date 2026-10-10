package correction

import (
	"context"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/models"
)

type correctionPlan struct {
	inputs          []ledger.TransactionInput
	breakagePending []breakage.PendingRecord
	realizations    creditrealization.CreateCorrectionInputs
}

func (p *correctionPlan) append(other correctionPlan) {
	p.inputs = append(p.inputs, other.inputs...)
	p.breakagePending = append(p.breakagePending, other.breakagePending...)
	p.realizations = append(p.realizations, other.realizations...)
}

func (c *Corrector) prepareCorrections(ctx context.Context, input Input) (correctionPlan, error) {
	var plan correctionPlan

	legacy := legacyCorrectionBatch{corrector: c, input: input}

	for _, correction := range input.Corrections {
		group, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{
			Namespace: input.Namespace,
			ID:        correction.Allocation.LedgerTransaction.TransactionGroupID,
		})
		if err != nil {
			return correctionPlan{}, fmt.Errorf("get original transaction group %s: %w", correction.Allocation.LedgerTransaction.TransactionGroupID, err)
		}

		// SortHint identifies the allocation's original FBO subaccount within the group.
		source, err := c.collectedSourceBySortHint(group, correction.Allocation.SortHint)
		if err != nil {
			return correctionPlan{}, err
		}

		originTracked := slices.ContainsFunc(source.transaction.Entries(), func(entry ledger.Entry) bool {
			return entry.Provenance().CollectionOriginID != nil
		})

		var collectionPlan correctionPlan

		if originTracked {
			collectionPlan, err = c.prepareOriginCorrection(ctx, input, correction, source)
		} else {
			collectionPlan, err = legacy.prepare(ctx, correction, source)
		}

		if err != nil {
			return correctionPlan{}, err
		}

		plan.append(collectionPlan)
	}

	resolvedLegacy, err := legacy.resolve(ctx)
	if err != nil {
		return correctionPlan{}, err
	}

	plan.append(resolvedLegacy)

	return plan, nil
}
