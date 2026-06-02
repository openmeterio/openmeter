package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) UpdateSubscriptionItemID(ctx context.Context, charge charges.Charge, newSubscriptionItemID string) (charges.Charge, error) {
	var errs []error

	if err := charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if newSubscriptionItemID == "" {
		errs = append(errs, errors.New("subscription item ID is required"))
	}

	if err := models.NewNillableGenericValidationError(errors.Join(errs...)); err != nil {
		return charges.Charge{}, err
	}

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		if err != nil {
			return charges.Charge{}, err
		}

		updatedCharge, err := s.flatFeeService.UpdateSubscriptionItemID(ctx, flatFeeCharge, newSubscriptionItemID)
		if err != nil {
			return charges.Charge{}, err
		}

		return charges.NewCharge(updatedCharge), nil
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		if err != nil {
			return charges.Charge{}, err
		}

		updatedCharge, err := s.usageBasedService.UpdateSubscriptionItemID(ctx, usageBasedCharge, newSubscriptionItemID)
		if err != nil {
			return charges.Charge{}, err
		}

		return charges.NewCharge(updatedCharge), nil
	case meta.ChargeTypeCreditPurchase:
		return charges.Charge{}, fmt.Errorf("updating subscription item ID is unsupported for charge type: %s", charge.Type())
	default:
		return charges.Charge{}, fmt.Errorf("unsupported charge type: %s", charge.Type())
	}
}
