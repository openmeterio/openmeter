package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

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

type InvocableCharge interface {
	GetChargeID() meta.ChargeID
	TriggerPatch(ctx context.Context, patch meta.Patch) (*charges.Charge, error)
}

func (s *service) newInvocableCharges(si charges.ChargeSearchItems) (map[string]InvocableCharge, error) {
	result := make(map[string]InvocableCharge, len(si))
	for _, si := range si {
		if _, exists := result[si.ID.ID]; exists {
			return nil, fmt.Errorf("duplicated charge ID: %s", si.ID.ID)
		}

		switch si.Type {
		case meta.ChargeTypeFlatFee:
			result[si.ID.ID] = &flatFeeInvocableCharge{
				chargeID:       si.ID,
				flatFeeService: s.flatFeeService,
			}
		case meta.ChargeTypeUsageBased:
			result[si.ID.ID] = &usageBasedInvocableCharge{
				chargeID:          si.ID,
				usageBasedService: s.usageBasedService,
			}
		default:
			return nil, fmt.Errorf("unsupported charge type: %s", si.Type)
		}
	}
	return result, nil
}

var _ InvocableCharge = (*flatFeeInvocableCharge)(nil)

type flatFeeInvocableCharge struct {
	chargeID       meta.ChargeID
	flatFeeService flatfee.Service
}

func (c *flatFeeInvocableCharge) TriggerPatch(ctx context.Context, patch meta.Patch) (*charges.Charge, error) {
	res, err := c.flatFeeService.TriggerPatch(ctx, c.chargeID, patch)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	return lo.ToPtr(charges.NewCharge(*res)), nil
}

func (c *flatFeeInvocableCharge) GetChargeID() meta.ChargeID {
	return c.chargeID
}

var _ InvocableCharge = (*usageBasedInvocableCharge)(nil)

type usageBasedInvocableCharge struct {
	chargeID          meta.ChargeID
	usageBasedService usagebased.Service
}

func (c *usageBasedInvocableCharge) TriggerPatch(ctx context.Context, patch meta.Patch) (*charges.Charge, error) {
	res, err := c.usageBasedService.TriggerPatch(ctx, c.chargeID, patch)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	return lo.ToPtr(charges.NewCharge(*res)), nil
}

func (c *usageBasedInvocableCharge) GetChargeID() meta.ChargeID {
	return c.chargeID
}
