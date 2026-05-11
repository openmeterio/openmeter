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

var transactionTemplatesByLegacyName = map[string]TransactionTemplate{
	legacyTemplateNameIssueCustomerReceivable:                     IssueCustomerReceivableTemplate{},
	legacyTemplateNameFundCustomerReceivable:                      legacyFundCustomerReceivableTemplate{},
	legacyTemplateNameSettleCustomerReceivablePayment:             legacySettleCustomerReceivablePaymentTemplate{},
	legacyTemplateNameAuthorizeCustomerReceivablePayment:          AuthorizeCustomerReceivablePaymentTemplate{},
	legacyTemplateNameSettleCustomerReceivableFromPayment:         SettleCustomerReceivableFromPaymentTemplate{},
	legacyTemplateNameAttributeCustomerAdvanceReceivableCostBasis: AttributeCustomerAdvanceReceivableCostBasisTemplate{},
	legacyTemplateNameCoverCustomerReceivable:                     CoverCustomerReceivableTemplate{},
	legacyTemplateNameTransferCustomerFBOToAccrued:                TransferCustomerFBOToAccruedTemplate{},
	legacyTemplateNameTransferCustomerFBOAdvanceToAccrued:         TransferCustomerFBOAdvanceToAccruedTemplate{},
	legacyTemplateNameTransferCustomerReceivableToAccrued:         TransferCustomerReceivableToAccruedTemplate{},
	legacyTemplateNameTranslateCustomerAccruedCostBasis:           TranslateCustomerAccruedCostBasisTemplate{},
	legacyTemplateNameRecognizeEarningsFromAttributableAccrued:    RecognizeEarningsFromAttributableAccruedTemplate{},
	legacyTemplateNameConvertCurrency:                             ConvertCurrencyTemplate{},
}

var transactionTemplatesByCode = map[TransactionTemplateCode]TransactionTemplate{
	IssueCustomerReceivableTemplate{}.code():                     IssueCustomerReceivableTemplate{},
	AuthorizeCustomerReceivablePaymentTemplate{}.code():          AuthorizeCustomerReceivablePaymentTemplate{},
	SettleCustomerReceivableFromPaymentTemplate{}.code():         SettleCustomerReceivableFromPaymentTemplate{},
	AttributeCustomerAdvanceReceivableCostBasisTemplate{}.code(): AttributeCustomerAdvanceReceivableCostBasisTemplate{},
	CoverCustomerReceivableTemplate{}.code():                     CoverCustomerReceivableTemplate{},
	TransferCustomerFBOToAccruedTemplate{}.code():                TransferCustomerFBOToAccruedTemplate{},
	TransferCustomerFBOAdvanceToAccruedTemplate{}.code():         TransferCustomerFBOAdvanceToAccruedTemplate{},
	TransferCustomerReceivableToAccruedTemplate{}.code():         TransferCustomerReceivableToAccruedTemplate{},
	TranslateCustomerAccruedCostBasisTemplate{}.code():           TranslateCustomerAccruedCostBasisTemplate{},
	RecognizeEarningsFromAttributableAccruedTemplate{}.code():    RecognizeEarningsFromAttributableAccruedTemplate{},
	ConvertCurrencyTemplate{}.code():                             ConvertCurrencyTemplate{},
}

func templateCode(template TransactionTemplate) (TransactionTemplateCode, error) {
	code := template.code()
	if code == "" {
		return "", fmt.Errorf("unknown transaction template code for %T", template)
	}

	return code, nil
}

func transactionTemplateByCode(code string) (TransactionTemplate, error) {
	template, ok := transactionTemplatesByCode[TransactionTemplateCode(code)]
	if !ok {
		return nil, fmt.Errorf("unknown correction template code %q", code)
	}

	return template, nil
}

func transactionTemplateByLegacyName(name string) (TransactionTemplate, error) {
	template, ok := transactionTemplatesByLegacyName[name]
	if !ok {
		return nil, fmt.Errorf("unknown correction template name %q", name)
	}

	return template, nil
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
