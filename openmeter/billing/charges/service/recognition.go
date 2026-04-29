package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func (s *service) recognizeCustomerEarnings(ctx context.Context, customerID customer.CustomerID, currencies ...currencyx.Code) error {
	for _, currency := range uniqueCurrencies(currencies) {
		if _, err := s.recognizerService.RecognizeEarnings(ctx, recognizer.RecognizeEarningsInput{
			CustomerID: customerID,
			At:         clock.Now(),
			Currency:   currency,
		}); err != nil {
			return fmt.Errorf("recognize earnings for currency %s: %w", currency, err)
		}
	}

	return nil
}

func (s *service) recognizeCreatedCreditPurchaseEarnings(ctx context.Context, created charges.Charges) error {
	scopes := make(map[currencyAndCustomerID]struct{})

	for _, c := range created {
		if c.Type() != meta.ChargeTypeCreditPurchase {
			continue
		}

		cp, err := c.AsCreditPurchaseCharge()
		if err != nil {
			return err
		}

		scopes[currencyAndCustomerID{
			currency: cp.Intent.Currency,
			customerID: customer.CustomerID{
				Namespace: cp.Namespace,
				ID:        cp.Intent.CustomerID,
			},
		}] = struct{}{}
	}

	for scope := range scopes {
		if err := s.recognizeCustomerEarnings(ctx, scope.customerID, scope.currency); err != nil {
			return err
		}
	}

	return nil
}

func uniqueCurrencies(currencies []currencyx.Code) []currencyx.Code {
	seen := make(map[currencyx.Code]bool, len(currencies))
	out := make([]currencyx.Code, 0, len(currencies))

	for _, currency := range currencies {
		if seen[currency] {
			continue
		}

		seen[currency] = true
		out = append(out, currency)
	}

	return out
}
