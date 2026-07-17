package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BalanceSnapshot struct {
	ent.Schema
}

func (BalanceSnapshot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BalanceSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.String("owner_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.JSON("grant_balances", balance.Map{}).Immutable().SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}),
		field.JSON("usage", &balance.SnapshottedUsage{}).Immutable().Optional().SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}),
		field.Float("balance").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Float("overage").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Time("at").Immutable(),
		// unit_config records the conversion regime this snapshot was computed under
		// (OM-400). The resume path only reuses a snapshot whose config matches the
		// owner's current one, so a later regime change (e.g. a backfill) forces a
		// recompute instead of mixing raw and converted units. Nil = raw. Immutable:
		// a snapshot's regime is fixed at write time.
		field.String("unit_config").
			GoType(&productcatalog.UnitConfig{}).
			ValueScanner(UnitConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable().
			Immutable(),
	}
}

func (BalanceSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "owner_id", "at").Annotations(
			entsql.IndexWhere("deleted_at IS NULL"),
		),
	}
}

func (BalanceSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entitlement", Entitlement.Type).
			Ref("balance_snapshot").
			Field("owner_id").
			Required().
			Immutable().
			Unique(),
	}
}
