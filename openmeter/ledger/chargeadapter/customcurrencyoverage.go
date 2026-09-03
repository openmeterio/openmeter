package chargeadapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type bookCustomCurrencyOverageInput struct {
	Namespace      string
	ChargeID       string
	CustomerID     string
	Annotations    models.Annotations
	BookedAt       time.Time
	CustomCurrency currencies.CurrencyReference
	FiatCurrency   currencyx.Code
	CustomAmount   alpacadecimal.Decimal
	FiatAmount     alpacadecimal.Decimal
	CostBasis      alpacadecimal.Decimal
	TaxConfig      productcatalog.TaxCodeConfig
}

type correctCustomCurrencyOverageInput struct {
	Namespace        string
	TransactionGroup ledgertransaction.GroupReference
	Annotations      models.Annotations
	BookedAt         time.Time
	CustomAmount     alpacadecimal.Decimal
	CostBasis        alpacadecimal.Decimal
}

type customCurrencyOverageHandler struct {
	ledger ledger.Ledger
	deps   transactions.ResolverDependencies
}

func newCustomCurrencyOverageHandler(ledgerService ledger.Ledger, resolverDeps transactions.ResolverDependencies) *customCurrencyOverageHandler {
	return &customCurrencyOverageHandler{
		ledger: ledgerService,
		deps:   resolverDeps,
	}
}

func (i bookCustomCurrencyOverageInput) Validate() error {
	var errs []error

	chargeID := models.NamespacedID{Namespace: i.Namespace, ID: i.ChargeID}
	if err := chargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	customerID := customer.CustomerID{Namespace: i.Namespace, ID: i.CustomerID}
	if err := customerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}

	if err := i.CustomCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("custom currency: %w", err))
	} else if !i.CustomCurrency.IsCustom() {
		errs = append(errs, errors.New("custom currency must be custom typed currency"))
	}

	if err := i.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	} else if !i.FiatCurrency.IsFiat() {
		errs = append(errs, errors.New("fiat currency must be fiat typed currency"))
	}

	if err := ledger.ValidateTransactionAmount(i.CustomAmount); err != nil {
		errs = append(errs, fmt.Errorf("custom amount: %w", err))
	}

	if err := ledger.ValidateTransactionAmount(i.FiatAmount); err != nil {
		errs = append(errs, fmt.Errorf("fiat amount: %w", err))
	}

	if err := ledger.ValidateCostBasis(i.CostBasis); err != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", err))
	} else if i.CostBasis.IsZero() {
		errs = append(errs, errors.New("cost basis must be positive"))
	}

	if err := i.TaxConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax config: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i correctCustomCurrencyOverageInput) Validate() error {
	var errs []error

	groupID := models.NamespacedID{Namespace: i.Namespace, ID: i.TransactionGroup.TransactionGroupID}
	if err := groupID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("transaction group: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}

	if err := ledger.ValidateTransactionAmount(i.CustomAmount); err != nil {
		errs = append(errs, fmt.Errorf("custom amount: %w", err))
	}

	if err := ledger.ValidateCostBasis(i.CostBasis); err != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", err))
	} else if i.CostBasis.IsZero() {
		errs = append(errs, errors.New("cost basis must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// book records uncovered custom-currency overage with credit-purchase-equivalent
// ledger legs, immediately consumed by the same charge. Issuance and consumption
// use the same route so the custom amount never becomes spendable customer
// balance; the resulting custom receivable is converted into the already-agreed,
// rounded gross fiat receivable that credit coverage and invoice payment resolve.
func (h *customCurrencyOverageHandler) book(ctx context.Context, input bookCustomCurrencyOverageInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	customerID := customer.CustomerID{
		Namespace: input.Namespace,
		ID:        input.CustomerID,
	}

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:                input.BookedAt,
			Amount:            input.CustomAmount,
			Currency:          input.CustomCurrency,
			CostBasisCurrency: &input.FiatCurrency,
			CostBasis:         &input.CostBasis,
			SourceChargeID:    &input.ChargeID,
		},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:                input.BookedAt,
			Amount:            input.CustomAmount,
			Currency:          input.CustomCurrency,
			TaxCode:           lo.ToPtr(input.TaxConfig.TaxCodeID),
			TaxBehavior:       (*ledger.TaxBehavior)(input.TaxConfig.Behavior),
			CostBasisCurrency: &input.FiatCurrency,
			CostBasis:         &input.CostBasis,
			SourceChargeID:    &input.ChargeID,
			SpendChargeID:     &input.ChargeID,
		},
		transactions.ConvertCurrencyTemplate{
			At:             input.BookedAt,
			SourceAmount:   input.FiatAmount,
			TargetAmount:   input.CustomAmount,
			CostBasis:      input.CostBasis,
			SourceCurrency: currencies.NewCurrencyReference(input.FiatCurrency),
			TargetCurrency: input.CustomCurrency,
			SourceChargeID: &input.ChargeID,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	for i, transactionInput := range inputs {
		if transactionInput != nil {
			inputs[i] = transactions.WithAnnotations(transactionInput, input.Annotations)
		}
	}

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Namespace,
		input.Annotations,
		inputs...,
	))
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

// correct reverses a complete custom-currency overage booking in dependency
// order: currency conversion, immediate consumption, then credit issuance.
func (h *customCurrencyOverageHandler) correct(ctx context.Context, input correctCustomCurrencyOverageInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	originalGroup, err := h.ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        input.TransactionGroup.TransactionGroupID,
	})
	if err != nil {
		return fmt.Errorf("get original transaction group: %w", err)
	}

	expectedForwardOrder := []string{
		transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
		transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
		transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
	}
	originalTransactions := originalGroup.Transactions()
	if len(originalTransactions) != len(expectedForwardOrder) {
		return fmt.Errorf("custom-currency overage group has %d transactions, expected %d", len(originalTransactions), len(expectedForwardOrder))
	}

	// The booking owns this transaction order, and transaction groups reload in
	// creation order. Validate that contract before unwinding it in reverse.
	for idx, transaction := range originalTransactions {
		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return fmt.Errorf("get original transaction direction: %w", err)
		}
		if direction != ledger.TransactionDirectionForward {
			return fmt.Errorf("original transaction %s is not forward", transaction.ID().ID)
		}

		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		if err != nil {
			return fmt.Errorf("get original transaction template code: %w", err)
		}
		if templateCode != expectedForwardOrder[idx] {
			return fmt.Errorf("unexpected transaction template %s at index %d in custom-currency overage group, expected %s", templateCode, idx, expectedForwardOrder[idx])
		}
	}

	correctionInputs := make([]ledger.TransactionInput, 0, len(originalTransactions))
	for idx := len(originalTransactions) - 1; idx >= 0; idx-- {
		originalTransaction := originalTransactions[idx]
		resolved, err := transactions.CorrectTransaction(ctx, h.deps, transactions.CorrectionInput{
			At:                  input.BookedAt,
			Amount:              input.CustomAmount,
			CostBasis:           &input.CostBasis,
			OriginalTransaction: originalTransaction,
			OriginalGroup:       originalGroup,
		})
		if err != nil {
			return fmt.Errorf("correct transaction template %s: %w", expectedForwardOrder[idx], err)
		}

		for _, correctionInput := range resolved {
			correctionInputs = append(correctionInputs, transactions.WithAnnotations(correctionInput, input.Annotations))
		}
	}

	if _, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Namespace,
		input.Annotations,
		correctionInputs...,
	)); err != nil {
		return fmt.Errorf("commit correction transaction group: %w", err)
	}

	return nil
}
