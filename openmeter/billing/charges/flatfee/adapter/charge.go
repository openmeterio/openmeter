package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) UpdateCharge(ctx context.Context, charge flatfee.Charge) error {
	if err := charge.ManagedModel.Validate(); err != nil {
		return err
	}

	if err := charge.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		intent := charge.Intent

		var discounts *productcatalog.Discounts
		if intent.PercentageDiscounts != nil {
			discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
		}

		proRating, err := proRatingConfigToDB(intent.ProRating)
		if err != nil {
			return err
		}

		update := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetPaymentTerm(intent.PaymentTerm).
			SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC)).
			SetDiscounts(discounts).
			SetOrClearFeatureID(charge.State.FeatureID).
			SetProRating(proRating).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(charge.State.AmountAfterProration)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          intent.Intent,
			Status:          charge.Status,
			AdvanceAfter:    meta.NormalizeOptionalTimestamp(charge.State.AdvanceAfter),
		})
		if err != nil {
			return err
		}

		_, err = update.Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (a *adapter) DeleteCharge(ctx context.Context, charge flatfee.Charge) error {
	if err := charge.ManagedModel.Validate(); err != nil {
		return err
	}

	if err := charge.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		update := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace))

		charge.DeletedAt = lo.ToPtr(clock.Now())
		charge.Status = meta.ChargeStatusDeleted

		update, err := chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          charge.Intent.Intent,
			Status:          charge.Status,
		})
		if err != nil {
			return err
		}

		if _, err := update.Save(ctx); err != nil {
			return err
		}

		return tx.metaAdapter.DeleteRegisteredCharge(ctx, charge.GetChargeID())
	})
}

func (a *adapter) CreateCharges(ctx context.Context, in flatfee.CreateChargesInput) ([]flatfee.Charge, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]flatfee.Charge, error) {
		creates, err := slicesx.MapWithErr(in.Intents, func(intent flatfee.IntentWithInitialStatus) (*db.ChargeFlatFeeCreate, error) {
			return tx.buildCreateFlatFeeCharge(in.Namespace, intent)
		})
		if err != nil {
			return nil, err
		}

		entities, err := tx.db.ChargeFlatFee.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		// Let's reserve the charge IDs
		err = tx.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{
			Namespace: in.Namespace,
			Type:      meta.ChargeTypeFlatFee,
			Charges: lo.Map(entities, func(entity *db.ChargeFlatFee, idx int) meta.IDWithUniqueReferenceID {
				return meta.IDWithUniqueReferenceID{
					ID:                entity.ID,
					UniqueReferenceID: entity.UniqueReferenceID,
				}
			}),
		})
		if err != nil {
			return nil, err
		}

		out := make([]flatfee.Charge, 0, len(entities))
		for _, entity := range entities {
			charge, err := MapChargeFlatFeeFromDB(entity, meta.ExpandNone)
			if err != nil {
				return nil, err
			}
			out = append(out, charge)
		}

		return out, nil
	})
}

func (a *adapter) GetByIDs(ctx context.Context, input flatfee.GetByIDsInput) ([]flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]flatfee.Charge, error) {
		query := tx.db.ChargeFlatFee.Query().
			Where(dbchargeflatfee.Namespace(input.Namespace)).
			Where(dbchargeflatfee.IDIn(input.IDs...))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query)
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, entities)
		if err != nil {
			return nil, err
		}

		out, err := slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeFlatFee) (flatfee.Charge, error) {
			return MapChargeFlatFeeFromDB(entity, input.Expands)
		})
		if err != nil {
			return nil, err
		}

		if input.Expands.Has(meta.ExpandRealizations) {
			if err := attachActiveLineageSegmentsToFlatFeeCharges(ctx, tx.db, input.Namespace, out); err != nil {
				return nil, err
			}
		}

		return out, nil
	})
}

func (a *adapter) GetByID(ctx context.Context, input flatfee.GetByIDInput) (flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return flatfee.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		query := tx.db.ChargeFlatFee.Query().
			Where(dbchargeflatfee.Namespace(input.ChargeID.Namespace)).
			Where(dbchargeflatfee.ID(input.ChargeID.ID))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query)
		}

		entity, err := query.First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return flatfee.Charge{}, models.NewGenericNotFoundError(fmt.Errorf("flat fee charge [id=%s] not found", input.ChargeID))
			}

			return flatfee.Charge{}, fmt.Errorf("querying flat fee charge [id=%s]: %w", input.ChargeID, err)
		}

		charge, err := MapChargeFlatFeeFromDB(entity, input.Expands)
		if err != nil {
			return flatfee.Charge{}, err
		}

		if input.Expands.Has(meta.ExpandRealizations) {
			charges := []flatfee.Charge{charge}
			if err := attachActiveLineageSegmentsToFlatFeeCharges(ctx, tx.db, input.ChargeID.Namespace, charges); err != nil {
				return flatfee.Charge{}, err
			}
			charge = charges[0]
		}

		return charge, nil
	})
}

func expandRealizations(query *db.ChargeFlatFeeQuery) *db.ChargeFlatFeeQuery {
	return query.WithCreditAllocations().
		WithInvoicedUsage().
		WithPayment()
}

func attachActiveLineageSegmentsToFlatFeeCharges(ctx context.Context, dbClient *db.Client, namespace string, charges []flatfee.Charge) error {
	realizationIDs := make([]string, 0)
	for _, charge := range charges {
		for _, realization := range charge.State.CreditRealizations {
			realizationIDs = append(realizationIDs, realization.ID)
		}
	}

	segmentsByRealizationID, err := chargesadapter.LoadActiveLineageSegments(ctx, dbClient, namespace, realizationIDs)
	if err != nil {
		return fmt.Errorf("load active lineage segments for flat fee charges: %w", err)
	}

	for chargeIdx := range charges {
		for realizationIdx := range charges[chargeIdx].State.CreditRealizations {
			realization := &charges[chargeIdx].State.CreditRealizations[realizationIdx]
			realization.ActiveLineageSegments = segmentsByRealizationID[realization.ID]
		}
	}

	return nil
}

func (a *adapter) buildCreateFlatFeeCharge(ns string, intent flatfee.IntentWithInitialStatus) (*db.ChargeFlatFeeCreate, error) {
	var discounts *productcatalog.Discounts
	if intent.PercentageDiscounts != nil {
		discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
	}

	proRating, err := proRatingConfigToDB(intent.ProRating)
	if err != nil {
		return nil, err
	}

	create := a.db.ChargeFlatFee.Create().
		SetNamespace(ns).
		SetPaymentTerm(intent.PaymentTerm).
		SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC)).
		SetSettlementMode(intent.SettlementMode).
		SetNillableFeatureID(intent.FeatureID).
		SetNillableFeatureKey(lo.EmptyableToPtr(intent.FeatureKey)).
		SetProRating(proRating).
		SetAmountBeforeProration(intent.AmountBeforeProration).
		SetAmountAfterProration(intent.AmountAfterProration)

	if discounts != nil {
		create = create.SetDiscounts(discounts)
	}

	create, err = chargemeta.Create[*db.ChargeFlatFeeCreate](create, chargemeta.CreateInput{
		Namespace: ns,
		Intent:    intent.Intent.Intent,
		Status:    intent.InitialStatus,
	})
	if err != nil {
		return nil, err
	}

	return create, nil
}
