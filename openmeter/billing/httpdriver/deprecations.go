package httpdriver

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

// Deprecation handlers for invoice line's rateCard

type invoiceLineRateCardItems struct {
	RateCard *apiv1models.InvoiceUsageBasedRateCard

	// Deprecated fields
	Price      *apiv1models.RateCardUsageBasedPrice
	TaxConfig  *apiv1models.TaxConfig
	FeatureKey *string
}

func (i *invoiceLineRateCardItems) ValidateDeprecatedFields() error {
	var errs []error

	if i.Price == nil {
		// Let's return early here, as we cannot validate the deprecated fields without a price
		return errors.New("price is required")
	}

	priceType, err := i.Price.Discriminator()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to parse price: %w", err))
	}

	if i.FeatureKey == nil && priceType != string(apiv1models.FlatPriceTypeFlat) {
		errs = append(errs, errors.New("featureKey is required"))
	}

	return errors.Join(errs...)
}

func (i *invoiceLineRateCardItems) ValidateRateCard() error {
	if i.RateCard == nil {
		return errors.New("rateCard is required")
	}

	var errs []error

	if i.RateCard.Price == nil {
		// Let's return early here, as we cannot validate the deprecated fields without a price
		return errors.New("price is required")
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
		errs = append(errs, errTaxConfigMismatch)
	}

	priceType, err := i.RateCard.Price.Discriminator()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to parse rateCard.price: %w", err))
	}

	if i.RateCard.FeatureKey == nil && priceType != string(apiv1models.FlatPriceTypeFlat) {
		errs = append(errs, errors.New("featureKey is required"))
	}

	if i.FeatureKey != nil && i.RateCard.FeatureKey != nil && *i.RateCard.FeatureKey != *i.FeatureKey {
		errs = append(errs, errors.New("featureKey must be equal to rateCard.featureKey"))
	}

	return errors.Join(errs...)
}

type invoiceLineRateCardParsed struct {
	Price *productcatalog.Price
	// TaxConfig is the line's raw v1 tax config. Handlers resolve it to a Tax Code reference
	// inside the request's transaction, so it stays unresolved here: mapping it to the domain
	// reference first would drop an unresolved stripe.code.
	TaxConfig  *apiv1models.TaxConfig
	FeatureKey string
	Discounts  productcatalog.Discounts
}

var errTaxConfigMismatch = errors.New("taxConfig must be equal to rateCard.taxConfig")

func mapAndValidateInvoiceLineRateCardDeprecatedFields(in invoiceLineRateCardItems) (*invoiceLineRateCardParsed, error) {
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

		return &invoiceLineRateCardParsed{
			Price:      price,
			TaxConfig:  in.TaxConfig,
			FeatureKey: lo.FromPtr(in.FeatureKey),
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

	var discounts productcatalog.Discounts
	if in.RateCard.Discounts != nil {
		discounts, err = AsDiscounts(in.RateCard.Discounts)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse discounts: %w", err),
			}
		}

		if err := discounts.ValidateForPrice(price); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("invalid rateCard.discounts: %w", err),
			}
		}
	}

	return &invoiceLineRateCardParsed{
		Price:      price,
		TaxConfig:  in.RateCard.TaxConfig,
		FeatureKey: lo.FromPtr(in.RateCard.FeatureKey),
		Discounts:  discounts,
	}, nil
}
