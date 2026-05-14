UPDATE
  "ledger_transactions" AS tx
SET
  "annotations" = jsonb_set(
    tx."annotations" - 'ledger.transaction.template_code',
    '{ledger.transaction.template_name}',
    to_jsonb(mapping.legacy_name::text),
    true
  )
FROM (
  VALUES
    ('customer.receivable.issue', 'IssueCustomerReceivableTemplate'),
    ('customer.receivable.payment.authorize', 'AuthorizeCustomerReceivablePaymentTemplate'),
    ('customer.receivable.payment.settle', 'SettleCustomerReceivableFromPaymentTemplate'),
    ('customer.receivable.advance.attribute', 'AttributeCustomerAdvanceReceivableCostBasisTemplate'),
    ('customer.receivable.cover', 'CoverCustomerReceivableTemplate'),
    ('customer.fbo.collect', 'TransferCustomerFBOToAccruedTemplate'),
    ('customer.fbo.advance.collect', 'TransferCustomerFBOAdvanceToAccruedTemplate'),
    ('customer.receivable.collect', 'TransferCustomerReceivableToAccruedTemplate'),
    ('customer.accrued.cost_basis.translate', 'TranslateCustomerAccruedCostBasisTemplate'),
    ('customer.accrued.earnings.recognize', 'RecognizeEarningsFromAttributableAccruedTemplate'),
    ('customer.fbo.currency.convert', 'ConvertCurrencyTemplate')
) AS mapping(template_code, legacy_name)
WHERE
  tx."annotations" ->> 'ledger.transaction.template_code' = mapping.template_code;
