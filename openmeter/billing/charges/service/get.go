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
)

func (s *service) GetByID(ctx context.Context, input charges.GetByIDInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	res, err := s.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: input.ChargeID.Namespace,
		IDs:       []string{input.ChargeID.ID},
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
		chargesItems, err := s.adapter.GetByIDs(ctx, charges.GetByIDsInput{
			Namespace: input.Namespace,
			IDs:       input.IDs,
		})
		if err != nil {
			return nil, err
		}

		return s.expandChargesWithTypes(ctx, input.Namespace, chargesItems, input.Expands)
	})
}

func (s *service) GetCurrentTotals(ctx context.Context, input usagebased.GetCurrentTotalsInput) (usagebased.GetCurrentTotalsResult, error) {
	return s.usageBasedService.GetCurrentTotals(ctx, input)
}

// expandChargesWithTypes fetches the charges by type and expands them with the given expands.
func (s *service) expandChargesWithTypes(ctx context.Context, namespace string, chargesItems charges.ChargeSearchItems, expands meta.Expands) (charges.Charges, error) {
	chargesByType := lo.GroupByMap(chargesItems, func(chargeMeta charges.ChargeSearchItem) (meta.ChargeType, string) {
		return chargeMeta.Type, chargeMeta.ID.ID
	})

	// Let's validate the type support
	referencedTypes := lo.Keys(chargesByType)
	for _, refType := range referencedTypes {
		if err := refType.Validate(); err != nil {
			return nil, err
		}
	}

	usageBased, err := s.usageBasedService.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: namespace,
		IDs:       chargesByType[meta.ChargeTypeUsageBased],
		Expands:   expands,
	})
	if err != nil {
		return nil, err
	}

	flatFees, err := s.flatFeeService.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: namespace,
		IDs:       chargesByType[meta.ChargeTypeFlatFee],
		Expands:   expands,
	})
	if err != nil {
		return nil, err
	}

	creditPurchases, err := s.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
		Namespace: namespace,
		IDs:       chargesByType[meta.ChargeTypeCreditPurchase],
		Expands:   expands,
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
		namespace,
		lo.Map(
			chargesItems,
			func(charge charges.ChargeSearchItem, _ int) string {
				return charge.ID.ID
			},
		), out)
}
