package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
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
		field.String("type").GoType(appentity.AppType("")).Immutable(),
		field.String("status").GoType(appentity.AppStatus("")),
	}
}
