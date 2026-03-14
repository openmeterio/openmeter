package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) Create(ctx context.Context, input charges.CreateInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Let's validate for unsupported charge types while we are building out the service
	for _, charge := range input.Intents {
		switch charge.Type() {
		case meta.ChargeTypeUsageBased:
			return nil, fmt.Errorf("unsupported charge type %s: %w", charge.Type(), meta.ErrUnsupported)
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		intentsByType, err := input.Intents.ByType()
		if err != nil {
			return nil, err
		}

		createdCharges := make([]charges.WithIndex[charges.Charge], 0, len(input.Intents))
		gatheringLinesToCreate := make([]gatheringLineWithCustomerID, 0, len(input.Intents))

		// Let's create all the flat fee charges in bulk and record any gathering lines to create
		flatFees, err := s.flatFeeService.Create(ctx, flatfee.CreateInput{
			Namespace: input.Namespace,
			Intents: lo.Map(intentsByType.FlatFee, func(intent charges.WithIndex[flatfee.Intent], _ int) flatfee.Intent {
				return intent.Value
			}),
		})
		if err != nil {
			return nil, err
		}

		createdCharges = append(
			createdCharges,
			lo.Map(flatFees, func(fee flatfee.ChargeWithGatheringLine, idx int) charges.WithIndex[charges.Charge] {
				return charges.WithIndex[charges.Charge]{
					Index: intentsByType.FlatFee[idx].Index,
					Value: charges.NewCharge(fee.Charge),
				}
			})...,
		)

		for _, fee := range flatFees {
			if fee.GatheringLineToCreate != nil {
				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *fee.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        fee.Charge.Intent.CustomerID,
					},
				})
			}
		}

		// Let's generate the gathering lines for the flat fees
		if err := s.createGatheringLines(ctx, gatheringLinesToCreate); err != nil {
			return nil, err
		}

		// Let's create all the credit purchase charges in bulk
		for _, intent := range intentsByType.CreditPurchase {
			charge, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: input.Namespace,
				Intent:    intent.Value,
			})
			if err != nil {
				return nil, err
			}

			createdCharges = append(createdCharges, charges.WithIndex[charges.Charge]{
				Index: intent.Index,
				Value: charges.NewCharge(charge),
			})
		}

		// Let's map the created charges to the original intents
		out := make([]charges.Charge, len(input.Intents))
		for _, createdCharge := range createdCharges {
			out[createdCharge.Index] = createdCharge.Value
		}

		return out, nil
	})
}

type invoiceChargePostCreateInput struct {
	namespace string
	flatFees  []flatfee.Charge
}

func (i invoiceChargePostCreateInput) Validate() error {
	var errs []error

	if i.namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	return errors.Join(errs...)
}

type currencyAndCustomerID struct {
	currency   currencyx.Code
	customerID customer.CustomerID
}

type gatheringLineWithCustomerID struct {
	gatheringLine billing.GatheringLine
	customerID    customer.CustomerID
}

func (s *service) createGatheringLines(ctx context.Context, gatheringLinesToCreate []gatheringLineWithCustomerID) error {
	if len(gatheringLinesToCreate) == 0 {
		return nil
	}

	uniqueCurrencyAndCustomerIDs := lo.Uniq(
		lo.Map(gatheringLinesToCreate, func(item gatheringLineWithCustomerID, _ int) currencyAndCustomerID {
			return currencyAndCustomerID{
				currency:   item.gatheringLine.Currency,
				customerID: item.customerID,
			}
		}),
	)

	for _, custAndCurrency := range uniqueCurrencyAndCustomerIDs {
		gatheringLinesForCurrencyAndCustomer := lo.FilterMap(gatheringLinesToCreate, func(item gatheringLineWithCustomerID, _ int) (billing.GatheringLine, bool) {
			return item.gatheringLine, item.customerID == custAndCurrency.customerID && item.gatheringLine.Currency == custAndCurrency.currency
		})

		if len(gatheringLinesForCurrencyAndCustomer) == 0 {
			continue
		}

		// Let's create the gathering invoice on invoicing side
		_, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: custAndCurrency.customerID,
			Currency: custAndCurrency.currency,
			Lines:    gatheringLinesForCurrencyAndCustomer,
		})
		if err != nil {
			return fmt.Errorf("creating pending invoice lines for charges: %w", err)
		}
	}

	return nil
}
