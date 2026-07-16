CREATE INDEX IF NOT EXISTS om_mig_bil_fee_cfg_id_idx
    ON billing_invoice_lines (fee_line_config_id);
CREATE INDEX IF NOT EXISTS om_mig_bil_ubp_cfg_id_idx
    ON billing_invoice_lines (usage_based_line_config_id);
CREATE INDEX IF NOT EXISTS om_mig_bil_parent_id_idx
    ON billing_invoice_lines (parent_line_id);
CREATE INDEX IF NOT EXISTS om_mig_bild_line_id_idx
    ON billing_invoice_line_discounts (line_id);
CREATE INDEX IF NOT EXISTS om_mig_biuld_line_id_idx
    ON billing_invoice_line_usage_discounts (line_id);
CREATE INDEX IF NOT EXISTS om_mig_bsidl_parent_id_idx
    ON billing_standard_invoice_detailed_lines (parent_line_id);
CREATE INDEX IF NOT EXISTS om_mig_bsidlad_line_id_idx
    ON billing_standard_invoice_detailed_line_amount_discounts (line_id);
