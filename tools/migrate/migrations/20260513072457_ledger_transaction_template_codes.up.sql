UPDATE
  "ledger_transactions" AS tx
SET
  "annotations" = jsonb_set(
    tx."annotations" - 'ledger.transaction.template_name',
    '{ledger.transaction.template_code}',
    to_jsonb(mapping.template_code::text),
    true
  )
FROM (
  VALUES
    ('IssueCustomerReceivableTemplate', 'customer.receivable.issue'),
    ('AuthorizeCustomerReceivablePaymentTemplate', 'customer.receivable.payment.authorize'),
    ('SettleCustomerReceivableFromPaymentTemplate', 'customer.receivable.payment.settle'),
    ('AttributeCustomerAdvanceReceivableCostBasisTemplate', 'customer.receivable.advance.attribute'),
    ('CoverCustomerReceivableTemplate', 'customer.receivable.cover'),
    ('TransferCustomerFBOToAccruedTemplate', 'customer.fbo.collect'),
    ('TransferCustomerFBOAdvanceToAccruedTemplate', 'customer.fbo.advance.collect'),
    ('TransferCustomerReceivableToAccruedTemplate', 'customer.receivable.collect'),
    ('TranslateCustomerAccruedCostBasisTemplate', 'customer.accrued.cost_basis.translate'),
    ('RecognizeEarningsFromAttributableAccruedTemplate', 'customer.accrued.earnings.recognize'),
    ('ConvertCurrencyTemplate', 'customer.fbo.currency.convert')
) AS mapping(legacy_name, template_code)
WHERE
  tx."annotations" ->> 'ledger.transaction.template_name' = mapping.legacy_name;
