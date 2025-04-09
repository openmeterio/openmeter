package httpdriver

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

// Deprecation handlers for flat fee line's rateCard

type flatFeeRateCardItems struct {
	RateCard *api.InvoiceFlatFeeRateCard

	// Deprecated fields
	PerUnitAmount *string
	PaymentTerm   *api.PricePaymentTerm
	Quantity      *string
	TaxConfig     *api.TaxConfig
}

func (i *flatFeeRateCardItems) ValidateDeprecatedFields() error {
	var errs []error

	if i.PerUnitAmount == nil {
		errs = append(errs, errors.New("perUnitAmount is required"))
	}

	if i.Quantity == nil {
		errs = append(errs, errors.New("quantity is required"))
	}

	return errors.Join(errs...)
}

func (i *flatFeeRateCardItems) ValidateRateCard() error {
	if i.RateCard == nil {
		return errors.New("rateCard is required")
	}

	var errs []error

	// Price validations

	if i.RateCard.Price == nil {
		return errors.New("rateCard.price is required")
	}

	if i.RateCard.Price.Type != api.FlatPriceWithPaymentTermTypeFlat {
		errs = append(errs, errors.New("rateCard.price.type must be flat"))
	}

	// Deprecated fields vs rateCard.price validations
	if i.PerUnitAmount != nil && i.RateCard.Price.Amount != *i.PerUnitAmount {
		errs = append(errs, errors.New("rateCard.price.amount must be equal to perUnitAmount"))
	}

	rateCardPaymentTerm := lo.FromPtrOr(i.RateCard.Price.PaymentTerm, api.PricePaymentTerm(defaultFlatFeePaymentTerm))
	if i.PaymentTerm != nil && rateCardPaymentTerm != *i.PaymentTerm {
		errs = append(errs, errors.New("rateCard.price.paymentTerm must be equal to paymentTerm"))
	}

	rateCardQuantityString := lo.FromPtrOr(i.RateCard.Quantity, defaultFlatFeeQuantity)
	rateCardQuantity, err := alpacadecimal.NewFromString(rateCardQuantityString)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to parse rateCard.quantity: %w", err))
	}

	if i.Quantity != nil {
		deprecatedQuantity, err := alpacadecimal.NewFromString(*i.Quantity)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse quantity: %w", err))
		}

		if !deprecatedQuantity.Equal(rateCardQuantity) {
			errs = append(errs, errors.New("quantity must be equal to rateCard.quantity"))
		}
	}

	// TaxConfig validations

	if i.TaxConfig != nil && i.RateCard.TaxConfig != nil && !reflect.DeepEqual(*i.TaxConfig, *i.RateCard.TaxConfig) {
		errs = append(errs, errors.New("taxConfig must be equal to rateCard.taxConfig"))
	}

	return errors.Join(errs...)
}

type flatFeeRateCardParsed struct {
	PerUnitAmount alpacadecimal.Decimal
	PaymentTerm   productcatalog.PaymentTermType
	Quantity      alpacadecimal.Decimal
	TaxConfig     *billing.TaxConfig
	Discounts     billing.Discounts
}

func mapAndValidateFlatFeeRateCardDeprecatedFields(in flatFeeRateCardItems) (*flatFeeRateCardParsed, error) {
	if in.RateCard == nil {
		// No rate card, so let's use the deprecated fields

		if err := in.ValidateDeprecatedFields(); err != nil {
			return nil, billing.ValidationError{
				Err: err,
			}
		}

		perUnitAmount, err := alpacadecimal.NewFromString(*in.PerUnitAmount)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse perUnitAmount: %w", err),
			}
		}

		quantity, err := alpacadecimal.NewFromString(*in.Quantity)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse quantity: %w", err),
			}
		}

		var taxConfig *billing.TaxConfig
		if in.TaxConfig != nil {
			taxConfig = mapTaxConfigToEntity(in.TaxConfig)
		}

		return &flatFeeRateCardParsed{
			PerUnitAmount: perUnitAmount,
			PaymentTerm:   lo.FromPtrOr((*productcatalog.PaymentTermType)(in.PaymentTerm), productcatalog.InAdvancePaymentTerm),
			Quantity:      quantity,
			TaxConfig:     taxConfig,
		}, nil
	}

	// Rate card is set, so we should rely on that
	if err := in.ValidateRateCard(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	var taxConfig *billing.TaxConfig
	if in.TaxConfig != nil {
		taxConfig = mapTaxConfigToEntity(in.TaxConfig)
	}

	perUnitAmount, err := alpacadecimal.NewFromString(in.RateCard.Price.Amount)
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("failed to parse perUnitAmount: %w", err),
		}
	}

	paymentTerm := lo.FromPtrOr(in.RateCard.Price.PaymentTerm, api.PricePaymentTerm(defaultFlatFeePaymentTerm))

	qty, err := alpacadecimal.NewFromString(lo.FromPtrOr(in.RateCard.Quantity, defaultFlatFeeQuantity))
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("failed to parse quantity: %w", err),
		}
	}

	var discounts billing.Discounts
	if in.RateCard.Discounts != nil {
		discounts = lo.Map(*in.RateCard.Discounts, func(d api.BillingDiscountPercentage, _ int) billing.Discount {
			return billing.NewDiscountFrom(AsPercentageDiscount(d))
		})
	}

	return &flatFeeRateCardParsed{
		PerUnitAmount: perUnitAmount,
		PaymentTerm:   productcatalog.PaymentTermType(paymentTerm),
		Quantity:      qty,
		TaxConfig:     taxConfig,
		Discounts:     discounts,
	}, nil
}

