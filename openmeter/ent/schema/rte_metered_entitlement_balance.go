package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type RTEMeteredEntitlementBalance struct {
	ent.Schema
}

func (RTEMeteredEntitlementBalance) Mixin() []ent.Mixin {
	return []ent.Mixin{entutils.NamespaceMixin{}, entutils.TimeMixin{}}
}

func (RTEMeteredEntitlementBalance) Fields() []ent.Field {
	return []ent.Field{
		field.String("entitlement_id").NotEmpty().Immutable().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.Time("calculated_at"),
		field.String("balance").GoType(json.RawMessage{}).
			ValueScanner(entutils.JSONStringValueScanner[json.RawMessage]()).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
	}
}

func (RTEMeteredEntitlementBalance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "entitlement_id").Unique().
			StorageKey("rte_metered_balance_scope").Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
