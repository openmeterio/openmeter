package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// OrganizationDefaultTaxCodes stores the organization-level default tax code references.
type OrganizationDefaultTaxCodes struct {
	ent.Schema
}

func (OrganizationDefaultTaxCodes) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (OrganizationDefaultTaxCodes) Fields() []ent.Field {
	return []ent.Field{
		// namespace is defined explicitly instead of via NamespaceMixin to avoid
		// the mixin's non-unique namespace index duplicating our unique constraint below.
		field.String("namespace").
			NotEmpty().
			Immutable(),
		field.String("invoicing_tax_code_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.String("credit_grant_tax_code_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
	}
}

func (OrganizationDefaultTaxCodes) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("invoicing_tax_code", TaxCode.Type).
			Ref("organization_default_invoicing").
			Field("invoicing_tax_code_id").
			Required().
			Unique(),
		edge.From("credit_grant_tax_code", TaxCode.Type).
			Ref("organization_default_credit_grant").
			Field("credit_grant_tax_code_id").
			Required().
			Unique(),
	}
}

func (OrganizationDefaultTaxCodes) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
	}
}
