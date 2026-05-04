package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Config struct {
	CreditPurchaseService creditpurchase.Service
	ChargesService        charges.Service
	CustomerService       customer.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.CreditPurchaseService == nil {
		errs = append(errs, errors.New("credit purchase service is required"))
	}

	if c.ChargesService == nil {
		errs = append(errs, errors.New("charges service is required"))
	}

	if c.CustomerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (creditgrant.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &service{
		creditPurchaseService: config.CreditPurchaseService,
		chargesService:        config.ChargesService,
		customerService:       config.CustomerService,
	}, nil
}

type service struct {
	creditPurchaseService creditpurchase.Service
	chargesService        charges.Service
	customerService       customer.Service
}

func (s *service) Create(ctx context.Context, input creditgrant.CreateInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err)
	}

	// Validate customer exists
	_, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
	})
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("get customer: %w", err)
	}

	// Build the credit purchase intent
	intent := toIntent(input)

	result, err := s.chargesService.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents:   charges.ChargeIntents{charges.NewChargeIntent(intent)},
	})
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("create credit grant charge: %w", err)
	}

	if len(result) != 1 {
		return creditpurchase.Charge{}, fmt.Errorf("expected 1 created charge, got %d", len(result))
	}

	createdChargeID, err := result[0].GetChargeID()
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("get created charge id: %w", err)
	}

	charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{
		ChargeID: createdChargeID,
		Expands:  meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("get created credit grant charge: %w", err)
	}

	cpCharge, err := charge.AsCreditPurchaseCharge()
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("charge is not a credit purchase: %w", err)
	}

	return cpCharge, nil
}

func (s *service) Get(ctx context.Context, input creditgrant.GetInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err)
	}

	charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: input.Namespace,
			ID:        input.ChargeID,
		},
		Expands: meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("get charge: %w", err)
	}

	cpCharge, err := charge.AsCreditPurchaseCharge()
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("charge is not a credit purchase: %w", err)
	}

	// Verify the charge belongs to the requested customer
	if cpCharge.Intent.CustomerID != input.CustomerID {
		return creditpurchase.Charge{}, fmt.Errorf("get charge: %w", models.NewGenericNotFoundError(
			fmt.Errorf("credit grant %s not found for customer %s", input.ChargeID, input.CustomerID),
		))
	}

	return cpCharge, nil
}

func (s *service) List(ctx context.Context, input creditgrant.ListInput) (pagination.Result[creditpurchase.Charge], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[creditpurchase.Charge]{}, fmt.Errorf("invalid input: %w", err)
	}

	listInput := creditpurchase.ListChargesInput{
		Page:        input.Page,
		Namespace:   input.Namespace,
		CustomerIDs: []string{input.CustomerID},
		Expands:     meta.Expands{meta.ExpandRealizations},
	}

	if input.Status != nil {
		listInput.Statuses = []meta.ChargeStatus{*input.Status}
	}

	if input.Currency != nil {
		listInput.Currencies = []currencyx.Code{*input.Currency}
	}

	return s.creditPurchaseService.List(ctx, listInput)
}

func (s *service) UpdateExternalSettlement(ctx context.Context, input creditgrant.UpdateExternalSettlementInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err)
	}

	charge, err := s.Get(ctx, creditgrant.GetInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID,
		ChargeID:   input.ChargeID,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	if charge.Intent.Settlement.Type() != creditpurchase.SettlementTypeExternal {
		return creditpurchase.Charge{}, models.NewValidationIssue(
			"credit_grant_external_settlement_not_supported",
			"credit grant is not externally funded",
			models.WithCriticalSeverity(),
			commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
			models.WithAttribute("charge_id", charge.ID),
		)
	}

	updated, err := s.chargesService.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID: meta.ChargeID{
			Namespace: input.Namespace,
			ID:        input.ChargeID,
		},
		TargetPaymentState: input.TargetStatus,
	})
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("update external settlement: %w", err)
	}

	return updated, nil
}

func toIntent(input creditgrant.CreateInput) creditpurchase.Intent {
	intent := creditpurchase.Intent{
		Intent: meta.Intent{
			Name:        input.Name,
			Description: input.Description,
			CustomerID:  input.CustomerID,
			Currency:    input.Currency,
			TaxConfig:   productcatalog.TaxCodeConfigFrom(input.TaxConfig),
			Metadata:    input.Labels,
			ManagedBy:   billing.ManuallyManagedLine,
			// TODO: replace with actual service period
			ServicePeriod:     timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now()},
			BillingPeriod:     timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now()},
			FullServicePeriod: timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now()},
		},
		CreditAmount: input.Amount,
		Settlement:   toSettlement(input),
	}

	if input.Priority != nil {
		p := int(*input.Priority)
		intent.Priority = &p
	}

	return intent
}

func toSettlement(input creditgrant.CreateInput) creditpurchase.Settlement {
	switch input.FundingMethod {
	case creditgrant.FundingMethodInvoice:
		settlement := creditpurchase.InvoiceSettlement{
			GenericSettlement: creditpurchase.GenericSettlement{
				Currency:  input.Purchase.Currency,
				CostBasis: lo.FromPtrOr(input.Purchase.PerUnitCostBasis, alpacadecimal.NewFromInt(1)),
			},
		}
		return creditpurchase.NewSettlement(settlement)

	case creditgrant.FundingMethodExternal:
		initialStatus := creditpurchase.CreatedInitialPaymentSettlementStatus
		if input.Purchase.AvailabilityPolicy != nil {
			initialStatus = *input.Purchase.AvailabilityPolicy
		}

		settlement := creditpurchase.ExternalSettlement{
			GenericSettlement: creditpurchase.GenericSettlement{
				Currency:  input.Purchase.Currency,
				CostBasis: lo.FromPtrOr(input.Purchase.PerUnitCostBasis, alpacadecimal.NewFromInt(1)),
			},
			InitialStatus: initialStatus,
		}
		return creditpurchase.NewSettlement(settlement)

	default: // FundingMethodNone → promotional
		return creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{})
	}
}
