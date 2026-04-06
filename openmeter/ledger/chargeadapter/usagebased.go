package chargeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// usageBasedHandler maps usage-based credit-only lifecycle events to ledger transaction templates.
type usageBasedHandler struct {
	ledger          ledger.Ledger
	accountResolver ledger.AccountResolver
	accountService  ledgeraccount.Service
}

var _ usagebased.Handler = (*usageBasedHandler)(nil)

func NewUsageBasedHandler(
	ledger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) usagebased.Handler {
	return &usageBasedHandler{
		ledger:          ledger,
		accountResolver: accountResolver,
		accountService:  accountService,
	}
}

func (h *usageBasedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.AmountToAllocate.IsZero() {
		return nil, nil
	}

	if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.CreditOnlySettlementMode); err != nil {
		return nil, fmt.Errorf("credits only usage accrued: %w", err)
	}

	groupID, inputs, err := allocateCreditsToAccrued(ctx, h.ledger, transactions.ResolverDependencies{
		AccountService:    h.accountResolver,
		SubAccountService: h.accountService,
	}, creditsOnlyAccrualRequest{
		Namespace:      input.Charge.Namespace,
		ChargeID:       input.Charge.ID,
		CustomerID:     input.Charge.Intent.CustomerID,
		At:             input.AllocateAt,
		Currency:       input.Charge.Intent.Currency,
		SettlementMode: input.Charge.Intent.SettlementMode,
	}, input.AmountToAllocate)
	if err != nil {
		return nil, err
	}
	if groupID == "" {
		return nil, nil
	}

	return creditRealizationsFromCollectedInputs(input.Charge.Intent.ServicePeriod, groupID, inputs...), nil
}

func (h *usageBasedHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Charge.Validate(); err != nil {
		return nil, fmt.Errorf("charge: %w", err)
	}

	if err := input.Run.Validate(); err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}

	if input.AllocateAt.IsZero() {
		return nil, fmt.Errorf("allocate at is required")
	}

	if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.CreditOnlySettlementMode); err != nil {
		return nil, fmt.Errorf("credits only usage accrued correction: %w", err)
	}

	currencyCalculator, err := input.Charge.Intent.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("get currency calculator: %w", err)
	}

	if err := input.Corrections.ValidateWith(currencyCalculator); err != nil {
		return nil, fmt.Errorf("corrections: %w", err)
	}

	return correctCreditsOnlyAccrued(ctx, h.ledger, transactions.ResolverDependencies{
		AccountService:    h.accountResolver,
		SubAccountService: h.accountService,
	}, CreditsOnlyUsageAccruedCorrectionInput{
		Namespace:   input.Charge.Namespace,
		ChargeID:    input.Charge.ID,
		AllocateAt:  input.AllocateAt,
		Corrections: input.Corrections,
	})
}
