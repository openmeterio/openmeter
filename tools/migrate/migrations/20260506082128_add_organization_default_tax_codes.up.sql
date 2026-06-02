-- create "organization_default_tax_codes" table
CREATE TABLE "organization_default_tax_codes" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "invoicing_tax_code_id" character(26) NOT NULL,
  "credit_grant_tax_code_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "org_dtc_credit_grant_tax_code_fk" FOREIGN KEY ("credit_grant_tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "org_dtc_invoicing_tax_code_fk" FOREIGN KEY ("invoicing_tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "organizationdefaulttaxcodes_id" to table: "organization_default_tax_codes"
CREATE UNIQUE INDEX "organizationdefaulttaxcodes_id" ON "organization_default_tax_codes" ("id");
-- create index "organizationdefaulttaxcodes_namespace" to table: "organization_default_tax_codes"
CREATE UNIQUE INDEX "organizationdefaulttaxcodes_namespace" ON "organization_default_tax_codes" ("namespace") WHERE (deleted_at IS NULL);
