package rating

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ StandardLineAccessor = (*ClampedStandardLineAccessor)(nil)

// ClampedStandardLineAccessor presents negative metered quantities as zero without modifying the wrapped line.
type ClampedStandardLineAccessor struct {
	StandardLineAccessor

	meteredQuantity              *alpacadecimal.Decimal
	meteredPreLinePeriodQuantity *alpacadecimal.Decimal
	validationWarnings           error
}

func NewClampedStandardLineAccessor(in StandardLineAccessor) (ClampedStandardLineAccessor, error) {
	var errs []error

	meteredQuantity, err := in.GetMeteredQuantity()
	if err != nil {
		return ClampedStandardLineAccessor{}, err
	}
	if meteredQuantity != nil && meteredQuantity.IsNegative() {
		meteredQuantity = lo.ToPtr(alpacadecimal.Zero)
		errs = append(errs, billing.WarnNegativeMeteredQuantityClamped)
	}

	meteredPreLinePeriodQuantity, err := in.GetMeteredPreLinePeriodQuantity()
	if err != nil {
		return ClampedStandardLineAccessor{}, err
	}
	if meteredPreLinePeriodQuantity != nil && meteredPreLinePeriodQuantity.IsNegative() {
		meteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
		errs = append(errs, billing.WarnNegativePreLinePeriodMeteredQuantityClamped)
	}

	return ClampedStandardLineAccessor{
		StandardLineAccessor:         in,
		meteredQuantity:              meteredQuantity,
		meteredPreLinePeriodQuantity: meteredPreLinePeriodQuantity,
		validationWarnings:           getValidationWarnings(in, errs),
	}, nil
}

func (a ClampedStandardLineAccessor) GetMeteredQuantity() (*alpacadecimal.Decimal, error) {
	return a.meteredQuantity, nil
}

func (a ClampedStandardLineAccessor) GetMeteredPreLinePeriodQuantity() (*alpacadecimal.Decimal, error) {
	return a.meteredPreLinePeriodQuantity, nil
}

// GetValidationWarnings returns warnings observed while reading quantities through the wrapper.
func (a ClampedStandardLineAccessor) GetValidationWarnings() error {
	return a.validationWarnings
}

func getValidationWarnings(in StandardLineAccessor, errs []error) error {
	warnings := errors.Join(errs...)
	if warnings == nil {
		return nil
	}

	attributes := models.Annotations{}
	meteredQuantity, err := in.GetMeteredQuantity()
	if err != nil {
		return errors.Join(warnings, fmt.Errorf("getting original metered quantity: %w", err))
	}
	if meteredQuantity != nil {
		attributes["original_metered_quantity"] = meteredQuantity.String()
	}

	meteredPreLinePeriodQuantity, err := in.GetMeteredPreLinePeriodQuantity()
	if err != nil {
		return errors.Join(warnings, fmt.Errorf("getting original pre-line metered quantity: %w", err))
	}
	if meteredPreLinePeriodQuantity != nil {
		attributes["original_pre_line_metered_quantity"] = meteredPreLinePeriodQuantity.String()
	}

	return billing.ValidationWithComponent(
		billing.ValidationComponentBillingRating,
		billing.ValidationWithAttributes(attributes, warnings),
	)
}
