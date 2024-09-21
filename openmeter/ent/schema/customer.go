// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// Customer stores information about a customer
type Customer struct {
	ent.Schema
}

func (Customer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
		entutils.CustomerAddressMixin{
			FieldPrefix: "billing",
		},
	}
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.String("currency").GoType(models.CurrencyCode("")).MinLen(3).MaxLen(3).Optional().Nillable(),
		field.String("timezone").GoType(timezone.Timezone("")).Optional().Nillable(),
		field.Enum("tax_provider").GoType(models.TaxProvider("")).Optional().Nillable(),
		field.Enum("invoicing_provider").GoType(models.InvoicingProvider("")).Optional().Nillable(),
		field.Enum("payment_provider").GoType(models.PaymentProvider("")).Optional().Nillable(),
		field.String("external_mapping_stripe_customer_id").Optional().Nillable(),
		field.String("name"),
		field.String("primary_email").Optional().Nillable(),
	}
}

func (Customer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subjects", CustomerSubjects.Type),
	}
}

// CustomerSubject stores the subject keys for a customer
type CustomerSubjects struct {
	ent.Schema
}

func (CustomerSubjects) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			Immutable().
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("subject_key").
			Immutable().
			NotEmpty(),
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
	}
}

func (CustomerSubjects) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("customer_id", "subject_key").
			Unique(),
	}
}

func (CustomerSubjects) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Ref("subjects").
			Field("customer_id").
			Required().
			Immutable().
			Unique(),
	}
}
