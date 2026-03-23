package service

import (
	"context"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) GetByID(ctx context.Context, input charges.GetByIDInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	res, err := s.GetByIDs(ctx, charges.GetByIDsInput{
		ChargeIDs: meta.ChargeIDs{input.ChargeID},
		Expands:   input.Expands,
	})
	if err != nil {
		return charges.Charge{}, err
	}

	if len(res) == 0 {
		return charges.Charge{}, charges.NewChargeNotFoundError(input.ChargeID.Namespace, input.ChargeID.ID)
	}

	return res[0], nil
}

func (s *service) GetByIDs(ctx context.Context, input charges.GetByIDsInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		chargesWithTypes, err := s.adapter.GetTypesByIDs(ctx, input.ChargeIDs)
		if err != nil {
			return nil, err
		}

		return s.expandChargesWithTypes(ctx, chargesWithTypes, input.Expands)
	})
}

// expandChargesWithTypes fetches the charges by type and expands them with the given expands.
func (s *service) expandChargesWithTypes(ctx context.Context, chargesWithTypes []charges.ChargeWithType, expands meta.Expands) (charges.Charges, error) {
	chargesByType := lo.GroupByMap(chargesWithTypes, func(chargeMeta charges.ChargeWithType) (meta.ChargeType, meta.ChargeID) {
		return chargeMeta.Type, chargeMeta.ChargeID
	})

	// Let's validate the type support
	referencedTypes := lo.Keys(chargesByType)
	for _, refType := range referencedTypes {
		if err := refType.Validate(); err != nil {
			return nil, err
		}
	}

	usageBased, err := s.usageBasedService.GetByIDs(ctx, usagebased.GetByIDsInput{
		IDs:     chargesByType[meta.ChargeTypeUsageBased],
		Expands: expands,
	})
	if err != nil {
		return nil, err
	}

	flatFees, err := s.flatFeeService.GetByIDs(ctx, flatfee.GetByIDsInput{
		IDs:     chargesByType[meta.ChargeTypeFlatFee],
		Expands: expands,
	})
	if err != nil {
		return nil, err
	}

	creditPurchases, err := s.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
		IDs:     chargesByType[meta.ChargeTypeCreditPurchase],
		Expands: expands,
	})
	if err != nil {
		return nil, err
	}

	out := slices.Concat(
		lo.Map(usageBased, func(charge usagebased.Charge, _ int) charges.Charge {
			return charges.NewCharge(charge)
		}),
		lo.Map(flatFees, func(charge flatfee.Charge, _ int) charges.Charge {
			return charges.NewCharge(charge)
		}),
		lo.Map(creditPurchases, func(charge creditpurchase.Charge, _ int) charges.Charge {
			return charges.NewCharge(charge)
		}),
	)

	return entutils.InIDOrder(
		lo.Map(
			chargesWithTypes,
			func(charge charges.ChargeWithType, _ int) models.NamespacedID {
				return models.NamespacedID{
					Namespace: charge.ChargeID.Namespace,
					ID:        charge.ChargeID.ID,
				}
			},
		), out)
}
