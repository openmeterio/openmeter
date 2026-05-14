package invoicedusage

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Mixin = entutils.RecursiveMixin[mixin]

type mixin struct {
	ent.Schema
}

func (mixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
		totals.Mixin{},
	}
}

func (mixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("service_period_from"),
		field.Time("service_period_to"),
		field.String("ledger_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),
	}
}

type Creator[T any] interface {
	entutils.NamespaceMixinCreator[T]
	entutils.IDMixinCreator[T]
	entutils.TimeMixinCreator[T]
	entutils.AnnotationsMixinSetter[T]
	totals.Setter[T]
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetNillableLedgerTransactionGroupID(ledgerTransactionGroupID *string) T
}

func Create[T Creator[T]](creator T, ns string, invoicedUsage AccruedUsage) T {
	var trnsGroupID *string
	if invoicedUsage.LedgerTransaction != nil {
		trnsGroupID = &invoicedUsage.LedgerTransaction.TransactionGroupID
	}

	creator = creator.SetAnnotations(invoicedUsage.Annotations).
		SetNamespace(ns).
		SetServicePeriodFrom(invoicedUsage.ServicePeriod.From.In(time.UTC)).
		SetServicePeriodTo(invoicedUsage.ServicePeriod.To.In(time.UTC)).
		SetNillableLedgerTransactionGroupID(trnsGroupID)

	creator = totals.Set(creator, invoicedUsage.Totals)

	return creator
}

type Getter interface {
	entutils.TimeMixinGetter
	entutils.NamespaceMixinGetter
	entutils.IDMixinGetter
	entutils.AnnotationsMixinGetter
	totals.TotalsGetter
	GetServicePeriodFrom() time.Time
	GetServicePeriodTo() time.Time
	GetLedgerTransactionGroupID() *string
}

func MapAccruedUsageFromDB(dbEntity Getter) AccruedUsage {
	var ledgerTransaction *ledgertransaction.GroupReference
	if dbEntity.GetLedgerTransactionGroupID() != nil {
		ledgerTransaction = &ledgertransaction.GroupReference{
			TransactionGroupID: *dbEntity.GetLedgerTransactionGroupID(),
		}
	}

	return AccruedUsage{
		NamespacedID: models.NamespacedID{
			Namespace: dbEntity.GetNamespace(),
			ID:        dbEntity.GetID(),
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),
		Annotations:  dbEntity.GetAnnotations(),
		ServicePeriod: timeutil.ClosedPeriod{
			From: dbEntity.GetServicePeriodFrom().In(time.UTC),
			To:   dbEntity.GetServicePeriodTo().In(time.UTC),
		},
		LedgerTransaction: ledgerTransaction,
		Totals:            totals.FromDB(dbEntity),
	}
}
