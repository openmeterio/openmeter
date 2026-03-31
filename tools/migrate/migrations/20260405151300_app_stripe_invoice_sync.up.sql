-- create "app_stripe_invoice_sync_plans" table
CREATE TABLE "app_stripe_invoice_sync_plans" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "app_id" character varying NOT NULL,
  "session_id" character varying NOT NULL,
  "phase" character varying NOT NULL,
  "status" character varying NOT NULL DEFAULT 'pending',
  "error" text NULL,
  "completed_at" timestamptz NULL,
  "invoice_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "app_stripe_invoice_sync_plans_billing_invoices_app_stripe_invoi" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "appstripeinvoicesyncplan_id" to table: "app_stripe_invoice_sync_plans"
CREATE UNIQUE INDEX "appstripeinvoicesyncplan_id" ON "app_stripe_invoice_sync_plans" ("id");
-- create index "appstripeinvoicesyncplan_namespace" to table: "app_stripe_invoice_sync_plans"
CREATE INDEX "appstripeinvoicesyncplan_namespace" ON "app_stripe_invoice_sync_plans" ("namespace");
-- create index "appstripeinvoicesyncplan_namespace_id" to table: "app_stripe_invoice_sync_plans"
CREATE UNIQUE INDEX "appstripeinvoicesyncplan_namespace_id" ON "app_stripe_invoice_sync_plans" ("namespace", "id");
-- create index "appstripeinvoicesyncplan_namespace_invoice_id_session_id" to table: "app_stripe_invoice_sync_plans"
CREATE UNIQUE INDEX "appstripeinvoicesyncplan_namespace_invoice_id_session_id" ON "app_stripe_invoice_sync_plans" ("namespace", "invoice_id", "session_id");
-- create index "appstripeinvoicesyncplan_namespace_invoice_id_status" to table: "app_stripe_invoice_sync_plans"
CREATE INDEX "appstripeinvoicesyncplan_namespace_invoice_id_status" ON "app_stripe_invoice_sync_plans" ("namespace", "invoice_id", "status");
-- create "app_stripe_invoice_sync_ops" table
CREATE TABLE "app_stripe_invoice_sync_ops" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "sequence" bigint NOT NULL,
  "type" character varying NOT NULL,
  "payload" jsonb NOT NULL,
  "idempotency_key" character varying NOT NULL,
  "status" character varying NOT NULL DEFAULT 'pending',
  "stripe_response" jsonb NULL,
  "error" text NULL,
  "completed_at" timestamptz NULL,
  "plan_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "app_stripe_invoice_sync_ops_app_stripe_invoice_sync_plans_opera" FOREIGN KEY ("plan_id") REFERENCES "app_stripe_invoice_sync_plans" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "appstripeinvoicesyncop_id" to table: "app_stripe_invoice_sync_ops"
CREATE UNIQUE INDEX "appstripeinvoicesyncop_id" ON "app_stripe_invoice_sync_ops" ("id");
-- create index "appstripeinvoicesyncop_plan_id_sequence" to table: "app_stripe_invoice_sync_ops"
CREATE UNIQUE INDEX "appstripeinvoicesyncop_plan_id_sequence" ON "app_stripe_invoice_sync_ops" ("plan_id", "sequence");
-- create index "appstripeinvoicesyncop_plan_id_status" to table: "app_stripe_invoice_sync_ops"
CREATE INDEX "appstripeinvoicesyncop_plan_id_status" ON "app_stripe_invoice_sync_ops" ("plan_id", "status");
