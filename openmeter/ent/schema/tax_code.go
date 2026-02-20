package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type TaxCodeAppMapping struct {
	AppType app.AppType `json:"app_type"`
	TaxCode string      `json:"tax_code"`
}

type TaxCodeAppMappings []TaxCodeAppMapping

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
			GoType(&TaxCodeAppMappings{}).
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

var TaxCodeAppMappingsValueScanner = entutils.JSONStringValueScanner[*TaxCodeAppMappings]()
