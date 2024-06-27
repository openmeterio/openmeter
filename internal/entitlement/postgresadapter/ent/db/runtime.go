// Code generated by ent, DO NOT EDIT.

package db

import (
	"time"

	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/usagereset"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/schema"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	entitlementMixin := schema.Entitlement{}.Mixin()
	entitlementMixinFields0 := entitlementMixin[0].Fields()
	_ = entitlementMixinFields0
	entitlementMixinFields1 := entitlementMixin[1].Fields()
	_ = entitlementMixinFields1
	entitlementMixinFields3 := entitlementMixin[3].Fields()
	_ = entitlementMixinFields3
	entitlementFields := schema.Entitlement{}.Fields()
	_ = entitlementFields
	// entitlementDescNamespace is the schema descriptor for namespace field.
	entitlementDescNamespace := entitlementMixinFields1[0].Descriptor()
	// entitlement.NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	entitlement.NamespaceValidator = entitlementDescNamespace.Validators[0].(func(string) error)
	// entitlementDescCreatedAt is the schema descriptor for created_at field.
	entitlementDescCreatedAt := entitlementMixinFields3[0].Descriptor()
	// entitlement.DefaultCreatedAt holds the default value on creation for the created_at field.
	entitlement.DefaultCreatedAt = entitlementDescCreatedAt.Default.(func() time.Time)
	// entitlementDescUpdatedAt is the schema descriptor for updated_at field.
	entitlementDescUpdatedAt := entitlementMixinFields3[1].Descriptor()
	// entitlement.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	entitlement.DefaultUpdatedAt = entitlementDescUpdatedAt.Default.(func() time.Time)
	// entitlement.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	entitlement.UpdateDefaultUpdatedAt = entitlementDescUpdatedAt.UpdateDefault.(func() time.Time)
	// entitlementDescFeatureKey is the schema descriptor for feature_key field.
	entitlementDescFeatureKey := entitlementFields[2].Descriptor()
	// entitlement.FeatureKeyValidator is a validator for the "feature_key" field. It is called by the builders before save.
	entitlement.FeatureKeyValidator = entitlementDescFeatureKey.Validators[0].(func(string) error)
	// entitlementDescSubjectKey is the schema descriptor for subject_key field.
	entitlementDescSubjectKey := entitlementFields[3].Descriptor()
	// entitlement.SubjectKeyValidator is a validator for the "subject_key" field. It is called by the builders before save.
	entitlement.SubjectKeyValidator = entitlementDescSubjectKey.Validators[0].(func(string) error)
	// entitlementDescID is the schema descriptor for id field.
	entitlementDescID := entitlementMixinFields0[0].Descriptor()
	// entitlement.DefaultID holds the default value on creation for the id field.
	entitlement.DefaultID = entitlementDescID.Default.(func() string)
	usageresetMixin := schema.UsageReset{}.Mixin()
	usageresetMixinFields0 := usageresetMixin[0].Fields()
	_ = usageresetMixinFields0
	usageresetMixinFields1 := usageresetMixin[1].Fields()
	_ = usageresetMixinFields1
	usageresetMixinFields2 := usageresetMixin[2].Fields()
	_ = usageresetMixinFields2
	usageresetFields := schema.UsageReset{}.Fields()
	_ = usageresetFields
	// usageresetDescNamespace is the schema descriptor for namespace field.
	usageresetDescNamespace := usageresetMixinFields1[0].Descriptor()
	// usagereset.NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	usagereset.NamespaceValidator = usageresetDescNamespace.Validators[0].(func(string) error)
	// usageresetDescCreatedAt is the schema descriptor for created_at field.
	usageresetDescCreatedAt := usageresetMixinFields2[0].Descriptor()
	// usagereset.DefaultCreatedAt holds the default value on creation for the created_at field.
	usagereset.DefaultCreatedAt = usageresetDescCreatedAt.Default.(func() time.Time)
	// usageresetDescUpdatedAt is the schema descriptor for updated_at field.
	usageresetDescUpdatedAt := usageresetMixinFields2[1].Descriptor()
	// usagereset.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	usagereset.DefaultUpdatedAt = usageresetDescUpdatedAt.Default.(func() time.Time)
	// usagereset.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	usagereset.UpdateDefaultUpdatedAt = usageresetDescUpdatedAt.UpdateDefault.(func() time.Time)
	// usageresetDescID is the schema descriptor for id field.
	usageresetDescID := usageresetMixinFields0[0].Descriptor()
	// usagereset.DefaultID holds the default value on creation for the id field.
	usagereset.DefaultID = usageresetDescID.Default.(func() string)
}
