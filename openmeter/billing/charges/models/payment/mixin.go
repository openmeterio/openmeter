package payment

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/convert"
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
	}
}

func (mixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("service_period_from"),
		field.Time("service_period_to"),

		field.Enum("status").
			GoType(Status("")),

		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		// TODO: Let's add edges to ledger
		field.String("authorized_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Time("authorized_at").Optional().Nillable(),

		field.String("settled_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Time("settled_at").Optional().Nillable(),
	}
}

type MutableFieldSetter[T any] interface {
	SetAmount(amount alpacadecimal.Decimal) T
	SetStatus(status Status) T
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetNillableAuthorizedTransactionGroupID(authorizedTransactionGroupID *string) T
	SetNillableAuthorizedAt(authorizedAt *time.Time) T
	SetNillableSettledTransactionGroupID(settledTransactionGroupID *string) T
	SetNillableSettledAt(settledAt *time.Time) T
}

type Creator[T any] interface {
	entutils.NamespaceMixinCreator[T]
	entutils.AnnotationsMixinSetter[T]
	entutils.TimeMixinCreator[T]
	MutableFieldSetter[T]
}

func Create[T Creator[T]](creator Creator[T], namespace string, paymentSettlement Base) T {
	return creator.SetAnnotations(paymentSettlement.Annotations).
		SetNamespace(namespace).
		SetServicePeriodFrom(paymentSettlement.ServicePeriod.From).
		SetServicePeriodTo(paymentSettlement.ServicePeriod.To).
		SetAmount(paymentSettlement.Amount).
		SetStatus(paymentSettlement.Status).
		SetNillableAuthorizedTransactionGroupID(paymentSettlement.Authorized.GetIDOrNull()).
		SetNillableAuthorizedAt(paymentSettlement.Authorized.GetTimeOrNull()).
		SetNillableSettledTransactionGroupID(paymentSettlement.Settled.GetIDOrNull()).
		SetNillableSettledAt(paymentSettlement.Settled.GetTimeOrNull())
}

type Updater[T any] interface {
	entutils.AnnotationsMixinSetter[T]
	entutils.TimeMixinUpdater[T]
	MutableFieldSetter[T]
}

func Update[T Updater[T]](updater Updater[T], in Payment) T {
	return updater.SetAnnotations(in.Annotations).
		SetServicePeriodFrom(in.ServicePeriod.From).
		SetServicePeriodTo(in.ServicePeriod.To).
		SetAmount(in.Amount).
		SetStatus(in.Status).
		SetNillableDeletedAt(convert.TimePtrIn(in.DeletedAt, time.UTC)).
		SetNillableAuthorizedTransactionGroupID(in.Authorized.GetIDOrNull()).
		SetNillableAuthorizedAt(convert.TimePtrIn(in.Authorized.GetTimeOrNull(), time.UTC)).
		SetNillableSettledTransactionGroupID(in.Settled.GetIDOrNull()).
		SetNillableSettledAt(convert.TimePtrIn(in.Settled.GetTimeOrNull(), time.UTC))
}

type Getter interface {
	entutils.NamespaceMixinGetter
	entutils.IDMixinGetter
	entutils.TimeMixinGetter
	entutils.AnnotationsMixinGetter
	GetServicePeriodFrom() time.Time
	GetServicePeriodTo() time.Time
	GetAmount() alpacadecimal.Decimal
	GetStatus() Status
	GetAuthorizedTransactionGroupID() *string
	GetAuthorizedAt() *time.Time
	GetSettledTransactionGroupID() *string
	GetSettledAt() *time.Time
}

func mapBaseFromDB(dbEntity Getter) Base {
	return Base{
		Annotations: dbEntity.GetAnnotations(),
		ServicePeriod: timeutil.ClosedPeriod{
			From: dbEntity.GetServicePeriodFrom().In(time.UTC),
			To:   dbEntity.GetServicePeriodTo().In(time.UTC),
		},
		Status:     dbEntity.GetStatus(),
		Amount:     dbEntity.GetAmount(),
		Authorized: mapTimedLedgerTransactionGroupReferenceFromDB(dbEntity.GetAuthorizedTransactionGroupID(), dbEntity.GetAuthorizedAt()),
		Settled:    mapTimedLedgerTransactionGroupReferenceFromDB(dbEntity.GetSettledTransactionGroupID(), dbEntity.GetSettledAt()),
	}
}

func mapPaymentFromDB(dbEntity Getter) Payment {
	return Payment{
		NamespacedID: models.NamespacedID{
			Namespace: dbEntity.GetNamespace(),
			ID:        dbEntity.GetID(),
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),
		Base:         mapBaseFromDB(dbEntity),
	}
}

func mapTimedLedgerTransactionGroupReferenceFromDB(reference *string, at *time.Time) *ledgertransaction.TimedGroupReference {
	if reference == nil || at == nil {
		return nil
	}

	return &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{
			TransactionGroupID: *reference,
		},
		Time: at.In(time.UTC),
	}
}
