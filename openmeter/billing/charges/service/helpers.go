package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type chargesByTypeResult struct {
	flatFees       []charges.FlatFeeCharge
	usageBased     []charges.UsageBasedCharge
	creditPurchase []charges.CreditPurchaseCharge
}

func chargesByType(in charges.Charges) (chargesByTypeResult, error) {
	result := chargesByTypeResult{
		flatFees:       make([]charges.FlatFeeCharge, 0, len(in)),
		usageBased:     make([]charges.UsageBasedCharge, 0, len(in)),
		creditPurchase: make([]charges.CreditPurchaseCharge, 0, len(in)),
	}

	for _, charge := range in {
		switch charge.Type() {
		case charges.ChargeTypeFlatFee:
			flatFee, err := charge.AsFlatFeeCharge()
			if err != nil {
				return chargesByTypeResult{}, err
			}

			result.flatFees = append(result.flatFees, flatFee)
		case charges.ChargeTypeUsageBased:
			usageBased, err := charge.AsUsageBasedCharge()
			if err != nil {
				return chargesByTypeResult{}, err
			}

			result.usageBased = append(result.usageBased, usageBased)
		case charges.ChargeTypeCreditPurchase:
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
