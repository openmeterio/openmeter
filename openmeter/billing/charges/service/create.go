package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/samber/lo"
)

func (s *service) CreateCharges(ctx context.Context, input charges.CreateChargeInputs) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Let's validate for unsupported charge types while we are building out the service
	for _, charge := range input.Intents {
		switch charge.Type() {
		case charges.ChargeTypeUsageBased, charges.ChargeTypeCreditPurchase:
			return nil, fmt.Errorf("unsupported charge type %s: %w", charge.Type(), charges.ErrUnsupported)
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		createdCharges, err := s.adapter.CreateCharges(ctx, input)
		if err != nil {
			return nil, err
		}

		createdChargesByType, err := chargesByType(createdCharges)
		if err != nil {
			return nil, err
		}

		if len(createdChargesByType.usageBased) > 0 || len(createdChargesByType.flatFees) > 0 {
			err := s.invoiceChargePostCreate(ctx, invoiceChargePostCreateInput{
				namespace:  input.Namespace,
				flatFees:   createdChargesByType.flatFees,
				usageBased: createdChargesByType.usageBased,
			})
			if err != nil {
				return nil, err
			}
		}

		return createdCharges, nil
	})
}

type invoiceChargePostCreateInput struct {
	namespace  string
	flatFees   []charges.FlatFeeCharge
	usageBased []charges.UsageBasedCharge
}

func (i invoiceChargePostCreateInput) Validate() error {
	var errs []error

	if i.namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	return errors.Join(errs...)
}

type currencyAndCustomerID struct {
	Currency   currencyx.Code
	CustomerID string
}

type gatheringLineWithCustomerID struct {
	gatheringLine billing.GatheringLine
	customerID    string
}

func (s *service) invoiceChargePostCreate(ctx context.Context, in invoiceChargePostCreateInput) error {
	// Let's execute the post create hooks for all the charges
	gatheringLinesToCreate := make([]gatheringLineWithCustomerID, 0, len(in.flatFees)+len(in.usageBased))
	for _, flatFee := range in.flatFees {
		res, err := s.flatFeeService.PostCreate(ctx, flatFee)
		if err != nil {
			return err
		}

		if res.GatheringLineToCreate != nil {
			if res.GatheringLineToCreate.ChargeID == nil {
				return fmt.Errorf("gathering line charge ID is nil")
			}

			if *res.GatheringLineToCreate.ChargeID != flatFee.ID {
				return fmt.Errorf("gathering line charge ID %s does not match charge ID %s", *res.GatheringLineToCreate.ChargeID, flatFee.ID)
			}

			gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
				gatheringLine: *res.GatheringLineToCreate,
				customerID:    flatFee.Intent.CustomerID,
			})
		}
	}

	if len(gatheringLinesToCreate) == 0 {
		return nil
	}

	uniqueCurrencyAndCustomerIDs := lo.Uniq(
		lo.Map(gatheringLinesToCreate, func(item gatheringLineWithCustomerID, _ int) currencyAndCustomerID {
			return currencyAndCustomerID{
				Currency:   item.gatheringLine.Currency,
				CustomerID: item.customerID,
			}
		}),
	)

	for _, custAndCurrency := range uniqueCurrencyAndCustomerIDs {
		gatheringLinesForCurrencyAndCustomer := lo.FilterMap(gatheringLinesToCreate, func(item gatheringLineWithCustomerID, _ int) (billing.GatheringLine, bool) {
			return item.gatheringLine, item.customerID == custAndCurrency.CustomerID && item.gatheringLine.Currency == custAndCurrency.Currency
		})

		if len(gatheringLinesForCurrencyAndCustomer) == 0 {
			continue
		}

		// Let's create the gathering invoice on invoicing side
		_, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: customer.CustomerID{
				Namespace: in.namespace,
				ID:        custAndCurrency.CustomerID,
			},
			Currency: custAndCurrency.Currency,
			Lines:    gatheringLinesForCurrencyAndCustomer,
		})
		if err != nil {
			return fmt.Errorf("creating pending invoice lines for charges: %w", err)
		}
	}

	return nil
}
