package schema

import (
	"encoding/json"
	"fmt"
	"slices"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// stringOneOf returns an ent field validator that rejects values not in the allowed set.
func stringOneOf(allowed []string) func(string) error {
	return func(v string) error {
		if !slices.Contains(allowed, v) {
			return fmt.Errorf("invalid value %q, must be one of %v", v, allowed)
		}
		return nil
	}
}

// AppStripeInvoiceSyncPlan holds a persistent, ordered set of Stripe operations for a single sync session.
type AppStripeInvoiceSyncPlan struct {
	ent.Schema
}

func (AppStripeInvoiceSyncPlan) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (AppStripeInvoiceSyncPlan) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoice_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable().
			NotEmpty(),
		field.String("app_id").
			Immutable().
			NotEmpty().
			Comment("The Stripe app ID that owns this sync plan"),
		field.String("session_id").
			Immutable().
			NotEmpty().
			Comment("Unique identifier for this sync session, scopes idempotency keys"),
		field.String("phase").
			GoType(invoicesync.SyncPlanPhase("")).
			Immutable().
			Validate(stringOneOf(invoicesync.SyncPlanPhase("").Values())).
			Comment("Which invoice state machine phase: draft, issuing, or delete"),
		field.String("status").
			GoType(invoicesync.PlanStatus("")).
			Default(string(invoicesync.PlanStatusPending)).
			Validate(stringOneOf(invoicesync.PlanStatus("").Values())),
		field.Text("error").
			Optional().
			Nillable(),
		field.Time("completed_at").
			Optional().
			Nillable(),
	}
}

func (AppStripeInvoiceSyncPlan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice", BillingInvoice.Type).
			Ref("app_stripe_invoice_sync_plans").
			Field("invoice_id").
			Required().
			Immutable().
			Unique(),
		edge.To("operations", AppStripeInvoiceSyncOp.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (AppStripeInvoiceSyncPlan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "invoice_id", "session_id").Unique(),
		index.Fields("namespace", "invoice_id", "status"),
	}
}

// AppStripeInvoiceSyncOp holds a single Stripe API operation within a sync plan.
type AppStripeInvoiceSyncOp struct {
	ent.Schema
}

func (AppStripeInvoiceSyncOp) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (AppStripeInvoiceSyncOp) Fields() []ent.Field {
	return []ent.Field{
		field.String("plan_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable().
			NotEmpty(),
		field.Int("sequence").
			Immutable().
			NonNegative().
			Comment("Execution order within the plan"),
		field.String("type").
			GoType(invoicesync.OpType("")).
			Immutable().
			Validate(stringOneOf(invoicesync.OpType("").Values())),
		field.JSON("payload", json.RawMessage{}).
			Comment("Operation-specific parameters serialized as JSON"),
		field.String("idempotency_key").
			Immutable().
			NotEmpty().
			Comment("Deterministic key: sha256(invoiceID + sessionID + sequence + opType)"),
		field.String("status").
			GoType(invoicesync.OpStatus("")).
			Default(string(invoicesync.OpStatusPending)).
			Validate(stringOneOf(invoicesync.OpStatus("").Values())),
		field.JSON("stripe_response", json.RawMessage{}).
			Optional().
			Comment("Raw Stripe response stored on completion for debugging/audit"),
		field.Text("error").
			Optional().
			Nillable(),
		field.Time("completed_at").
			Optional().
			Nillable(),
	}
}

func (AppStripeInvoiceSyncOp) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sync_plan", AppStripeInvoiceSyncPlan.Type).
			Ref("operations").
			Field("plan_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (AppStripeInvoiceSyncOp) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("plan_id", "sequence").Unique(),
		index.Fields("plan_id", "status"),
	}
}
