// Code generated by ent, DO NOT EDIT.

package db

import (
	"time"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/product"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/schema"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	creditentryMixin := schema.CreditEntry{}.Mixin()
	creditentryMixinFields0 := creditentryMixin[0].Fields()
	_ = creditentryMixinFields0
	creditentryMixinFields1 := creditentryMixin[1].Fields()
	_ = creditentryMixinFields1
	creditentryFields := schema.CreditEntry{}.Fields()
	_ = creditentryFields
	// creditentryDescCreatedAt is the schema descriptor for created_at field.
	creditentryDescCreatedAt := creditentryMixinFields1[0].Descriptor()
	// creditentry.DefaultCreatedAt holds the default value on creation for the created_at field.
	creditentry.DefaultCreatedAt = creditentryDescCreatedAt.Default.(func() time.Time)
	// creditentryDescUpdatedAt is the schema descriptor for updated_at field.
	creditentryDescUpdatedAt := creditentryMixinFields1[1].Descriptor()
	// creditentry.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	creditentry.DefaultUpdatedAt = creditentryDescUpdatedAt.Default.(func() time.Time)
	// creditentry.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	creditentry.UpdateDefaultUpdatedAt = creditentryDescUpdatedAt.UpdateDefault.(func() time.Time)
	// creditentryDescNamespace is the schema descriptor for namespace field.
	creditentryDescNamespace := creditentryFields[0].Descriptor()
	// creditentry.NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	creditentry.NamespaceValidator = creditentryDescNamespace.Validators[0].(func(string) error)
	// creditentryDescSubject is the schema descriptor for subject field.
	creditentryDescSubject := creditentryFields[1].Descriptor()
	// creditentry.SubjectValidator is a validator for the "subject" field. It is called by the builders before save.
	creditentry.SubjectValidator = creditentryDescSubject.Validators[0].(func(string) error)
	// creditentryDescPriority is the schema descriptor for priority field.
	creditentryDescPriority := creditentryFields[6].Descriptor()
	// creditentry.DefaultPriority holds the default value on creation for the priority field.
	creditentry.DefaultPriority = creditentryDescPriority.Default.(uint8)
	// creditentryDescEffectiveAt is the schema descriptor for effective_at field.
	creditentryDescEffectiveAt := creditentryFields[7].Descriptor()
	// creditentry.DefaultEffectiveAt holds the default value on creation for the effective_at field.
	creditentry.DefaultEffectiveAt = creditentryDescEffectiveAt.Default.(func() time.Time)
	// creditentryDescID is the schema descriptor for id field.
	creditentryDescID := creditentryMixinFields0[0].Descriptor()
	// creditentry.DefaultID holds the default value on creation for the id field.
	creditentry.DefaultID = creditentryDescID.Default.(func() string)
	productMixin := schema.Product{}.Mixin()
	productMixinFields0 := productMixin[0].Fields()
	_ = productMixinFields0
	productMixinFields1 := productMixin[1].Fields()
	_ = productMixinFields1
	productFields := schema.Product{}.Fields()
	_ = productFields
	// productDescCreatedAt is the schema descriptor for created_at field.
	productDescCreatedAt := productMixinFields1[0].Descriptor()
	// product.DefaultCreatedAt holds the default value on creation for the created_at field.
	product.DefaultCreatedAt = productDescCreatedAt.Default.(func() time.Time)
	// productDescUpdatedAt is the schema descriptor for updated_at field.
	productDescUpdatedAt := productMixinFields1[1].Descriptor()
	// product.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	product.DefaultUpdatedAt = productDescUpdatedAt.Default.(func() time.Time)
	// product.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	product.UpdateDefaultUpdatedAt = productDescUpdatedAt.UpdateDefault.(func() time.Time)
	// productDescNamespace is the schema descriptor for namespace field.
	productDescNamespace := productFields[0].Descriptor()
	// product.NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	product.NamespaceValidator = productDescNamespace.Validators[0].(func(string) error)
	// productDescName is the schema descriptor for name field.
	productDescName := productFields[1].Descriptor()
	// product.NameValidator is a validator for the "name" field. It is called by the builders before save.
	product.NameValidator = productDescName.Validators[0].(func(string) error)
	// productDescMeterSlug is the schema descriptor for meter_slug field.
	productDescMeterSlug := productFields[2].Descriptor()
	// product.MeterSlugValidator is a validator for the "meter_slug" field. It is called by the builders before save.
	product.MeterSlugValidator = productDescMeterSlug.Validators[0].(func(string) error)
	// productDescArchived is the schema descriptor for archived field.
	productDescArchived := productFields[4].Descriptor()
	// product.DefaultArchived holds the default value on creation for the archived field.
	product.DefaultArchived = productDescArchived.Default.(bool)
	// productDescID is the schema descriptor for id field.
	productDescID := productMixinFields0[0].Descriptor()
	// product.DefaultID holds the default value on creation for the id field.
	product.DefaultID = productDescID.Default.(func() string)
}
