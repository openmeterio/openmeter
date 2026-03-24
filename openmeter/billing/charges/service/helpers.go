package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

type chargesByTypeResult struct {
	flatFees       []flatfee.Charge
	usageBased     []usagebased.Charge
	creditPurchase []creditpurchase.Charge
}

func chargesByType(in charges.Charges) (chargesByTypeResult, error) {
	result := chargesByTypeResult{
		flatFees:       make([]flatfee.Charge, 0, len(in)),
		usageBased:     make([]usagebased.Charge, 0, len(in)),
		creditPurchase: make([]creditpurchase.Charge, 0, len(in)),
	}

	for _, charge := range in {
		switch charge.Type() {
		case meta.ChargeTypeFlatFee:
			flatFee, err := charge.AsFlatFeeCharge()
			if err != nil {
				return chargesByTypeResult{}, err
			}

			result.flatFees = append(result.flatFees, flatFee)
		case meta.ChargeTypeUsageBased:
			ub, err := charge.AsUsageBasedCharge()
			if err != nil {
				return chargesByTypeResult{}, err
			}

			result.usageBased = append(result.usageBased, ub)
		case meta.ChargeTypeCreditPurchase:
			creditPurchase, err := charge.AsCreditPurchaseCharge()
			if err != nil {
				return chargesByTypeResult{}, err
			}

			result.creditPurchase = append(result.creditPurchase, creditPurchase)
		default:
			return chargesByTypeResult{}, fmt.Errorf("unsupported charge type: %s", charge.Type())
		}
	}

	return result, nil
}
