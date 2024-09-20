package entutils

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ResourceMixin adds common fields
type ResourceMixin struct {
	mixin.Schema
}

func (ResourceMixin) Fields() []ent.Field {
	var fields []ent.Field
	fields = append(fields, IDMixin{}.Fields()...)
	fields = append(fields, KeyMixin{}.Fields()...)
	fields = append(fields, NamespaceMixin{}.Fields()...)
	fields = append(fields, MetadataAnnotationsMixin{}.Fields()...)
	fields = append(fields, TimeMixin{}.Fields()...)

	return fields
}

func (ResourceMixin) Indexes() []ent.Index {
	var indexes []ent.Index
	indexes = append(indexes, IDMixin{}.Indexes()...)
	indexes = append(indexes, NamespaceMixin{}.Indexes()...)
	indexes = append(indexes, MetadataAnnotationsMixin{}.Indexes()...)
	indexes = append(indexes, TimeMixin{}.Indexes()...)
	indexes = append(indexes, index.Fields("namespace", "id").Unique())

	// Key mixin indexes are not used, as now that we know we have namespaces, we can use a better index

	// Soft deleted items should not create a conflict with a new item with the same key.
	// The proper index would be:
	//
	// 	CREATE UNIQE INDEX x ON y (namespace, key) WHERE deleted_at IS NULL
	//
	// ENT only supports WHERE clauses on indexes via manually creating migrations, so
	// we could approximate that behavior using this index.
	//
	// Caveats: If two resources with the same key are deleted in the same microsecond then the
	// deletion will fail. (e.g. by doing a create, delete, create, delete in the same microsecond)
	indexes = append(indexes, index.Fields("namespace", "key", "deleted_at").Unique())

	return indexes
}

// IDMixin adds the ID field to the schema
type IDMixin struct {
	mixin.Schema
}

// Fields of the IDMixin.
func (IDMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			DefaultFunc(func() string {
				return ulid.Make().String()
			}).
			Unique().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
	}
}

func (IDMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id").Unique(),
	}
}

// KeyMixin adds the key field to the schema
type KeyMixin struct {
	mixin.Schema
}

// Fields of the KeyMixin.
func (KeyMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			Immutable(),
	}
}

// NamespaceMixin can be used for namespaced entities
type NamespaceMixin struct {
	mixin.Schema
}

// Fields of the IDMixin.
func (NamespaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").
			NotEmpty().
			Immutable(),
	}
}

func (NamespaceMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace"),
	}
}

// NamespaceMixin can be used for namespaced entities
type MetadataAnnotationsMixin struct {
	mixin.Schema
}

// Fields of the IDMixin.
func (MetadataAnnotationsMixin) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("metadata", map[string]string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

// TimeMixin adds the created_at and updated_at fields to the schema
type TimeMixin struct {
	mixin.Schema
}

// Fields of the TimeMixin.
func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Time("updated_at").
			Default(clock.Now).
			UpdateDefault(clock.Now),
		field.Time("deleted_at").
			Optional().
			Nillable(),
	}
}

// CustomerAddressMixin adds address fields to a customer, used by billing to snapshot addresses for invoices
type CustomerAddressMixin struct {
	ent.Schema
	FieldPrefix string
}

func (c CustomerAddressMixin) Fields() []ent.Field {
	return []ent.Field{
		// PII fields
		field.String(fmt.Sprintf("%s_address_country", c.FieldPrefix)).GoType(models.CountryCode("")).MinLen(2).MaxLen(2).Optional().Nillable(),
		field.String(fmt.Sprintf("%s_address_postal_code", c.FieldPrefix)).Optional().Nillable(),
		field.String(fmt.Sprintf("%s_address_state", c.FieldPrefix)).Optional().Nillable(),
		field.String(fmt.Sprintf("%s_address_city", c.FieldPrefix)).Optional().Nillable(),
		field.String(fmt.Sprintf("%s_address_line1", c.FieldPrefix)).Optional().Nillable(),
		field.String(fmt.Sprintf("%s_address_line2", c.FieldPrefix)).Optional().Nillable(),
		field.String(fmt.Sprintf("%s_address_phone_number", c.FieldPrefix)).Optional().Nillable(),
	}
}
