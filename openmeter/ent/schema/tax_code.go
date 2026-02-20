package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type TaxCodeAppMapping struct {
	AppType app.AppType `json:"app_type"`
	TaxCode string      `json:"tax_code"`
}

type TaxCodeAppMappings []TaxCodeAppMapping

// Tax code stores information about an entities tax code
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

var TaxCodeAppMappingsValueScanner = entutils.JSONStringValueScanner[*TaxCodeAppMappings]()
