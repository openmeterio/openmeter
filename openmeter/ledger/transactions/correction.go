package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type CorrectionInput struct {
	At     time.Time
	Amount alpacadecimal.Decimal

	OriginalTransaction ledger.Transaction
	OriginalGroup       ledger.TransactionGroup
}

type CorrectionScope = CorrectionInput

func (i CorrectionScope) Validate() error {
	if i.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if i.OriginalTransaction == nil {
		return fmt.Errorf("original transaction is required")
	}

	return nil
}

func CorrectTransaction(
	ctx context.Context,
	deps ResolverDependencies,
	scope CorrectionScope,
) ([]ledger.TransactionInput, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("validate correction input: %w", err)
	}

	direction, err := ledger.TransactionDirectionFromAnnotations(scope.OriginalTransaction.Annotations())
	if err != nil {
		return nil, fmt.Errorf("transaction direction: %w", err)
	}

	if direction == ledger.TransactionDirectionCorrection {
		return nil, fmt.Errorf("cannot correct a correction transaction")
	}

	templateName, err := ledger.TransactionTemplateNameFromAnnotations(scope.OriginalTransaction.Annotations())
	if err != nil {
		return nil, fmt.Errorf("transaction template name: %w", err)
	}

	template, err := transactionTemplateByName(templateName)
	if err != nil {
		return nil, err
	}

	outputs, err := correctTemplate(ctx, deps, scope, template)
	if err != nil {
		return nil, err
	}

	annotated := make([]ledger.TransactionInput, 0, len(outputs))
	for _, output := range outputs {
		annotated = append(annotated, annotateTemplateTransaction(output, template, ledger.TransactionDirectionCorrection))
	}

	return annotated, nil
}

func transactionTemplateByName(name string) (TransactionTemplate, error) {
	switch name {
	case templateName(IssueCustomerReceivableTemplate{}):
		return IssueCustomerReceivableTemplate{}, nil
	case templateName(FundCustomerReceivableTemplate{}):
		return FundCustomerReceivableTemplate{}, nil
	case templateName(SettleCustomerReceivablePaymentTemplate{}):
		return SettleCustomerReceivablePaymentTemplate{}, nil
	case templateName(AttributeCustomerAdvanceReceivableCostBasisTemplate{}):
		return AttributeCustomerAdvanceReceivableCostBasisTemplate{}, nil
	case templateName(CoverCustomerReceivableTemplate{}):
		return CoverCustomerReceivableTemplate{}, nil
	case templateName(TransferCustomerFBOToAccruedTemplate{}):
		return TransferCustomerFBOToAccruedTemplate{}, nil
	case templateName(TransferCustomerFBOBucketToAccruedTemplate{}):
		return TransferCustomerFBOBucketToAccruedTemplate{}, nil
	case templateName(TransferCustomerReceivableToAccruedTemplate{}):
		return TransferCustomerReceivableToAccruedTemplate{}, nil
	case templateName(TranslateCustomerAccruedCostBasisTemplate{}):
		return TranslateCustomerAccruedCostBasisTemplate{}, nil
	case templateName(RecognizeEarningsFromAttributableAccruedTemplate{}):
		return RecognizeEarningsFromAttributableAccruedTemplate{}, nil
	case templateName(ConvertCurrencyTemplate{}):
		return ConvertCurrencyTemplate{}, nil
	default:
		return nil, fmt.Errorf("unknown correction template %q", name)
	}
}

func correctTemplate(
	ctx context.Context,
	deps ResolverDependencies,
	scope CorrectionScope,
	template TransactionTemplate,
) ([]ledger.TransactionInput, error) {
	switch typ := any(template).(type) {
	case CustomerTransactionTemplate:
		return typ.correct(ctx, scope, deps)
	case OrgTransactionTemplate:
		return typ.correct(ctx, scope, deps)
	default:
		return nil, fmt.Errorf("unsupported correction template type %T", template)
	}
}

func templateCorrectionNotImplemented(name string) error {
	return fmt.Errorf("%s correction is not implemented", name)
}
