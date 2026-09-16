package chargemeta

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionReferenceUpdate[P ~func(*sql.Selector), U any] interface {
	Where(...P) U
	SetSubscriptionPhaseID(string) U
	SetSubscriptionItemID(string) U
	Save(context.Context) (int, error)
}

func UpdateSubscriptionReference[
	P ~func(*sql.Selector),
	U SubscriptionReferenceUpdate[P, U],
](
	ctx context.Context,
	update U,
	input meta.UpdateSubscriptionReferenceInput,
) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate input: %w", err)
	}

	updatedCharges, err := update.
		Where(P(func(selector *sql.Selector) {
			selector.Where(sql.And(
				sql.EQ(selector.C("id"), input.ChargeID.ID),
				sql.EQ(selector.C("namespace"), input.ChargeID.Namespace),
				sql.EQ(selector.C("subscription_id"), input.Target.SubscriptionID),
			))
		})).
		SetSubscriptionPhaseID(input.Target.PhaseID).
		SetSubscriptionItemID(input.Target.ItemID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update subscription reference: %w", err)
	}
	if updatedCharges != 1 {
		return models.NewGenericPreConditionFailedError(fmt.Errorf(
			"expected to update one charge, updated %d",
			updatedCharges,
		))
	}

	return nil
}
