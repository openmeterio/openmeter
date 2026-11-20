package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// MergeTaxConfigs merges two line tax configs with overrides taking precedence:
// the override's Tax Code ID wins when non-empty, and the behavior coalesces
// override → base. A nil override yields a clone of base and vice versa.
func MergeTaxConfigs(base, overrides *productcatalog.TaxCodeConfig) *productcatalog.TaxCodeConfig {
	if base != nil && overrides != nil {
		out := overrides.Clone()

		if out.TaxCodeID == "" {
			out.TaxCodeID = base.TaxCodeID
		}

		if out.Behavior == nil {
			out.Behavior = base.Behavior
		}

		return lo.ToPtr(out)
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

// authoredTaxCodeIssue classifies an authored Tax Code reference against resolved Tax Codes:
// nil when the ID exists and is live, ErrTaxCodeNotFound when the namespace has no such row,
// ErrTaxCodeDeleted when it is soft-deleted. The issue message names the ID.
func authoredTaxCodeIssue(id string, resolved map[string]*taxcode.TaxCode) error {
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

// ValidateTaxCodeID rejects a missing or deleted Tax Code. Use it for authored references only:
// a reference copied from stored data may point at a Tax Code deleted since, and that is fine.
func ValidateTaxCodeID(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace, id string) error {
	if id == "" {
		return nil
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, id)
	if err != nil {
		return fmt.Errorf("resolving tax code %s: %w", id, err)
	}

	return authoredTaxCodeIssue(id, taxCodes)
}

// AuthoredTaxCodeID is one Tax Code ID an API caller authored, together with the field path
// its validation issues should carry.
type AuthoredTaxCodeID struct {
	TaxCodeID string
	// FieldPath locates the reference, e.g. "lines/0/tax_config/tax_code". References with an
	// empty TaxCodeID are skipped, so their FieldPath is never used.
	FieldPath string
}

// ValidateAuthoredTaxCodeIDsInput carries the authored Tax Code references of one request.
type ValidateAuthoredTaxCodeIDsInput struct {
	Namespace string
	Resolver  productcatalog.TaxCodeResolver
	// References holds one entry per authored reference; duplicate IDs are resolved in one
	// batch but each failing reference gets its own issue under its own field path.
	References []AuthoredTaxCodeID
}

func (i ValidateAuthoredTaxCodeIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Resolver == nil {
		errs = append(errs, errors.New("resolver is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ValidateAuthoredTaxCodeIDs rejects authored Tax Code references that point at a missing or
// soft-deleted Tax Code: missing ones with ErrTaxCodeNotFound, deleted ones with
// ErrTaxCodeDeleted, each under the reference's field path. Use it for references an API caller
// authored in the current request; references copied from stored data may point at a Tax Code
// deleted since and must not run through this.
func ValidateAuthoredTaxCodeIDs(ctx context.Context, input ValidateAuthoredTaxCodeIDsInput) error {
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
		return fmt.Errorf("resolving authored tax codes: %w", err)
	}

	var errs []error
	for _, ref := range input.References {
		if err := authoredTaxCodeIssue(ref.TaxCodeID, resolved); err != nil {
			errs = append(errs, ValidationWithFieldPrefix(ref.FieldPath, err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return ValidationError{Err: err}
	}

	return nil
}
