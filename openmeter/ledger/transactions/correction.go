package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

const legacyAnnotationTransactionTemplateName = "ledger.transaction.template_name"

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
	_ context.Context,
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

	template, err := transactionTemplateFromAnnotations(scope.OriginalTransaction.Annotations())
	if err != nil {
		return nil, fmt.Errorf("transaction template: %w", err)
	}

	outputs, err := correctTemplate(scope, template)
	if err != nil {
		return nil, err
	}

	annotated := make([]ledger.TransactionInput, 0, len(outputs))
	for _, output := range outputs {
		annotatedOutput, err := annotateTemplateTransaction(output, template, ledger.TransactionDirectionCorrection)
		if err != nil {
			return nil, err
		}

		annotated = append(annotated, annotatedOutput)
	}

	return annotated, nil
}

func transactionTemplateFromAnnotations(annotations models.Annotations) (TransactionTemplate, error) {
	if _, ok := annotations[ledger.AnnotationTransactionTemplateCode]; ok {
		code, err := ledger.TransactionTemplateCodeFromAnnotations(annotations)
		if err != nil {
			return nil, fmt.Errorf("code: %w", err)
		}

		return transactionTemplateByCode(code)
	}

	name, err := transactionTemplateNameFromAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("name: %w", err)
	}

	return transactionTemplateByLegacyName(name)
}

func transactionTemplateNameFromAnnotations(annotations models.Annotations) (string, error) {
	raw, ok := annotations[legacyAnnotationTransactionTemplateName]
	if !ok {
		return "", fmt.Errorf("transaction template name annotation is required")
	}

	templateName, ok := raw.(string)
	if !ok || templateName == "" {
		return "", fmt.Errorf("transaction template name annotation is invalid")
	}

	return templateName, nil
}

func transactionTemplateByCode(code string) (TransactionTemplate, error) {
	switch code {
	case string(TemplateCodeIssueCustomerReceivable):
		return IssueCustomerReceivableTemplate{}, nil
	case string(TemplateCodeAuthorizeCustomerReceivablePayment):
		return AuthorizeCustomerReceivablePaymentTemplate{}, nil
	case string(TemplateCodeSettleCustomerReceivableFromPayment):
		return SettleCustomerReceivableFromPaymentTemplate{}, nil
	case string(TemplateCodeAttributeCustomerAdvanceReceivableCostBasis):
		return AttributeCustomerAdvanceReceivableCostBasisTemplate{}, nil
	case string(TemplateCodeCoverCustomerReceivable):
		return CoverCustomerReceivableTemplate{}, nil
	case string(TemplateCodeTransferCustomerFBOToAccrued):
		return TransferCustomerFBOToAccruedTemplate{}, nil
	case string(TemplateCodeTransferCustomerFBOAdvanceToAccrued):
		return TransferCustomerFBOAdvanceToAccruedTemplate{}, nil
	case string(TemplateCodeTransferCustomerReceivableToAccrued):
		return TransferCustomerReceivableToAccruedTemplate{}, nil
	case string(TemplateCodeTranslateCustomerAccruedCostBasis):
		return TranslateCustomerAccruedCostBasisTemplate{}, nil
	case string(TemplateCodeRecognizeEarningsFromAttributableAccrued):
		return RecognizeEarningsFromAttributableAccruedTemplate{}, nil
	case string(TemplateCodeConvertCurrency):
		return ConvertCurrencyTemplate{}, nil
	default:
		return nil, fmt.Errorf("unknown correction template code %q", code)
	}
}

func transactionTemplateByLegacyName(name string) (TransactionTemplate, error) {
	switch name {
	case legacyTemplateNameIssueCustomerReceivable:
		return IssueCustomerReceivableTemplate{}, nil
	case legacyTemplateNameFundCustomerReceivable:
		return legacyFundCustomerReceivableTemplate{}, nil
	case legacyTemplateNameSettleCustomerReceivablePayment:
		return legacySettleCustomerReceivablePaymentTemplate{}, nil
	case legacyTemplateNameAuthorizeCustomerReceivablePayment:
		return AuthorizeCustomerReceivablePaymentTemplate{}, nil
	case legacyTemplateNameSettleCustomerReceivableFromPayment:
		return SettleCustomerReceivableFromPaymentTemplate{}, nil
	case legacyTemplateNameAttributeCustomerAdvanceReceivableCostBasis:
		return AttributeCustomerAdvanceReceivableCostBasisTemplate{}, nil
	case legacyTemplateNameCoverCustomerReceivable:
		return CoverCustomerReceivableTemplate{}, nil
	case legacyTemplateNameTransferCustomerFBOToAccrued:
		return TransferCustomerFBOToAccruedTemplate{}, nil
	case legacyTemplateNameTransferCustomerFBOAdvanceToAccrued:
		return TransferCustomerFBOAdvanceToAccruedTemplate{}, nil
	case legacyTemplateNameTransferCustomerReceivableToAccrued:
		return TransferCustomerReceivableToAccruedTemplate{}, nil
	case legacyTemplateNameTranslateCustomerAccruedCostBasis:
		return TranslateCustomerAccruedCostBasisTemplate{}, nil
	case legacyTemplateNameRecognizeEarningsFromAttributableAccrued:
		return RecognizeEarningsFromAttributableAccruedTemplate{}, nil
	case legacyTemplateNameConvertCurrency:
		return ConvertCurrencyTemplate{}, nil
	default:
		return nil, fmt.Errorf("unknown correction template name %q", name)
	}
}

func correctTemplate(scope CorrectionScope, template TransactionTemplate) ([]ledger.TransactionInput, error) {
	switch typ := any(template).(type) {
	case CustomerTransactionTemplate:
		return typ.correct(scope)
	case OrgTransactionTemplate:
		return typ.correct(scope)
	default:
		return nil, fmt.Errorf("unsupported correction template type %T", template)
	}
}

func templateCorrectionNotImplemented(template string) error {
	return fmt.Errorf("%s correction is not implemented", template)
}
