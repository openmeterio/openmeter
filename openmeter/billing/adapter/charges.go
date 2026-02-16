package billingadapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) SetChargeIDsOnInvoiceLines(ctx context.Context, input billing.SetChargeIDsOnInvoiceLinesInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		lineIDsByChargeID := make(map[string][]string)
		for lineID, chargeID := range input.LineIDToChargeID {
			lineIDsByChargeID[chargeID] = append(lineIDsByChargeID[chargeID], lineID)
		}

		for chargeID, lineIDs := range lineIDsByChargeID {
			update := tx.db.BillingInvoiceLine.Update().
				Where(billinginvoiceline.Namespace(input.Namespace)).
				Where(billinginvoiceline.IDIn(lineIDs...)).
				SetChargeID(chargeID)
			update.Mutation().ResetUpdatedAt()
			err := update.Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (a *adapter) SetChargeIDsOnSplitlineGroups(ctx context.Context, input billing.SetChargeIDsOnSplitlineGroupsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		groupIDsByChargeID := make(map[string][]string)
		for groupID, chargeID := range input.GroupIDToChargeID {
			groupIDsByChargeID[chargeID] = append(groupIDsByChargeID[chargeID], groupID)
		}

		for chargeID, groupIDs := range groupIDsByChargeID {
			update := tx.db.BillingInvoiceSplitLineGroup.Update().
				Where(billinginvoicesplitlinegroup.Namespace(input.Namespace)).
				Where(billinginvoicesplitlinegroup.IDIn(groupIDs...)).
				SetChargeID(chargeID)
			update.Mutation().ResetUpdatedAt()
			err := update.Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
