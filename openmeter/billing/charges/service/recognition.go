package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func (s *service) recognizeCustomerEarnings(ctx context.Context, customerID customer.CustomerID, currencies ...currencyx.Code) error {
	for _, currency := range lo.Uniq(currencies) {
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
	// Credit purchases can make existing accrued usage attributable by assigning
	// cost basis. Recognition is customer+currency scoped, so dedupe that scope.
	scopes := make([]currencyAndCustomerID, 0, len(created))

	for _, c := range created {
		if c.Type() != meta.ChargeTypeCreditPurchase {
			continue
		}

		customerID, err := c.GetCustomerID()
		if err != nil {
			return err
		}

		currency, err := c.GetCurrency()
		if err != nil {
			return err
		}

		scopes = append(scopes, currencyAndCustomerID{
			currency:   currency,
			customerID: customerID,
		})
	}

	for _, scope := range lo.Uniq(scopes) {
		if err := s.recognizeCustomerEarnings(ctx, scope.customerID, scope.currency); err != nil {
			return err
		}
	}

	return nil
}
