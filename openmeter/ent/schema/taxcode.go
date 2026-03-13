package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"entgo.io/ent/schema/mixin"
)

// Tax code stores information about an entity's tax code
type TaxCode struct {
	ent.Schema
}

func (TaxCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
	}
}

func (TaxCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("app_mappings").
			GoType(&taxcode.TaxCodeAppMappings{}).
			ValueScanner(TaxCodeAppMappingsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

func (TaxCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (TaxCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_workflow_configs", BillingWorkflowConfig.Type),
		edge.To("billing_customer_overrides", BillingCustomerOverride.Type),
		edge.To("billing_invoice_lines", BillingInvoiceLine.Type),
		edge.To("billing_invoice_split_line_groups", BillingInvoiceSplitLineGroup.Type),
		edge.To("billing_standard_invoice_detailed_lines", BillingStandardInvoiceDetailedLine.Type),
		edge.To("subscription_items", SubscriptionItem.Type),
		edge.To("plan_rate_cards", PlanRateCard.Type),
		edge.To("addon_rate_cards", AddonRateCard.Type),
	}
}

var TaxCodeAppMappingsValueScanner = entutils.JSONStringValueScanner[*taxcode.TaxCodeAppMappings]()

// TaxMixin adds tax_code_id and tax_behavior fields to a schema.
type TaxMixin struct {
	mixin.Schema
}

func (TaxMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("tax_code_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Enum("tax_behavior").
			GoType(productcatalog.TaxBehavior("")).
			Optional().
			Nillable(),
	}
}
