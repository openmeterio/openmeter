package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) AdvanceCreditOnlyCharges(ctx context.Context, input charges.AdvanceCreditOnlyChargesInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if len(input.ChargeIDs) == 0 {
		return nil, nil
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		res, err := s.adapter.GetChargesByIDs(ctx, charges.GetChargesByIDsInput{
			Namespace: input.Namespace,
			ChargeIDs: input.ChargeIDs,
			Expands:   charges.Expands{charges.ExpandRealizations},
		})
		if err != nil {
			return nil, err
		}

		return mapChargesByType(res, handlerByType{
			flatFee: func(charge charges.FlatFeeCharge) (charges.FlatFeeCharge, error) {
				if charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
					return charge, nil
				}

				return s.flatFeeOrchestrator.AdvanceCreditOnlyCharge(ctx, charge)
			},
		})
	})
}
