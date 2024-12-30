-- atlas:nolint MF101
-- create index "appstripe_namespace_stripe_account_id_stripe_livemode" to table: "app_stripes"
CREATE UNIQUE INDEX "appstripe_namespace_stripe_account_id_stripe_livemode" ON "app_stripes" ("namespace", "stripe_account_id", "stripe_livemode");
