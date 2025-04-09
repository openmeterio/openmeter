package billingservice

import (
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (s *Service) generateDiscountCorrelationIDs(discounts billing.Discounts) (billing.Discounts, error) {
	return slicesx.MapWithErr(discounts, func(discount billing.Discount) (billing.Discount, error) {
		switch discount.Type() {
		case productcatalog.PercentageDiscountType:
			percentageDiscount, err := discount.AsPercentage()
			if err != nil {
				return discount, err
			}

			if percentageDiscount.CorrelationID == "" {
				percentageDiscount.CorrelationID = ulid.Make().String()
			} else {
				_, err := ulid.Parse(percentageDiscount.CorrelationID)
				if err != nil {
					return discount, fmt.Errorf("invalid correlation ID: %w", err)
				}
			}

			return billing.NewDiscountFrom(percentageDiscount), nil
		case productcatalog.UsageDiscountType:
			usageDiscount, err := discount.AsUsage()
			if err != nil {
				return discount, err
			}

			if usageDiscount.CorrelationID == "" {
				usageDiscount.CorrelationID = ulid.Make().String()
			} else {
				_, err := ulid.Parse(usageDiscount.CorrelationID)
				if err != nil {
					return discount, fmt.Errorf("invalid correlation ID: %w", err)
				}
			}

			return billing.NewDiscountFrom(usageDiscount), nil
		default:
			return discount, fmt.Errorf("unsupported discount type: %s", discount.Type())
		}
	})
}
