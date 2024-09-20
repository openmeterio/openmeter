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
	"github.com/openmeterio/openmeter/openmeter/billing/provider/openmetersandbox"
	"github.com/openmeterio/openmeter/openmeter/billing/provider/stripe"
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

		field.Enum("tax_provider").GoType(provider.TaxProvider("")).Optional().Nillable(),
		field.String("tax_provider_config").
			GoType(provider.TaxConfiguration{}).
			ValueScanner(ProviderTaxConfigurationValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}).Optional(),

		field.Enum("invoicing_provider").GoType(provider.InvoicingProvider("")).Optional().Nillable(),
		field.String("invoicing_provider_config").
			GoType(provider.InvoicingConfiguration{}).
			ValueScanner(ProviderInvoicingConfigurationValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}).Optional(),

		field.Enum("payment_provider").GoType(provider.PaymentProvider("")).Optional().Nillable(),
		field.String("payment_provider_config").
			GoType(provider.PaymentConfiguration{}).
			ValueScanner(ProviderPaymentConfigurationValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}).Optional(),

		field.String("workflow_config_id").
			NotEmpty(),
		field.Bool("default").
			Default(false),
	}
}

func (BillingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
		edge.From("billing_workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_profile").
			Field("workflow_config_id").
			Unique().
			Required(),
		edge.From("customers", Customer.Type).
			Ref("override_billing_profile"),
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

	Config T `json:"config,omitempty"`
}

func billingProviderTypeFromField(ns *sql.NullString) (provider.Type, error) {
	if !ns.Valid {
		return "", errors.New("backend config is null")
	}

	data := []byte(ns.String)

	var meta provider.Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}

	return meta.Type, nil
}

