-- reverse: create index "appstripeinvoicesyncop_plan_id_status" to table: "app_stripe_invoice_sync_ops"
DROP INDEX "appstripeinvoicesyncop_plan_id_status";
-- reverse: create index "appstripeinvoicesyncop_plan_id_sequence" to table: "app_stripe_invoice_sync_ops"
DROP INDEX "appstripeinvoicesyncop_plan_id_sequence";
-- reverse: create index "appstripeinvoicesyncop_id" to table: "app_stripe_invoice_sync_ops"
DROP INDEX "appstripeinvoicesyncop_id";
-- reverse: create "app_stripe_invoice_sync_ops" table
DROP TABLE "app_stripe_invoice_sync_ops";
-- reverse: create index "appstripeinvoicesyncplan_namespace_invoice_id_status" to table: "app_stripe_invoice_sync_plans"
DROP INDEX "appstripeinvoicesyncplan_namespace_invoice_id_status";
-- reverse: create index "appstripeinvoicesyncplan_namespace_invoice_id_session_id" to table: "app_stripe_invoice_sync_plans"
DROP INDEX "appstripeinvoicesyncplan_namespace_invoice_id_session_id";
-- reverse: create index "appstripeinvoicesyncplan_namespace_id" to table: "app_stripe_invoice_sync_plans"
DROP INDEX "appstripeinvoicesyncplan_namespace_id";
-- reverse: create index "appstripeinvoicesyncplan_namespace" to table: "app_stripe_invoice_sync_plans"
DROP INDEX "appstripeinvoicesyncplan_namespace";
-- reverse: create index "appstripeinvoicesyncplan_id" to table: "app_stripe_invoice_sync_plans"
DROP INDEX "appstripeinvoicesyncplan_id";
-- reverse: create "app_stripe_invoice_sync_plans" table
DROP TABLE "app_stripe_invoice_sync_plans";
