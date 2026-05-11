package transactions

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type TransactionTemplateCode string

const (
	TemplateCodeIssueCustomerReceivable                     TransactionTemplateCode = "customer.receivable.issue"
	TemplateCodeAuthorizeCustomerReceivablePayment          TransactionTemplateCode = "customer.receivable.payment.authorize"
	TemplateCodeSettleCustomerReceivableFromPayment         TransactionTemplateCode = "customer.receivable.payment.settle"
	TemplateCodeAttributeCustomerAdvanceReceivableCostBasis TransactionTemplateCode = "customer.receivable.advance.attribute"
	TemplateCodeCoverCustomerReceivable                     TransactionTemplateCode = "customer.receivable.cover"
	TemplateCodeTransferCustomerFBOToAccrued                TransactionTemplateCode = "customer.fbo.collect"
	TemplateCodeTransferCustomerFBOAdvanceToAccrued         TransactionTemplateCode = "customer.fbo.advance.collect"
	TemplateCodeTransferCustomerReceivableToAccrued         TransactionTemplateCode = "customer.receivable.collect"
	TemplateCodeTranslateCustomerAccruedCostBasis           TransactionTemplateCode = "customer.accrued.cost_basis.translate"
	TemplateCodeRecognizeEarningsFromAttributableAccrued    TransactionTemplateCode = "customer.accrued.earnings.recognize"
	TemplateCodeConvertCurrency                             TransactionTemplateCode = "customer.fbo.currency.convert"
)

const (
	legacyTemplateNameIssueCustomerReceivable                     = "IssueCustomerReceivableTemplate"
	legacyTemplateNameFundCustomerReceivable                      = "FundCustomerReceivableTemplate"
	legacyTemplateNameSettleCustomerReceivablePayment             = "SettleCustomerReceivablePaymentTemplate"
	legacyTemplateNameAuthorizeCustomerReceivablePayment          = "AuthorizeCustomerReceivablePaymentTemplate"
	legacyTemplateNameSettleCustomerReceivableFromPayment         = "SettleCustomerReceivableFromPaymentTemplate"
	legacyTemplateNameAttributeCustomerAdvanceReceivableCostBasis = "AttributeCustomerAdvanceReceivableCostBasisTemplate"
	legacyTemplateNameCoverCustomerReceivable                     = "CoverCustomerReceivableTemplate"
	legacyTemplateNameTransferCustomerFBOToAccrued                = "TransferCustomerFBOToAccruedTemplate"
	legacyTemplateNameTransferCustomerFBOAdvanceToAccrued         = "TransferCustomerFBOAdvanceToAccruedTemplate"
	legacyTemplateNameTransferCustomerReceivableToAccrued         = "TransferCustomerReceivableToAccruedTemplate"
	legacyTemplateNameTranslateCustomerAccruedCostBasis           = "TranslateCustomerAccruedCostBasisTemplate"
	legacyTemplateNameRecognizeEarningsFromAttributableAccrued    = "RecognizeEarningsFromAttributableAccruedTemplate"
	legacyTemplateNameConvertCurrency                             = "ConvertCurrencyTemplate"
)

func templateCode(template TransactionTemplate) (TransactionTemplateCode, error) {
	switch any(template).(type) {
	case IssueCustomerReceivableTemplate, *IssueCustomerReceivableTemplate:
		return TemplateCodeIssueCustomerReceivable, nil
	case AuthorizeCustomerReceivablePaymentTemplate, *AuthorizeCustomerReceivablePaymentTemplate:
		return TemplateCodeAuthorizeCustomerReceivablePayment, nil
	case SettleCustomerReceivableFromPaymentTemplate, *SettleCustomerReceivableFromPaymentTemplate:
		return TemplateCodeSettleCustomerReceivableFromPayment, nil
	case AttributeCustomerAdvanceReceivableCostBasisTemplate, *AttributeCustomerAdvanceReceivableCostBasisTemplate:
		return TemplateCodeAttributeCustomerAdvanceReceivableCostBasis, nil
	case CoverCustomerReceivableTemplate, *CoverCustomerReceivableTemplate:
		return TemplateCodeCoverCustomerReceivable, nil
	case TransferCustomerFBOToAccruedTemplate, *TransferCustomerFBOToAccruedTemplate:
		return TemplateCodeTransferCustomerFBOToAccrued, nil
	case TransferCustomerFBOAdvanceToAccruedTemplate, *TransferCustomerFBOAdvanceToAccruedTemplate:
		return TemplateCodeTransferCustomerFBOAdvanceToAccrued, nil
	case TransferCustomerReceivableToAccruedTemplate, *TransferCustomerReceivableToAccruedTemplate:
		return TemplateCodeTransferCustomerReceivableToAccrued, nil
	case TranslateCustomerAccruedCostBasisTemplate, *TranslateCustomerAccruedCostBasisTemplate:
		return TemplateCodeTranslateCustomerAccruedCostBasis, nil
	case RecognizeEarningsFromAttributableAccruedTemplate, *RecognizeEarningsFromAttributableAccruedTemplate:
		return TemplateCodeRecognizeEarningsFromAttributableAccrued, nil
	case ConvertCurrencyTemplate, *ConvertCurrencyTemplate:
		return TemplateCodeConvertCurrency, nil
	default:
		return "", fmt.Errorf("unknown transaction template code for %T", template)
	}
}

func TemplateCode(template TransactionTemplate) string {
	code, err := templateCode(template)
	if err != nil {
		panic(err)
	}

	return string(code)
}

func annotateTemplateTransaction(input ledger.TransactionInput, template TransactionTemplate, direction ledger.TransactionDirection) (ledger.TransactionInput, error) {
	code, err := templateCode(template)
	if err != nil {
		return nil, err
	}

	return WithAnnotations(input, ledger.TransactionAnnotations(string(code), direction)), nil
}

func appendResolvedTemplateTransaction(
	inputs []ledger.TransactionInput,
	input ledger.TransactionInput,
	template TransactionTemplate,
	direction ledger.TransactionDirection,
) ([]ledger.TransactionInput, error) {
	if input == nil {
		return inputs, nil
	}

	annotated, err := annotateTemplateTransaction(input, template, direction)
	if err != nil {
		return nil, err
	}

	return append(inputs, annotated), nil
}
