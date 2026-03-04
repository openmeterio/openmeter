package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type Config struct {
	Adapter               charges.Adapter
	CreditPurchaseHandler charges.CreditPurchaseHandler
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.CreditPurchaseHandler == nil {
		errs = append(errs, errors.New("credit purchase handler cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (charges.CreditPurchaseOrchestrator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:               config.Adapter,
		creditPurchaseHandler: config.CreditPurchaseHandler,
	}, nil
}

type service struct {
	adapter               charges.Adapter
	creditPurchaseHandler charges.CreditPurchaseHandler
}

func (s *service) PostCreate(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	switch charge.Intent.Settlement.Type() {
	case charges.CreditPurchaseSettlementTypePromotional:
		return s.onPromotionalCreditPurchase(ctx, charge)
	case charges.CreditPurchaseSettlementTypeInvoice:
		return charges.CreditPurchaseCharge{}, fmt.Errorf("invoice credit purchase settlements are not supported: %w", charges.ErrUnsupported)
	case charges.CreditPurchaseSettlementTypeExternal:
		return s.onExternalCreditPurchase(ctx, charge)
	default:
		return charges.CreditPurchaseCharge{}, fmt.Errorf("invalid credit purchase settlement type: %s", charge.Intent.Settlement.Type())
	}
}