// Deprecation handlers for usage based line's rateCard

type usageBasedRateCardItems struct {
	RateCard *api.InvoiceUsageBasedRateCard

	// Deprecated fields
	Price      *api.RateCardUsageBasedPrice
	TaxConfig  *api.TaxConfig
	FeatureKey *string
}

func (i *usageBasedRateCardItems) ValidateDeprecatedFields() error {
	var errs []error

	if i.Price == nil {
		errs = append(errs, errors.New("price is required"))
	}

	if i.FeatureKey == nil {
		errs = append(errs, errors.New("featureKey is required"))
	}

	return errors.Join(errs...)
}

func (i *usageBasedRateCardItems) ValidateRateCard() error {
	if i.RateCard == nil {
		return errors.New("rateCard is required")
	}

	var errs []error

	if i.RateCard.Price == nil {
		errs = append(errs, errors.New("price is required"))
	}

	if i.Price != nil && i.RateCard.Price != nil {
		deprecatedPrice, err := productcataloghttp.AsPrice(*i.Price)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse price: %w", err))
		}

		rateCardPrice, err := productcataloghttp.AsPrice(*i.RateCard.Price)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse rateCard.price: %w", err))
		}

		if !rateCardPrice.Equal(deprecatedPrice) {
			errs = append(errs, errors.New("price and rateCard.price must be equal"))
		}
	}

	if i.TaxConfig != nil && i.RateCard.TaxConfig != nil && !reflect.DeepEqual(*i.TaxConfig, *i.RateCard.TaxConfig) {
		errs = append(errs, errors.New("taxConfig must be equal to rateCard.taxConfig"))
	}

	if i.RateCard.FeatureKey == nil {
		errs = append(errs, errors.New("featureKey is required"))
	}

	if i.FeatureKey != nil && i.RateCard.FeatureKey != nil && *i.RateCard.FeatureKey != *i.FeatureKey {
		errs = append(errs, errors.New("featureKey must be equal to rateCard.featureKey"))
	}

	return errors.Join(errs...)
}

type usageBasedRateCardParsed struct {
	Price      *productcatalog.Price
	TaxConfig  *billing.TaxConfig
	FeatureKey string
	Discounts  billing.Discounts
}

func mapAndValidateUsageBasedRateCardDeprecatedFields(in usageBasedRateCardItems) (*usageBasedRateCardParsed, error) {
	if in.RateCard == nil {
		// No rate card, so let's use the deprecated fields

		if err := in.ValidateDeprecatedFields(); err != nil {
			return nil, billing.ValidationError{
				Err: err,
			}
		}

		price, err := productcataloghttp.AsPrice(*in.Price)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse price: %w", err),
			}
		}

		return &usageBasedRateCardParsed{
			Price:      price,
			TaxConfig:  mapTaxConfigToEntity(in.TaxConfig),
			FeatureKey: *in.FeatureKey,
		}, nil
	}

	// Rate card is set, so let's use that

	if err := in.ValidateRateCard(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	price, err := productcataloghttp.AsPrice(*in.RateCard.Price)
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("failed to parse rateCard.price: %w", err),
		}
	}

	var discounts billing.Discounts
	if in.RateCard.Discounts != nil {
		discounts, err = AsDiscounts(*in.RateCard.Discounts)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse discounts: %w", err),
			}
		}
	}

	return &usageBasedRateCardParsed{
		Price:      price,
		TaxConfig:  mapTaxConfigToEntity(in.RateCard.TaxConfig),
		FeatureKey: *in.RateCard.FeatureKey,
		Discounts:  discounts,
	}, nil
}
