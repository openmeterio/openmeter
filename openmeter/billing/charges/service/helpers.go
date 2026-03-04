package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

type handlerByType struct {
	flatFee        func(charge charges.FlatFeeCharge) (charges.FlatFeeCharge, error)
	usageBased     func(charge charges.UsageBasedCharge) (charges.UsageBasedCharge, error)
	creditPurchase func(charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error)
}

func mapChargesByType(in charges.Charges, handlerByType handlerByType) (charges.Charges, error) {
	return slicesx.MapWithErr(in, func(charge charges.Charge) (charges.Charge, error) {
		switch charge.Type() {
		case charges.ChargeTypeFlatFee:
			flatFee, err := charge.AsFlatFeeCharge()
			if err != nil {
				return charges.Charge{}, err
			}

			if handlerByType.flatFee == nil {
				return charges.Charge{}, fmt.Errorf("cannot handle flat fee charge: %w", charges.ErrUnsupported)
			}

			flatFee, err = handlerByType.flatFee(flatFee)
			if err != nil {
				return charges.Charge{}, err
			}

			return charges.NewCharge(flatFee), nil
		case charges.ChargeTypeUsageBased:
			usageBased, err := charge.AsUsageBasedCharge()
			if err != nil {
				return charges.Charge{}, err
			}

			if handlerByType.usageBased == nil {
				return charges.Charge{}, fmt.Errorf("cannot handle usage based charge: %w", charges.ErrUnsupported)
			}

			usageBased, err = handlerByType.usageBased(usageBased)
			if err != nil {
				return charges.Charge{}, err
			}

			return charges.NewCharge(usageBased), nil

		case charges.ChargeTypeCreditPurchase:
			creditPurchase, err := charge.AsCreditPurchaseCharge()
			if err != nil {
				return charges.Charge{}, err
			}

			if handlerByType.creditPurchase == nil {
				return charges.Charge{}, fmt.Errorf("cannot handle credit purchase charge: %w", charges.ErrUnsupported)
			}

			creditPurchase, err = handlerByType.creditPurchase(creditPurchase)
			if err != nil {
				return charges.Charge{}, err
			}

			return charges.NewCharge(creditPurchase), nil

		default:
			return charges.Charge{}, fmt.Errorf("unsupported charge type: %s", charge.Type())
		}
	})
}
