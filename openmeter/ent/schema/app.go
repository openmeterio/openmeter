package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// App stores information about an installed app
type App struct {
	ent.Schema
}

func (App) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
	}
}

func (App) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("description"),
		field.String("type").GoType(app.AppType("")).Immutable(),
		field.String("status").GoType(app.AppStatus("")),

		// Stripe specific fields
		field.String("stripe_account_id").Optional().Nillable().Immutable(),
		field.Bool("stripe_livemode").Optional().Nillable().Immutable(),
	}
}
