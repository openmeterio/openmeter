package billing

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TaxConfig is the billing-layer tax configuration. It extends productcatalog.TaxConfig with
// TaxCode, the resolved entity snapshot stamped at invoice snapshot time.
type TaxConfig struct {
	productcatalog.TaxConfig
	// TaxCode is the resolved TaxCode entity, stamped at invoice snapshot time.
	TaxCode *taxcode.TaxCode `json:"tax_code,omitempty"`
}

// FromProductCatalog promotes a productcatalog.TaxConfig into a billing.TaxConfig.
// TaxCode is left nil; stamp it after resolving the entity.
func FromProductCatalog(c *productcatalog.TaxConfig) *TaxConfig {
	if c == nil {
		return nil
	}

	return &TaxConfig{
		TaxConfig: c.Clone(),
	}
}

// ToProductCatalog strips the billing-only TaxCode field and returns the intent-level config.
func (c *TaxConfig) ToProductCatalog() *productcatalog.TaxConfig {
	if c == nil {
		return nil
	}

	return lo.ToPtr(c.TaxConfig.Clone())
}

func (c *TaxConfig) Equal(v *TaxConfig) bool {
	if c == nil && v == nil {
		return true
	}

	if c == nil || v == nil {
		return false
	}

	if (c.TaxCode != nil && v.TaxCode == nil) || (c.TaxCode == nil && v.TaxCode != nil) {
		return false
	}

	if c.TaxCode != nil && c.TaxCode.ID != v.TaxCode.ID {
		return false
	}

	return c.TaxConfig.Equal(&v.TaxConfig)
}

func (c *TaxConfig) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if c.Behavior != nil {
		if err := c.Behavior.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.Stripe != nil {
		if err := c.Stripe.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid stripe config: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c TaxConfig) Clone() TaxConfig {
	out := TaxConfig{
		TaxConfig: c.TaxConfig.Clone(),
	}

	if c.TaxCode != nil {
		tc := *c.TaxCode
		tc.AppMappings = slices.Clone(c.TaxCode.AppMappings)
		if c.TaxCode.Description != nil {
			tc.Description = lo.ToPtr(*c.TaxCode.Description)
		}

		out.TaxCode = &tc
	}

	return out
}

// MergeTaxConfigs merges two billing TaxConfigs with overrides taking precedence.
//
// Stripe and TaxCodeID are two encodings of the same intent-level tax-code identity, so they
// merge as a unit: a config that overrides only the Stripe code must not inherit the base's
// (different) TaxCodeID, which would leave the result pointing at two different tax entities.
//
// TaxCode (resolved entity) is intentionally excluded: merge operates on intent-level configs,
// not snapshotted invoice lines.
func MergeTaxConfigs(base, overrides *TaxConfig) *TaxConfig {
	if base != nil && overrides != nil {
		stripe, taxCodeID := base.Stripe, base.TaxCodeID
		if overrides.Stripe != nil || overrides.TaxCodeID != nil {
			stripe, taxCodeID = overrides.Stripe, overrides.TaxCodeID
		}

		return &TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Behavior:  lo.CoalesceOrEmpty(overrides.Behavior, base.Behavior),
				Stripe:    stripe,
				TaxCodeID: taxCodeID,
			},
		}
	}

	if overrides != nil {
		c := overrides.Clone()
		return &c
	}

	if base != nil {
		c := base.Clone()
		return &c
	}

	return nil
}

// taxCodeReferenceIssue classifies a Tax Code reference against resolved Tax Codes:
// nil when the ID exists and is live, ErrTaxCodeNotFound when the namespace has no such row,
// ErrTaxCodeDeleted when it is soft-deleted. The issue message names the ID.
func taxCodeReferenceIssue(id string, resolved productcatalog.TaxCodesByID) error {
	issue := ErrTaxCodeNotFound

	switch tc := resolved[id]; {
	case tc == nil:
	case tc.IsDeleted():
		issue = ErrTaxCodeDeleted
	default:
		return nil
	}

	issue.Message = fmt.Sprintf("%s [tax_code.id=%s]", issue.Message, id)

	return issue
}

// ValidateTaxCodeID rejects a missing or deleted Tax Code. Use it for references the caller
// sent in this request only: a reference copied from stored data (inherited) may point at a Tax
// Code deleted since, and that is fine.
func ValidateTaxCodeID(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace, id string) error {
	if id == "" {
		return nil
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, id)
	if err != nil {
		return fmt.Errorf("resolving tax code %s: %w", id, err)
	}

	return taxCodeReferenceIssue(id, taxCodes)
}

// RequestTaxCodeRef is one Tax Code ID the caller sent in this request, together with the field
// path its validation issues should carry.
type RequestTaxCodeRef struct {
	TaxCodeID string
	// FieldPath locates the reference, e.g. "lines/0/tax_config/tax_code". References with an
	// empty TaxCodeID are skipped, so their FieldPath is never used.
	FieldPath string
}

// ValidateRequestTaxCodeIDsInput carries the Tax Code references the caller sent in one request.
type ValidateRequestTaxCodeIDsInput struct {
	Namespace string
	Resolver  productcatalog.TaxCodeResolver
	// References holds one entry per reference; duplicate IDs are resolved in one
	// batch but each failing reference gets its own issue under its own field path.
	References []RequestTaxCodeRef
}

func (i ValidateRequestTaxCodeIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Resolver == nil {
		errs = append(errs, errors.New("resolver is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ValidateRequestTaxCodeIDs rejects Tax Code references that point at a missing or soft-deleted
// Tax Code: missing ones with ErrTaxCodeNotFound, deleted ones with ErrTaxCodeDeleted, each under
// the reference's field path. Use it for references the caller sent in this request; references
// copied from stored data (inherited) may point at a Tax Code deleted since and must not run
// through this. References with an empty ID are skipped.
func ValidateRequestTaxCodeIDs(ctx context.Context, input ValidateRequestTaxCodeIDsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	ids := make([]string, 0, len(input.References))
	seen := make(map[string]struct{}, len(input.References))
	for _, ref := range input.References {
		if ref.TaxCodeID == "" {
			continue
		}

		if _, ok := seen[ref.TaxCodeID]; ok {
			continue
		}

		seen[ref.TaxCodeID] = struct{}{}
		ids = append(ids, ref.TaxCodeID)
	}

	if len(ids) == 0 {
		return nil
	}

	resolved, err := input.Resolver.BatchResolve(ctx, input.Namespace, ids...)
	if err != nil {
		return fmt.Errorf("resolving request tax codes: %w", err)
	}

	var errs []error
	for _, ref := range input.References {
		if ref.TaxCodeID == "" {
			continue
		}

		if err := taxCodeReferenceIssue(ref.TaxCodeID, resolved); err != nil {
			errs = append(errs, ValidationWithFieldPrefix(ref.FieldPath, err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return ValidationError{Err: err}
	}

	return nil
}
