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
	}
}

type Creator[T any] interface {
	SetNamespace(namespace string) T
	SetAnnotations(annotations models.Annotations) T
	SetLineID(lineID string) T
	SetNillableLineID(lineID *string) T
	SetAmount(amount alpacadecimal.Decimal) T
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetLedgerTransactionGroupID(ledgerTransactionGroupID string) T
}

func Create[T Creator[T]](creator Creator[T], ns string, realization CreateInput) T {
	return creator.SetAnnotations(realization.Annotations).
		SetNamespace(ns).
		SetNillableLineID(realization.LineID).
		SetAmount(realization.Amount).
		SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
		SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
		SetLedgerTransactionGroupID(realization.LedgerTransaction.TransactionGroupID)
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
}

func MapFromDB(dbEntity Getter) Realization {
	return Realization{
		NamespacedID: models.NamespacedID{
			Namespace: dbEntity.GetNamespace(),
			ID:        dbEntity.GetID(),
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),

		CreateInput: CreateInput{
			Annotations: dbEntity.GetAnnotations(),

			ServicePeriod: timeutil.ClosedPeriod{
				From: dbEntity.GetServicePeriodFrom().In(time.UTC),
				To:   dbEntity.GetServicePeriodTo().In(time.UTC),
			},
			Amount: dbEntity.GetAmount(),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: dbEntity.GetLedgerTransactionGroupID(),
			},
			LineID: dbEntity.GetLineID(),
		},
	}
}
