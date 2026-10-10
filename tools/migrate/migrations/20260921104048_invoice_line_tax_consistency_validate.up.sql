-- Validate the invoice-line tax consistency checks separately from the data
-- repair and guarded installation so PostgreSQL does not hold their write locks
-- for the duration of the historical table scan.
ALTER TABLE billing_invoice_lines
  VALIDATE CONSTRAINT billing_invoice_line_tax_behavior_consistency;

ALTER TABLE billing_invoice_lines
  VALIDATE CONSTRAINT billing_invoice_line_tax_code_consistency;
