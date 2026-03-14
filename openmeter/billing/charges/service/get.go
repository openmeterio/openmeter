package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) GetByID(ctx context.Context, input charges.GetByIDInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	res, err := s.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: input.ChargeID.Namespace,
		ChargeIDs: []string{input.ChargeID.ID},
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
		// Let's fetch the metas so that we know the charge types
		chargeMetas, err := s.metaAdapter.GetByIDs(ctx, meta.GetByIDsInput{
			Namespace: input.Namespace,
			ChargeIDs: input.ChargeIDs,
		})
		if err != nil {
			return nil, err
		}

		if len(chargeMetas) == 0 {
			return nil, nil
		}

		chargesWithIndex := lo.Map(chargeMetas, func(chargeMeta meta.Charge, idx int) charges.WithIndex[meta.Charge] {
			return charges.WithIndex[meta.Charge]{
				Index: idx,
				Value: chargeMeta,
			}
		})

		chargesByType := lo.GroupBy(chargesWithIndex, func(chargeMeta charges.WithIndex[meta.Charge]) meta.ChargeType {
			return chargeMeta.Value.Type
		})

		// Let's validate the type support
		referencedTypes := lo.Keys(chargesByType)
		for _, refType := range referencedTypes {
			if err := refType.Validate(); err != nil {
				return nil, err
			}

			if refType == meta.ChargeTypeUsageBased {
				return nil, fmt.Errorf("usage based charges are not supported: %w", meta.ErrUnsupported)
			}
		}

		out := make(charges.Charges, len(chargesWithIndex))
		nrFetched := 0

		// Let's fetch flat fees
		flatFees, err := s.flatFeeService.GetByIDs(ctx, flatfee.GetByIDsInput{
			Namespace: input.Namespace,
			Charges: lo.Map(chargesByType[meta.ChargeTypeFlatFee], func(chargeMeta charges.WithIndex[meta.Charge], _ int) meta.Charge {
				return chargeMeta.Value
			}),
			Expands: input.Expands,
		})
		if err != nil {
			return nil, err
		}

		for i, flatFee := range flatFees {
			targetIndex := chargesByType[meta.ChargeTypeFlatFee][i].Index
			out[targetIndex] = charges.NewCharge(flatFee)
			nrFetched++
		}

		// Let's fetch credit purchases
		creditPurchases, err := s.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
			Namespace: input.Namespace,
			Charges: lo.Map(chargesByType[meta.ChargeTypeCreditPurchase], func(chargeMeta charges.WithIndex[meta.Charge], _ int) meta.Charge {
				return chargeMeta.Value
			}),
			Expands: input.Expands,
		})
		if err != nil {
			return nil, err
		}

		for i, creditPurchase := range creditPurchases {
			targetIndex := chargesByType[meta.ChargeTypeCreditPurchase][i].Index
			out[targetIndex] = charges.NewCharge(creditPurchase)
			nrFetched++
		}

		if nrFetched != len(out) {
			return nil, fmt.Errorf("expected to fetch %d charges, got %d", len(out), nrFetched)
		}

		return out, nil
	})
}
