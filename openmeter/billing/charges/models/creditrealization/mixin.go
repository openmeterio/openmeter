package creditrealization

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Mixin holds the columns every credit realization table has. The pointer to
// the realization a correction corrects is not one of them: tables that need it
// add CorrectionMixin.
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
	}
}

// CorrectionMixin lets a correction name the allocation it corrects. It is
// separate from Mixin because not every credit realization table can point at
// the corrected row.
type CorrectionMixin struct {
	mixin.Schema

	SelfReferenceType   any
	SelfReferenceSymbol string
}

func (CorrectionMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("corrects_realization_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),
	}
}

func (m CorrectionMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("allocation", m.SelfReferenceType).
			Field("corrects_realization_id").
			Unique().
			StorageKey(edge.Symbol(m.SelfReferenceSymbol)),
		edge.From("corrections", m.SelfReferenceType).
			Ref("allocation"),
	}
}

// MixinCreator and MixinGetter cover the columns of Mixin. Creator and Getter
// add the column of CorrectionMixin.
type MixinCreator[T any] interface {
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
	SetSortHint(sortHint int) T
}

type Creator[T any] interface {
	MixinCreator[T]
	SetNillableCorrectsRealizationID(correctsRealizationID *string) T
}

func CreateMixin[T MixinCreator[T]](creator MixinCreator[T], ns string, sortHint int, realization CreateInput) T {
	create := creator.SetAnnotations(realization.Annotations).
		SetNamespace(ns).
		SetNillableLineID(realization.LineID).
		SetAmount(realization.Amount).
		SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
		SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
		SetLedgerTransactionGroupID(realization.LedgerTransaction.TransactionGroupID).
		SetSortHint(sortHint).
		SetType(realization.Type)

	if realization.ID != "" {
		create = create.SetID(realization.ID)
	}

	return create
}

func Create[T Creator[T]](creator Creator[T], ns string, sortHint int, realization CreateInput) T {
	return CreateMixin[T](creator, ns, sortHint, realization).
		SetNillableCorrectsRealizationID(realization.CorrectsRealizationID)
}

type MixinGetter interface {
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
}

type Getter interface {
	MixinGetter
	GetCorrectsRealizationID() *string
}

func FromDBMixinRealization(dbEntity MixinGetter) Realization {
	return Realization{
		NamespacedModel: models.NamespacedModel{
			Namespace: dbEntity.GetNamespace(),
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),

		CreateInput: CreateInput{
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
			LineID: dbEntity.GetLineID(),
			Type:   dbEntity.GetType(),
		},
		SortHint: dbEntity.GetSortHint(),
	}
}

func FromDBMixinRealizations[T MixinGetter](dbEntities []T) Realizations {
	realizations := make(Realizations, 0, len(dbEntities))
	for _, dbEntity := range dbEntities {
		realizations = append(realizations, FromDBMixinRealization(dbEntity))
	}

	return realizations
}

func FromDBRealization(dbEntity Getter) Realization {
	realization := FromDBMixinRealization(dbEntity)
	realization.CorrectsRealizationID = dbEntity.GetCorrectsRealizationID()

	return realization
}

func FromDBRealizations[T Getter](dbEntities []T) Realizations {
	realizations := make(Realizations, 0, len(dbEntities))
	for _, dbEntity := range dbEntities {
		realizations = append(realizations, FromDBRealization(dbEntity))
	}

	return realizations
}

func FromDBRealizationsOrErr[T Getter](dbEntities []T, err error) (Realizations, error) {
	if err != nil {
		return nil, err
	}

	return FromDBRealizations(dbEntities), nil
}
