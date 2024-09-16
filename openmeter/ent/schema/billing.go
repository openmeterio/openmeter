package schema

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BillingProfile struct {
	ent.Schema
}

func (BillingProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingProfile) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			Immutable(),
		field.String("provider_config").
			GoType(provider.Configuration{}).
			ValueScanner(ProviderConfigValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
		field.JSON("billing_config", billing.Configuration{}).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
		field.Bool("default").
			Default(false),
	}
}

func (BillingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
	}
}

func (BillingProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key"),
		index.Fields("namespace", "id"),
		index.Fields("namespace", "default"),
	}
}

type providerConfigSerde[T any] struct {
	provider.Meta

	Config T `json:"config"`
}

var ProviderConfigValueScanner = field.ValueScannerFunc[provider.Configuration, *sql.NullString]{
	V: func(config provider.Configuration) (driver.Value, error) {
		switch config.Type {
		case provider.TypeOpenMeter:
			return json.Marshal(providerConfigSerde[provider.OpenMeterConfig]{
				Meta:   provider.Meta{Type: provider.TypeOpenMeter},
				Config: config.OpenMeter,
			})
		case provider.TypeStripe:
			return json.Marshal(providerConfigSerde[provider.StripeConfig]{
				Meta:   provider.Meta{Type: provider.TypeStripe},
				Config: config.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", config.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.Configuration, error) {
		if !ns.Valid {
			return provider.Configuration{}, errors.New("backend config is null")
		}

		data := []byte(ns.String)

		var meta provider.Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			return provider.Configuration{}, err
		}

		switch meta.Type {
		case provider.TypeOpenMeter:
			serde := providerConfigSerde[provider.OpenMeterConfig]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Configuration{}, err
			}

			return provider.Configuration{
				Meta:      serde.Meta,
				OpenMeter: serde.Config,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[provider.StripeConfig]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Configuration{}, err
			}

			return provider.Configuration{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.Configuration{}, fmt.Errorf("unknown backend type: %s", meta.Type)
		}
	},
}

type BillingInvoiceItem struct {
	ent.Schema
}

func (BillingInvoiceItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
	}
}

func (BillingInvoiceItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoice_id").
			Optional().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),
		field.String("customer_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),

		field.Time("period_start"),
		field.Time("period_end"),
		field.Time("invoice_at"),

		// TODO[dependency]: overrides (as soon as plan override entities are ready)

		field.Other("quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.Other("unit_price", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.String("currency").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "varchar(3)",
			}),
		field.JSON("tax_code_override", invoice.TaxOverrides{}).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
	}
}

func (BillingInvoiceItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "invoice_id"),
		index.Fields("namespace", "customer_id"),
	}
}

func (BillingInvoiceItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice", BillingInvoice.Type).
			Ref("billing_invoice_items").
			Field("invoice_id").
			Unique(),
		// TODO[dependency]: Customer edge, as soon as customer entities are ready

	}
}

type BillingInvoice struct {
	ent.Schema
}

func (BillingInvoice) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
	}
}

func (BillingInvoice) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			Immutable(),
		field.String("customer_id").
			NotEmpty().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}).
			Immutable(),
		field.String("billing_profile_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),
		field.Time("voided_at").
			Optional(),
		field.String("currency").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "varchar(3)",
			}),
		field.Other("total_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.Time("due_date"),
		field.Enum("status").
			GoType(invoice.InvoiceStatus("")),

		field.String("provider_config").
			GoType(provider.Configuration{}).
			ValueScanner(ProviderConfigValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
		field.JSON("billing_config", billing.Configuration{}).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
		field.String("provider_reference").
			GoType(provider.Reference{}).
			ValueScanner(ProviderReferenceValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),

		field.Time("period_start"),
		field.Time("period_end"),
	}
}

func (BillingInvoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "key"),
		index.Fields("namespace", "customer_id"),
		index.Fields("namespace", "due_date"),
		index.Fields("namespace", "status"),
	}
}

func (BillingInvoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_profile", BillingProfile.Type).
			Ref("billing_invoices").
			Field("billing_profile_id").
			Required().
			Unique().
			Immutable(), // Billing profile changes are forbidden => invoice must be voided in this case
		edge.To("billing_invoice_items", BillingInvoiceItem.Type),
		// Customer edge!
	}
}

type providerReferenceSerde[T any] struct {
	provider.Meta

	Reference T `json:"ref"`
}

var ProviderReferenceValueScanner = field.ValueScannerFunc[provider.Reference, *sql.NullString]{
	V: func(ref provider.Reference) (driver.Value, error) {
		switch ref.Type {
		case provider.TypeOpenMeter:
			return json.Marshal(providerReferenceSerde[provider.OpenMeterReference]{
				Meta:      provider.Meta{Type: provider.TypeOpenMeter},
				Reference: ref.OpenMeter,
			})
		case provider.TypeStripe:
			return json.Marshal(providerReferenceSerde[provider.StripeReference]{
				Meta:      provider.Meta{Type: provider.TypeStripe},
				Reference: ref.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", ref.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.Reference, error) {
		if !ns.Valid {
			return provider.Reference{}, errors.New("backend config is null")
		}

		data := []byte(ns.String)

		var meta provider.Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			return provider.Reference{}, err
		}

		switch meta.Type {
		case provider.TypeOpenMeter:
			serde := providerReferenceSerde[provider.OpenMeterReference]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Reference{}, err
			}

			return provider.Reference{
				Meta:      serde.Meta,
				OpenMeter: serde.Reference,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[provider.StripeReference]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Reference{}, err
			}

			return provider.Reference{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.Reference{}, fmt.Errorf("unknown backend type: %s", meta.Type)
		}
	},
}
