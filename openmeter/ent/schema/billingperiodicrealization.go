package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// BillingPeriodicRealizationRun is a namespace's realization run for one period.
type BillingPeriodicRealizationRun struct {
	ent.Schema
}

func (BillingPeriodicRealizationRun) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingPeriodicRealizationRun) Fields() []ent.Field {
	return []ent.Field{
		// The latest run of the namespace with an earlier period when this one was
		// created. Periods can be skipped, so this is not always one period back.
		field.String("previous_run_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			Optional().
			Nillable().
			Immutable(),
		// End of the period the run realizes usage in. The period excludes it.
		field.Time("period_to").Immutable(),
		// Usage up to the period end is only safe to read once the collection
		// delay has passed. Stored so that changing the delay does not move
		// existing runs.
		field.Time("realize_after").Immutable(),
		field.Enum("status").GoType(periodicrealization.RunStatus("")),
		field.Time("started_at").Optional().Nillable(),
		field.Time("finished_at").Optional().Nillable(),
		// Written when the run is closed. Checkpoints created for the run after
		// that are not counted.
		field.Int("charges_total").NonNegative().Optional().Nillable(),
		field.Int("charges_done").NonNegative().Optional().Nillable(),
		field.Int("charges_failed").NonNegative().Optional().Nillable(),
		field.Int("charges_skipped").NonNegative().Optional().Nillable(),
	}
}

func (BillingPeriodicRealizationRun) Edges() []ent.Edge {
	return []ent.Edge{
		// One-to-one, so previous_run_id is unique: two runs pointing at the same
		// previous run would fork the namespace's history.
		edge.To("next_run", BillingPeriodicRealizationRun.Type).
			Unique().
			StorageKey(edge.Symbol("billing_periodic_realization_run_previous_fk")).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.From("previous_run", BillingPeriodicRealizationRun.Type).
			Ref("next_run").
			Field("previous_run_id").
			Unique().
			Immutable(),
		edge.To("errors", BillingPeriodicRealizationRunError.Type).
			StorageKey(edge.Symbol("billing_periodic_run_error_run_fk")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("charge_states", BillingPeriodicRealizationChargeState.Type).
			StorageKey(edge.Symbol("billing_periodic_charge_state_run_fk")).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.To("usage_based_runs", ChargeUsageBasedRuns.Type).
			StorageKey(edge.Symbol("charge_ub_run_periodic_realization_run_fk")).
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (BillingPeriodicRealizationRun) Indexes() []ent.Index {
	return []ent.Index{
		// deleted_at is left out on purpose: a soft-deleted run must not free its
		// period or the namespace's running slot.
		index.Fields("namespace", "period_to").Unique(),
		index.Fields("realize_after").
			StorageKey("billing_periodic_realization_runs_due").
			Annotations(entsql.IndexWhere("status = 'scheduled' AND deleted_at IS NULL")),
		index.Fields("namespace").
			Unique().
			StorageKey("billing_periodic_realization_runs_one_running").
			Annotations(entsql.IndexWhere("status = 'running'")),
		// Two runs without a previous run would be two separate histories for the
		// namespace. Together with the unique previous_run_id this means a new run
		// can only be appended at the end of the chain. Soft-deleted runs still
		// count: they keep their place in the chain.
		index.Fields("namespace").
			Unique().
			StorageKey("billing_periodic_realization_runs_one_first").
			Annotations(entsql.IndexWhere("previous_run_id IS NULL")),
	}
}

func (BillingPeriodicRealizationRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"billing_periodic_realization_run_realize_after": "realize_after >= period_to",
		}),
	}
}

// BillingPeriodicRealizationRunError is an append-only failure record of a run.
// The charge state only keeps the latest outcome, so this is the failure history.
type BillingPeriodicRealizationRunError struct {
	ent.Schema
}

func (BillingPeriodicRealizationRunError) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingPeriodicRealizationRunError) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			Immutable(),
		// Empty when the run itself failed rather than one of its charges.
		field.String("charge_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			Optional().
			Nillable().
			Immutable(),
		field.String("code").NotEmpty().Immutable(),
		field.Text("reason").NotEmpty().Immutable(),
	}
}

func (BillingPeriodicRealizationRunError) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", BillingPeriodicRealizationRun.Type).
			Ref("errors").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
		edge.From("usage_based_charge", ChargeUsageBased.Type).
			Ref("periodic_realization_errors").
			Field("charge_id").
			Unique().
			Immutable(),
	}
}

func (BillingPeriodicRealizationRunError) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "run_id"),
		index.Fields("charge_id"),
	}
}

// BillingPeriodicRealizationChargeState is the work tracker of periodic
// realization: one row per usage-based charge, stamped for the run that last
// collected it. A run's remaining work is its pending rows.
type BillingPeriodicRealizationChargeState struct {
	ent.Schema
}

func (BillingPeriodicRealizationChargeState) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingPeriodicRealizationChargeState) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			NotEmpty().
			Immutable().
			Unique(),
		// Copied from the charge so pending work can be listed per customer
		// without joining the charge.
		field.String("customer_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			NotEmpty().
			Immutable(),
		field.String("target_run_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			NotEmpty(),
		field.Enum("status").GoType(periodicrealization.ChargeStatus("")),
		// Boundary of the last checkpoint this tracker recorded as done. It can be
		// earlier than the run's period end when the checkpoint was created early.
		field.Time("realized_through").Optional().Nillable(),
	}
}

func (BillingPeriodicRealizationChargeState) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("usage_based_charge", ChargeUsageBased.Type).
			Ref("periodic_realization_state").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("target_run", BillingPeriodicRealizationRun.Type).
			Ref("charge_states").
			Field("target_run_id").
			Unique().
			Required(),
	}
}

func (BillingPeriodicRealizationChargeState) Indexes() []ent.Index {
	return []ent.Index{
		// Serves both "is anything still pending for this run" and listing the
		// customers that have pending work, without reading the rows.
		index.Fields("namespace", "target_run_id", "status", "customer_id").
			StorageKey("billing_periodic_charge_state_run_work"),
	}
}
