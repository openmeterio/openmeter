package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	chargedb "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) RegisterCharges(ctx context.Context, in meta.RegisterChargesInput) error {
	if err := in.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		creates, err := slicesx.MapWithErr(in.Charges, func(charge meta.IDWithUniqueReferenceID) (*db.ChargeCreate, error) {
			create := tx.db.Charge.Create().
				SetNamespace(in.Namespace).
				SetType(in.Type).
				SetID(charge.ID).
				SetNillableUniqueReferenceID(charge.UniqueReferenceID).
				SetCreatedAt(clock.Now())

			switch in.Type {
			case meta.ChargeTypeFlatFee:
				create = create.SetChargeFlatFeeID(charge.ID)
			case meta.ChargeTypeUsageBased:
				create = create.SetChargeUsageBasedID(charge.ID)
			case meta.ChargeTypeCreditPurchase:
				create = create.SetChargeCreditPurchaseID(charge.ID)
			default:
				return nil, fmt.Errorf("unknown charge type: %s", in.Type)
			}

			return create, nil
		})
		if err != nil {
			return err
		}

		_, err = tx.db.Charge.CreateBulk(creates...).Save(ctx)
		return err
	})
}

func (a *adapter) DeleteRegisteredCharge(ctx context.Context, in meta.DeleteRegisteredChargeInput) error {
	if err := in.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.Charge.UpdateOneID(in.ID).
			Where(
				chargedb.DeletedAtIsNil(),
				chargedb.Namespace(in.Namespace),
			).SetDeletedAt(clock.Now()).Exec(ctx)
	})
}
