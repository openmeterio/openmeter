package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (s *service) CreateCharges(ctx context.Context, input charges.CreateChargeInputs) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Let's validate for unsupported charge types while we are building out the service
	for _, charge := range input.Intents {
		switch charge.Type() {
		case charges.ChargeTypeUsageBased:
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

		resultsByChargeID := make(map[charges.ChargeID]charges.Charge)

		if len(createdChargesByType.usageBased) > 0 || len(createdChargesByType.flatFees) > 0 {
			result, err := s.invoiceChargePostCreate(ctx, invoiceChargePostCreateInput{
				namespace:  input.Namespace,
				flatFees:   createdChargesByType.flatFees,
				usageBased: createdChargesByType.usageBased,
			})
			if err != nil {
				return nil, err
			}

			if result != nil {
				for _, flatFee := range result.flatFees {
					resultsByChargeID[flatFee.GetChargeID()] = charges.NewCharge(flatFee)
				}
				for _, usageBased := range result.usageBased {
					resultsByChargeID[usageBased.GetChargeID()] = charges.NewCharge(usageBased)
				}
			}
		}

		if len(createdChargesByType.creditPurchase) > 0 {
			for _, creditPurchase := range createdChargesByType.creditPurchase {
				result, err := s.creditPurchaseService.PostCreate(ctx, creditPurchase)
				if err != nil {
					return nil, err
				}

				resultsByChargeID[result.GetChargeID()] = charges.NewCharge(result)
			}
		}

		out, err := slicesx.MapWithErr(createdCharges, func(charge charges.Charge) (charges.Charge, error) {
			chargeID, err := charge.GetChargeID()
			if err != nil {
				return charges.Charge{}, err
			}

			updatedCharge, ok := resultsByChargeID[chargeID]
			if !ok {
				return charges.Charge{}, fmt.Errorf("charge %s not found", chargeID)
			}

			return updatedCharge, nil
		})
		if err != nil {
			return nil, err
		}

		return out, nil
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

type invoiceChargePostCreateResult struct {
	flatFees   []charges.FlatFeeCharge
	usageBased []charges.UsageBasedCharge
}

func (s *service) invoiceChargePostCreate(ctx context.Context, in invoiceChargePostCreateInput) (*invoiceChargePostCreateResult, error) {
	result := invoiceChargePostCreateResult{
		flatFees:   make([]charges.FlatFeeCharge, 0, len(in.flatFees)),
		usageBased: make([]charges.UsageBasedCharge, 0, len(in.usageBased)),
	}

	// Let's execute the post create hooks for all the charges
	gatheringLinesToCreate := make([]gatheringLineWithCustomerID, 0, len(in.flatFees)+len(in.usageBased))
	for _, flatFee := range in.flatFees {
		res, err := s.flatFeeService.PostCreate(ctx, flatFee)
		if err != nil {
			return nil, err
		}

		result.flatFees = append(result.flatFees, res.Charge)

		if res.GatheringLineToCreate != nil {
			if res.GatheringLineToCreate.ChargeID == nil {
				return nil, fmt.Errorf("gathering line charge ID is nil")
			}

			if *res.GatheringLineToCreate.ChargeID != flatFee.ID {
				return nil, fmt.Errorf("gathering line charge ID %s does not match charge ID %s", *res.GatheringLineToCreate.ChargeID, flatFee.ID)
			}

			gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
				gatheringLine: *res.GatheringLineToCreate,
				customerID:    flatFee.Intent.CustomerID,
			})
		}
	}

	if len(gatheringLinesToCreate) == 0 {
		return nil, nil
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
			return nil, fmt.Errorf("creating pending invoice lines for charges: %w", err)
		}
	}

	return nil, nil
}
