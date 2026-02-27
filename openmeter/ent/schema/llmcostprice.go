package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// LLMCostPrice stores canonical LLM pricing (global + namespace overrides).
type LLMCostPrice struct {
	ent.Schema
}

func (LLMCostPrice) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.MetadataMixin{},
		entutils.TimeMixin{},
	}
}

func (LLMCostPrice) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").
			Optional().
			Nillable().
			Comment("Nil for global prices, set for namespace overrides"),
		field.String("provider").
			NotEmpty(),
		field.String("model_id").
			NotEmpty(),
		field.String("model_name").
			Default(""),
		field.Other("input_per_token", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("output_per_token", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("input_cached_per_token", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Default(alpacadecimal.Decimal{}),
		field.Other("reasoning_per_token", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Default(alpacadecimal.Decimal{}),
		field.Other("cache_write_per_token", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Default(alpacadecimal.Decimal{}),
		field.String("currency").
			Default("USD"),
		field.String("source").
			NotEmpty(),
		field.String("source_prices").
			GoType(llmcost.SourcePricesMap{}).
			ValueScanner(entutils.JSONStringValueScanner[llmcost.SourcePricesMap]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
		field.Time("effective_from"),
		field.Time("effective_to").
			Optional().
			Nillable(),
	}
}

func (LLMCostPrice) Indexes() []ent.Index {
	return []ent.Index{
		// Unique: one active price per provider+model+namespace+effective_from
		index.Fields("provider", "model_id", "namespace", "effective_from").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
		// Lookup by namespace for override resolution
		index.Fields("namespace", "provider", "model_id").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		// Global price lookup (namespace IS NULL)
		index.Fields("provider", "model_id").
			Annotations(entsql.IndexWhere("deleted_at IS NULL AND namespace IS NULL")),
	}
}
