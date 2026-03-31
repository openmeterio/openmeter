package creditrealization

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Mixin = entutils.RecursiveMixin[mixinBase]

type mixinBase struct {
	mixin.Schema
}

func (mixinBase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (m mixinBase) Fields() []ent.Field {
	return []ent.Field{
		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Time("service_period_from"),
		field.Time("service_period_to"),

		field.String("ledger_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),

		field.Int("sort_hint"),

		field.Enum("type").
			GoType(Type("")).
			Immutable(),

		field.String("corrects_realization_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),
	}
}

func (mixinBase) Edges() []ent.Edge {
	return []ent.Edge{
		// TODO: add edge to the correction entry
		// edge.To("corrects_realization", Realization.Type),
	}
}

type Creator[T any] interface {
	SetID(id string) T
	SetNamespace(namespace string) T
	SetAnnotations(annotations models.Annotations) T
	SetLineID(lineID string) T
	SetNillableLineID(lineID *string) T
	SetAmount(amount alpacadecimal.Decimal) T
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetLedgerTransactionGroupID(ledgerTransactionGroupID string) T
	SetType(t Type) T
	SetNillableCorrectsRealizationID(correctsRealizationID *string) T
	SetSortHint(sortHint int) T
}

func Create[T Creator[T]](creator Creator[T], ns string, sortHint int, realization AdapterCreateInput) T {
	create := creator.SetAnnotations(realization.Annotations).
		SetNamespace(ns).
		SetNillableLineID(realization.LineID).
		SetAmount(realization.Amount).
		SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
		SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
		SetLedgerTransactionGroupID(realization.LedgerTransaction.TransactionGroupID).
		SetSortHint(sortHint).
		SetType(realization.Type).
		SetNillableCorrectsRealizationID(realization.CorrectsRealizationID)

	if realization.ID != "" {
		create = create.SetID(realization.ID)
	}

	return create
}

type Getter interface {
	entutils.TimeMixinGetter
	entutils.NamespaceMixinGetter
	entutils.IDMixinGetter
	entutils.AnnotationsMixinGetter

	GetLineID() *string
	GetAmount() alpacadecimal.Decimal
	GetServicePeriodFrom() time.Time
	GetServicePeriodTo() time.Time
	GetLedgerTransactionGroupID() string
	GetSortHint() int
	GetType() Type
	GetCorrectsRealizationID() *string
}

func MapFromDB(dbEntity Getter) Realization {
	return Realization{
		NamespacedModel: models.NamespacedModel{
			Namespace: dbEntity.GetNamespace(),
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),

		AdapterCreateInput: AdapterCreateInput{
			ID:          dbEntity.GetID(),
			Annotations: dbEntity.GetAnnotations(),

			ServicePeriod: timeutil.ClosedPeriod{
				From: dbEntity.GetServicePeriodFrom().In(time.UTC),
				To:   dbEntity.GetServicePeriodTo().In(time.UTC),
			},
			Amount: dbEntity.GetAmount(),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: dbEntity.GetLedgerTransactionGroupID(),
			},
			LineID:                dbEntity.GetLineID(),
			Type:                  dbEntity.GetType(),
			CorrectsRealizationID: dbEntity.GetCorrectsRealizationID(),
		},
		SortHint: dbEntity.GetSortHint(),
	}
}