var ProviderTaxConfigurationValueScanner = field.ValueScannerFunc[provider.TaxConfiguration, *sql.NullString]{
	V: func(ref provider.TaxConfiguration) (driver.Value, error) {
		switch ref.Type {
		case provider.TypeOpenMeterSandbox:
			return json.Marshal(providerConfigSerde[openmetersandbox.TaxConfiguration]{
				Meta:   provider.Meta{Type: provider.TypeOpenMeterSandbox},
				Config: ref.OpenMeter,
			})
		case provider.TypeStripe:
			return json.Marshal(providerConfigSerde[stripe.TaxConfiguration]{
				Meta:   provider.Meta{Type: provider.TypeStripe},
				Config: ref.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", ref.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.TaxConfiguration, error) {
		providerType, err := billingProviderTypeFromField(ns)
		if err != nil {
			return provider.TaxConfiguration{}, err
		}

		data := []byte(ns.String)

		switch providerType {
		case provider.TypeOpenMeterSandbox:
			serde := providerConfigSerde[openmetersandbox.TaxConfiguration]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.TaxConfiguration{}, err
			}

			return provider.TaxConfiguration{
				Meta:      serde.Meta,
				OpenMeter: serde.Config,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[stripe.TaxConfiguration]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.TaxConfiguration{}, err
			}

			return provider.TaxConfiguration{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.TaxConfiguration{}, fmt.Errorf("unknown backend type: %s", providerType)
		}
	},
}

var ProviderInvoicingConfigurationValueScanner = field.ValueScannerFunc[provider.InvoicingConfiguration, *sql.NullString]{
	V: func(ref provider.InvoicingConfiguration) (driver.Value, error) {
		switch ref.Type {
		case provider.TypeOpenMeterSandbox:
			return json.Marshal(providerConfigSerde[openmetersandbox.InvoicingConfiguration]{
				Meta:   provider.Meta{Type: provider.TypeOpenMeterSandbox},
				Config: ref.OpenMeter,
			})
		case provider.TypeStripe:
			return json.Marshal(providerConfigSerde[stripe.InvoicingConfiguration]{
				Meta:   provider.Meta{Type: provider.TypeStripe},
				Config: ref.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", ref.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.InvoicingConfiguration, error) {
		providerType, err := billingProviderTypeFromField(ns)
		if err != nil {
			return provider.InvoicingConfiguration{}, err
		}

		data := []byte(ns.String)

		switch providerType {
		case provider.TypeOpenMeterSandbox:
			serde := providerConfigSerde[openmetersandbox.InvoicingConfiguration]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.InvoicingConfiguration{}, err
			}

			return provider.InvoicingConfiguration{
				Meta:      serde.Meta,
				OpenMeter: serde.Config,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[stripe.InvoicingConfiguration]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.InvoicingConfiguration{}, err
			}

			return provider.InvoicingConfiguration{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.InvoicingConfiguration{}, fmt.Errorf("unknown backend type: %s", providerType)
		}
	},
}

var ProviderPaymentConfigurationValueScanner = field.ValueScannerFunc[provider.PaymentConfiguration, *sql.NullString]{
	V: func(ref provider.PaymentConfiguration) (driver.Value, error) {
		switch ref.Type {
		case provider.TypeOpenMeterSandbox:
			return json.Marshal(providerConfigSerde[openmetersandbox.PaymentConfiguration]{
				Meta:   provider.Meta{Type: provider.TypeOpenMeterSandbox},
				Config: ref.OpenMeter,
			})
		case provider.TypeStripe:
			return json.Marshal(providerConfigSerde[stripe.PaymentConfiguration]{
				Meta:   provider.Meta{Type: provider.TypeStripe},
				Config: ref.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", ref.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.PaymentConfiguration, error) {
		providerType, err := billingProviderTypeFromField(ns)
		if err != nil {
			return provider.PaymentConfiguration{}, err
		}

		data := []byte(ns.String)

		switch providerType {
		case provider.TypeOpenMeterSandbox:
			serde := providerConfigSerde[openmetersandbox.PaymentConfiguration]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.PaymentConfiguration{}, err
			}

			return provider.PaymentConfiguration{
				Meta:      serde.Meta,
				OpenMeter: serde.Config,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[stripe.PaymentConfiguration]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.PaymentConfiguration{}, err
			}

			return provider.PaymentConfiguration{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.PaymentConfiguration{}, fmt.Errorf("unknown backend type: %s", providerType)
		}
	},
}

type BillingWorkflowConfig struct {
	ent.Schema
}

func (BillingWorkflowConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingWorkflowConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("alignment").
			GoType(billing.AlignmentKind("")),

		// TODO: later we will add more alignment details here (e.g. monthly, yearly, etc.)

		field.Int64("collection_period_seconds"),

		field.Bool("invoice_auto_advance").
			Nillable(),

		field.Int64("invoice_draft_period_seconds"),

		field.Int64("invoice_due_after_seconds"),

		field.Enum("invoice_collection_method").
			GoType(billing.CollectionMethod("")),

		field.Enum("invoice_line_item_resolution").
			GoType(billing.GranualityResolution("")),

		field.Bool("invoice_line_item_per_subject").
			Default(false),
	}
}

func (BillingWorkflowConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

func (BillingWorkflowConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
		edge.To("billing_profile", BillingProfile.Type),
	}
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

		field.Enum("tax_provider").GoType(provider.TaxProvider("")).Optional().Nillable(),
		field.String("tax_provider_config").
			GoType(provider.TaxConfiguration{}).
			ValueScanner(ProviderTaxConfigurationValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}).Optional(),

		field.Enum("invoicing_provider").GoType(provider.InvoicingProvider("")).Optional().Nillable(),
		field.String("invoicing_provider_config").
			GoType(provider.InvoicingConfiguration{}).
			ValueScanner(ProviderInvoicingConfigurationValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}).Optional(),

		field.Enum("payment_provider").GoType(provider.PaymentProvider("")).Optional().Nillable(),
		field.String("payment_provider_config").
			GoType(provider.PaymentConfiguration{}).
			ValueScanner(ProviderPaymentConfigurationValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}).Optional(),

		field.String("workflow_config_id").
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),

		// TODO[later]: Add either provider annotations or typed provider status fields

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
		edge.From("billing_workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_invoices").
			Field("workflow_config_id").
			Unique().
			Required(),
		edge.To("billing_invoice_items", BillingInvoiceItem.Type),
	}
}
