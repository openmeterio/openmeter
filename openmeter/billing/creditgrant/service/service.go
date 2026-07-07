package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	customerbilling "github.com/openmeterio/openmeter/openmeter/billing/validators/customerbilling"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Config struct {
	CreditPurchaseService creditpurchase.Service
	ChargesService        charges.Service
	BillingService        billing.Service
	CustomerService       customer.Service
	CreditVoidService     creditvoid.Service
	TransactionManager    transaction.Creator
}

func (c Config) Validate() error {
	var errs []error

	if c.CreditPurchaseService == nil {
		errs = append(errs, errors.New("credit purchase service is required"))
	}

	if c.ChargesService == nil {
		errs = append(errs, errors.New("charges service is required"))
	}

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service is required"))
	}

	if c.CustomerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	if c.CreditVoidService == nil {
		errs = append(errs, errors.New("credit void service is required"))
	}

	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
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
		billingService:        config.BillingService,
		customerService:       config.CustomerService,
		creditVoidService:     config.CreditVoidService,
		transactionManager:    config.TransactionManager,
	}, nil
}

type service struct {
	creditPurchaseService creditpurchase.Service
	chargesService        charges.Service
	billingService        billing.Service
	customerService       customer.Service
	creditVoidService     creditvoid.Service
	transactionManager    transaction.Creator
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

	if input.FundingMethod == creditgrant.FundingMethodInvoice {
		if err := customerbilling.ValidateCustomerInvoicingApp(
			ctx,
			s.billingService,
			customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
			[]app.CapabilityType{
				app.CapabilityTypeCalculateTax,
				app.CapabilityTypeInvoiceCustomers,
				app.CapabilityTypeCollectPayments,
			},
		); err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("invalid billing setup: %w", err)
		}
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

	return s.Get(ctx, creditgrant.GetInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID,
		ChargeID:   createdChargeID.ID,
	})
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
		switch *input.Status {
		case creditgrant.GrantStatusPending:
			listInput.Statuses = []meta.ChargeStatus{meta.ChargeStatusCreated}
		case creditgrant.GrantStatusActive:
			// Final charges read as public status active (promotional grants
			// settle straight to final) until their optional expiry time passes.
			listInput.Statuses = []meta.ChargeStatus{meta.ChargeStatusActive, meta.ChargeStatusFinal}
			listInput.Voided = lo.ToPtr(false)
			listInput.Expiration = &creditpurchase.ListChargesExpirationFilter{
				AsOf:    clock.Now().UTC(),
				Expired: false,
			}
		case creditgrant.GrantStatusExpired:
			listInput.Voided = lo.ToPtr(false)
			listInput.Expiration = &creditpurchase.ListChargesExpirationFilter{
				AsOf:    clock.Now().UTC(),
				Expired: true,
			}
		case creditgrant.GrantStatusVoided:
			listInput.Voided = lo.ToPtr(true)
		default:
			return pagination.Result[creditpurchase.Charge]{}, models.NewGenericValidationError(fmt.Errorf("invalid grant status filter: %s", *input.Status))
		}
	}

	if input.Currency != nil {
		listInput.Currencies = []currencyx.Code{*input.Currency}
	}

	listInput.Key = input.Key

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

func (s *service) Void(ctx context.Context, input creditgrant.VoidInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err)
	}

	charge, err := s.Get(ctx, creditgrant.GetInput(input))
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	if charge.State.VoidedAt != nil {
		return charge, nil
	}

	if err := validateChargeVoidable(charge); err != nil {
		return creditpurchase.Charge{}, err
	}

	return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (creditpurchase.Charge, error) {
		result, err := s.creditVoidService.VoidCreditPurchase(ctx, creditvoid.VoidCreditPurchaseInput{
			CustomerID: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
			ChargeID:  charge.ID,
			Currency:  charge.Intent.Currency,
			ExpiresAt: charge.Intent.ExpiresAt,
			Annotations: ledger.ChargeAnnotations(models.NamespacedID{
				Namespace: charge.Namespace,
				ID:        charge.ID,
			}),
		})
		if err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("void credit purchase: %w", err)
		}

		voided, err := s.creditPurchaseService.MarkVoided(ctx, creditpurchase.MarkVoidedInput{
			ChargeID: charge.GetChargeID(),
			VoidedAt: result.VoidedAt,
		})
		if err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("mark charge voided: %w", err)
		}

		return charge.WithBase(voided), nil
	})
}

// validateChargeVoidable rejects charges whose lifecycle state makes a
// ledger-only void dishonest: a created (pending) charge has not funded FBO
// yet and would still fund later, a deleted charge is not readable as a
// grant, and an already expired charge has nothing left to void.
func validateChargeVoidable(charge creditpurchase.Charge) error {
	if charge.DeletedAt != nil {
		return models.NewGenericNotFoundError(fmt.Errorf("credit grant %s not found", charge.ID))
	}

	metaStatus, err := charge.Status.ToMetaChargeStatus()
	if err != nil {
		return fmt.Errorf("charge status: %w", err)
	}

	switch metaStatus {
	case meta.ChargeStatusActive, meta.ChargeStatusFinal:
	case meta.ChargeStatusCreated:
		return models.NewGenericConflictError(fmt.Errorf("credit grant %s is pending and cannot be voided", charge.ID))
	default:
		return models.NewGenericConflictError(fmt.Errorf("credit grant %s cannot be voided in status %s", charge.ID, charge.Status))
	}

	if charge.Intent.ExpiresAt != nil && !charge.Intent.ExpiresAt.After(clock.Now()) {
		return models.NewGenericConflictError(fmt.Errorf("credit grant %s has already expired", charge.ID))
	}

	return nil
}

func toIntent(input creditgrant.CreateInput) creditpurchase.Intent {
	effectiveAt := lo.FromPtrOr(input.EffectiveAt, clock.Now()).UTC()
	period := timeutil.ClosedPeriod{From: effectiveAt, To: effectiveAt}

	intent := creditpurchase.Intent{
		Intent: meta.Intent{
			CustomerID: input.CustomerID,
			Currency:   input.Currency,
			ManagedBy:  billing.ManuallyManagedLine,
			TaxConfig:  productcatalog.TaxCodeConfigFrom(input.TaxConfig),
		},
		IntentMutableFields: creditpurchase.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:        input.Name,
				Description: input.Description,
				Metadata:    input.Labels,
				// TODO: replace with actual service period
				ServicePeriod:     period,
				BillingPeriod:     period,
				FullServicePeriod: period,
			},
			CreditAmount: input.Amount,
			EffectiveAt:  input.EffectiveAt,
			ExpiresAt:    calculateExpiresAt(effectiveAt, input.ExpiresAfter),
			Settlement:   toSettlement(input),
		},
		Key: input.Key,
	}

	if input.Filters != nil {
		intent.FeatureFilters = creditpurchase.FeatureFilters(input.Filters.Features).Normalize()
	}

	if input.Priority != nil {
		p := int(*input.Priority)
		intent.Priority = &p
	}

	return intent
}

func calculateExpiresAt(from time.Time, expiresAfter *datetime.ISODuration) *time.Time {
	if expiresAfter == nil {
		return nil
	}

	expiresAt, _ := expiresAfter.AddTo(from)

	return &expiresAt
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
