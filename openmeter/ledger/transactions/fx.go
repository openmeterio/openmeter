package transactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	goblcurrency "github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ConvertCurrencyTemplate struct {
	At           time.Time
	SourceAmount alpacadecimal.Decimal
	TargetAmount alpacadecimal.Decimal
	CostBasis    alpacadecimal.Decimal

	SourceCurrency currencyx.Code
	TargetCurrency currencyx.Code
}

func (t ConvertCurrencyTemplate) Validate() error {
	var errs []error

	if t.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	sourceAmountValid := true
	if err := ledger.ValidateTransactionAmount(t.SourceAmount); err != nil {
		errs = append(errs, fmt.Errorf("source amount: %w", err))
		sourceAmountValid = false
	}

	targetAmountValid := true
	if err := ledger.ValidateTransactionAmount(t.TargetAmount); err != nil {
		errs = append(errs, fmt.Errorf("target amount: %w", err))
		targetAmountValid = false
	}

	costBasisValid := true
	if err := ledger.ValidateCostBasis(t.CostBasis); err != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", err))
		costBasisValid = false
	} else if t.CostBasis.IsZero() {
		errs = append(errs, errors.New("cost basis must be positive"))
		costBasisValid = false
	}

	sourceCurrencyDefinition := goblcurrency.Get(goblcurrency.Code(t.SourceCurrency))
	if err := ledger.ValidateCurrency(t.SourceCurrency); err != nil {
		errs = append(errs, fmt.Errorf("source currency: %w", err))
		sourceCurrencyDefinition = nil
	} else if sourceCurrencyDefinition == nil {
		errs = append(errs, errors.New("source currency must be a known fiat currency"))
	}

	if err := ledger.ValidateCurrency(t.TargetCurrency); err != nil {
		errs = append(errs, fmt.Errorf("target currency: %w", err))
	} else if goblcurrency.Get(goblcurrency.Code(t.TargetCurrency)) != nil {
		errs = append(errs, errors.New("target currency must be custom"))
	}

	if sourceAmountValid && targetAmountValid && costBasisValid && sourceCurrencyDefinition != nil {
		expectedSourceAmount := t.TargetAmount.Mul(t.CostBasis).Round(int32(sourceCurrencyDefinition.Subunits))
		if !t.SourceAmount.Equal(expectedSourceAmount) {
			errs = append(errs, fmt.Errorf(
				"source amount: expected %s from target amount multiplied by cost basis, got %s",
				expectedSourceAmount.String(),
				t.SourceAmount.String(),
			))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ CustomerTransactionTemplate = (ConvertCurrencyTemplate{})

func (t ConvertCurrencyTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
}

func (t ConvertCurrencyTemplate) typeGuard() guard {
	return true
}

func (t ConvertCurrencyTemplate) code() TransactionTemplateCode {
	return TemplateCodeConvertCurrency
}

func (t ConvertCurrencyTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	costBasis := t.CostBasis
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	sourceAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.SourceCurrency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source sub-account: %w", err)
	}

	targetAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.TargetCurrency,
		ExchangeSourceCurrency:         lo.ToPtr(t.SourceCurrency),
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get target sub-account: %w", err)
	}

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	brokerageSource, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:  t.SourceCurrency,
		CostBasis: &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage source sub-account: %w", err)
	}

	brokerageTarget, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:               t.TargetCurrency,
		ExchangeSourceCurrency: lo.ToPtr(t.SourceCurrency),
		CostBasis:              &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage target sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			// Source currency
			{
				address: sourceAccount.Address(),
				amount:  t.SourceAmount.Neg(),
			},
			{
				address: brokerageSource.Address(),
				amount:  t.SourceAmount,
			},
			// Target currency
			{
				address: targetAccount.Address(),
				amount:  t.TargetAmount,
			},
			{
				address: brokerageTarget.Address(),
				amount:  t.TargetAmount.Neg(),
			},
		},
	}, nil
}
