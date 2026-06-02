package addons

import (
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
	currency "github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ToAPILabels(source addon.Addon) *apiv3.Labels {
	return labels.FromMetadataAnnotations(source.Metadata, source.Annotations)
}

func ToAPIAddonStatus(source addon.Addon) (apiv3.AddonStatus, error) {
	switch source.Status() {
	case productcatalog.AddonStatusDraft:
		return apiv3.AddonStatusDraft, nil
	case productcatalog.AddonStatusActive:
		return apiv3.AddonStatusActive, nil
	case productcatalog.AddonStatusArchived:
		return apiv3.AddonStatusArchived, nil
	default:
		return "", fmt.Errorf("invalid add-on status: %s", source.Status())
	}
}

func ToAPIValidationAttributes(attrs models.Attributes) *map[string]any {
	if len(attrs) == 0 {
		return nil
	}

	out := attrs.AsStringMap()

	if len(out) == 0 {
		return nil
	}

	return &out
}

func ToAPIProductCatalogValidationErrors(source addon.Addon) (*[]apiv3.ProductCatalogValidationError, error) {
	issues, err := source.AsProductCatalogAddon().ValidationErrors()
	if err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		return nil, nil
	}

	var result []apiv3.ProductCatalogValidationError

	for _, issue := range issues {
		result = append(result, apiv3.ProductCatalogValidationError{
			Message:    issue.Message(),
			Field:      issue.Field().JSONPath(),
			Code:       string(issue.Code()),
			Attributes: ToAPIValidationAttributes(issue.Attributes()),
		})
	}

	return &result, nil
}

func ToAPIAddonInstanceType(source productcatalog.AddonInstanceType) (apiv3.AddonInstanceType, error) {
	switch source {
	case productcatalog.AddonInstanceTypeMultiple:
		return apiv3.AddonInstanceTypeMultiple, nil
	case productcatalog.AddonInstanceTypeSingle:
		return apiv3.AddonInstanceTypeSingle, nil
	default:
		return "", fmt.Errorf("unexpected enum element: %v", source)
	}
}

func FromAPIAddonInstanceType(source apiv3.AddonInstanceType) (productcatalog.AddonInstanceType, error) {
	switch source {
	case apiv3.AddonInstanceTypeMultiple:
		return productcatalog.AddonInstanceTypeMultiple, nil
	case apiv3.AddonInstanceTypeSingle:
		return productcatalog.AddonInstanceTypeSingle, nil
	default:
		return "", fmt.Errorf("unexpected enum element: %v", source)
	}
}

// ToAPIAddon converts a domain Addon to the v3 API representation.
func ToAPIAddon(source addon.Addon) (apiv3.Addon, error) {
	var result apiv3.Addon

	result.CreatedAt = source.ManagedModel.CreatedAt
	result.Currency = string(source.AddonMeta.Currency)
	result.DeletedAt = source.ManagedModel.DeletedAt
	result.Description = source.AddonMeta.Description
	result.EffectiveFrom = source.AddonMeta.EffectivePeriod.EffectiveFrom
	result.EffectiveTo = source.AddonMeta.EffectivePeriod.EffectiveTo
	result.Id = source.NamespacedID.ID

	instanceType, err := ToAPIAddonInstanceType(source.AddonMeta.InstanceType)
	if err != nil {
		return result, err
	}
	result.InstanceType = instanceType

	result.Key = source.AddonMeta.Key
	result.Labels = ToAPILabels(source)
	result.Name = source.AddonMeta.Name

	status, err := ToAPIAddonStatus(source)
	if err != nil {
		return result, err
	}
	result.Status = status

	result.UpdatedAt = source.ManagedModel.UpdatedAt

	validationErrors, err := ToAPIProductCatalogValidationErrors(source)
	if err != nil {
		return result, err
	}
	result.ValidationErrors = validationErrors

	result.Version = source.AddonMeta.Version

	rcs, err := ToAPIBillingRateCards(source.RateCards.AsProductCatalogRateCards())
	if err != nil {
		return result, err
	}
	result.RateCards = rcs

	return result, nil
}

// FromAPICreateAddonRequest converts a v3 CreateAddonRequest to a domain CreateAddonInput.
func FromAPICreateAddonRequest(namespace string, body apiv3.CreateAddonRequest) (addon.CreateAddonInput, error) {
	instanceType, err := FromAPIAddonInstanceType(body.InstanceType)
	if err != nil {
		return addon.CreateAddonInput{}, err
	}

	metadata, err := labels.ToMetadata(body.Labels)
	if err != nil {
		return addon.CreateAddonInput{}, err
	}

	rcs, err := FromAPIBillingRateCards(body.RateCards)
	if err != nil {
		return addon.CreateAddonInput{}, err
	}

	return addon.CreateAddonInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Key:          body.Key,
				Name:         body.Name,
				Description:  body.Description,
				Currency:     currency.Code(body.Currency),
				InstanceType: instanceType,
				Metadata:     metadata,
			},
			RateCards: rcs,
		},
	}, nil
}

// FromAPIUpsertAddonRequest converts a v3 UpsertAddonRequest to a domain UpdateAddonInput.
func FromAPIUpsertAddonRequest(namespace string, addonID string, body apiv3.UpsertAddonRequest) (addon.UpdateAddonInput, error) {
	instanceType, err := FromAPIAddonInstanceType(body.InstanceType)
	if err != nil {
		return addon.UpdateAddonInput{}, err
	}

	metadata, err := labels.ToMetadata(body.Labels)
	if err != nil {
		return addon.UpdateAddonInput{}, err
	}

	rcs, err := FromAPIBillingRateCards(body.RateCards)
	if err != nil {
		return addon.UpdateAddonInput{}, err
	}

	return addon.UpdateAddonInput{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        addonID,
		},
		Name:         lo.ToPtr(body.Name),
		Description:  body.Description,
		Metadata:     &metadata,
		InstanceType: &instanceType,
		RateCards:    &rcs,
	}, nil
}

// ToAPIBillingRateCards converts domain RateCards to v3 BillingRateCard slice.
func ToAPIBillingRateCards(rcs productcatalog.RateCards) ([]apiv3.BillingRateCard, error) {
	if len(rcs) == 0 {
		return nil, nil
	}

	result := make([]apiv3.BillingRateCard, 0, len(rcs))
	for _, rc := range rcs {
		apiRC, err := ToAPIBillingRateCard(rc)
		if err != nil {
			return nil, err
		}
		result = append(result, apiRC)
	}
	return result, nil
}

func ToAPIBillingRateCard(rc productcatalog.RateCard) (apiv3.BillingRateCard, error) {
	meta := rc.AsMeta()

	result := apiv3.BillingRateCard{
		Key:         meta.Key,
		Name:        meta.Name,
		Description: meta.Description,
		Labels:      labels.FromMetadata(meta.Metadata),
	}

	// Feature
	if meta.FeatureID != nil {
		result.Feature = &apiv3.FeatureReferenceItem{Id: *meta.FeatureID}
	}

	// TaxConfig
	if meta.TaxConfig != nil {
		result.TaxConfig = ToAPIBillingRateCardTaxConfig(meta.TaxConfig)
	}

	// Discounts
	if !meta.Discounts.IsEmpty() {
		result.Discounts = ToAPIBillingRateCardDiscounts(meta.Discounts)
	}

	switch rc.Type() {
	case productcatalog.FlatFeeRateCardType:
		flatRC := rc.(*productcatalog.FlatFeeRateCard)
		// billing_cadence stays nil for flat fee

		if meta.Price == nil {
			// free price
			var price apiv3.BillingPrice
			if err := price.FromBillingPriceFree(apiv3.BillingPriceFree{
				Type: apiv3.BillingPriceFreeTypeFree,
			}); err != nil {
				return result, fmt.Errorf("failed to encode free price: %w", err)
			}
			result.Price = price
		} else {
			flatPrice, err := meta.Price.AsFlat()
			if err != nil {
				return result, fmt.Errorf("failed to cast FlatPrice: %w", err)
			}

			pt, err := ToAPIBillingPricePaymentTerm(flatPrice.PaymentTerm)
			if err != nil {
				return result, err
			}
			result.PaymentTerm = pt

			var price apiv3.BillingPrice
			if err := price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
				Type:   apiv3.BillingPriceFlatTypeFlat,
				Amount: flatPrice.Amount.String(),
			}); err != nil {
				return result, fmt.Errorf("failed to encode flat price: %w", err)
			}
			result.Price = price
		}

		_ = flatRC // billing_cadence is nil, nothing else needed

	case productcatalog.UsageBasedRateCardType:
		usageRC := rc.(*productcatalog.UsageBasedRateCard)
		bc := usageRC.BillingCadence
		isoStr := bc.ISOString().String()
		result.BillingCadence = &isoStr

		if meta.Price == nil {
			// usage-based with no price: encode as free
			var price apiv3.BillingPrice
			if err := price.FromBillingPriceFree(apiv3.BillingPriceFree{
				Type: apiv3.BillingPriceFreeTypeFree,
			}); err != nil {
				return result, fmt.Errorf("failed to encode free price: %w", err)
			}
			result.Price = price
		} else {
			price, commitments, paymentTerm, err := ToAPIBillingPrice(*meta.Price)
			if err != nil {
				return result, err
			}
			result.Price = price
			result.Commitments = commitments
			result.PaymentTerm = paymentTerm
		}

	default:
		return result, fmt.Errorf("unsupported rate card type: %s", rc.Type())
	}

	return result, nil
}

func ToAPIBillingPrice(price productcatalog.Price) (apiv3.BillingPrice, *apiv3.BillingSpendCommitments, *apiv3.BillingPricePaymentTerm, error) {
	var apiPrice apiv3.BillingPrice
	var commitments *apiv3.BillingSpendCommitments
	var paymentTerm *apiv3.BillingPricePaymentTerm

	switch price.Type() {
	case productcatalog.FlatPriceType:
		flatPrice, err := price.AsFlat()
		if err != nil {
			return apiPrice, nil, nil, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		paymentTerm, err = ToAPIBillingPricePaymentTerm(flatPrice.PaymentTerm)
		if err != nil {
			return apiPrice, nil, nil, err
		}

		if err := apiPrice.FromBillingPriceFlat(apiv3.BillingPriceFlat{
			Type:   apiv3.BillingPriceFlatTypeFlat,
			Amount: flatPrice.Amount.String(),
		}); err != nil {
			return apiPrice, nil, nil, fmt.Errorf("failed to encode flat price: %w", err)
		}

	case productcatalog.UnitPriceType:
		unitPrice, err := price.AsUnit()
		if err != nil {
			return apiPrice, nil, nil, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}

		commitments = ToAPIBillingSpendCommitments(unitPrice.MinimumAmount, unitPrice.MaximumAmount)

		if err := apiPrice.FromBillingPriceUnit(apiv3.BillingPriceUnit{
			Type:   apiv3.BillingPriceUnitTypeUnit,
			Amount: unitPrice.Amount.String(),
		}); err != nil {
			return apiPrice, nil, nil, fmt.Errorf("failed to encode unit price: %w", err)
		}

	case productcatalog.TieredPriceType:
		tieredPrice, err := price.AsTiered()
		if err != nil {
			return apiPrice, nil, nil, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}

		commitments = ToAPIBillingSpendCommitments(tieredPrice.MinimumAmount, tieredPrice.MaximumAmount)
		apiTiers := ToAPIBillingPriceTiers(tieredPrice.Tiers)

		switch tieredPrice.Mode {
		case productcatalog.GraduatedTieredPrice:
			if err := apiPrice.FromBillingPriceGraduated(apiv3.BillingPriceGraduated{
				Type:  apiv3.BillingPriceGraduatedTypeGraduated,
				Tiers: apiTiers,
			}); err != nil {
				return apiPrice, nil, nil, fmt.Errorf("failed to encode graduated price: %w", err)
			}
		case productcatalog.VolumeTieredPrice:
			if err := apiPrice.FromBillingPriceVolume(apiv3.BillingPriceVolume{
				Type:  apiv3.BillingPriceVolumeTypeVolume,
				Tiers: apiTiers,
			}); err != nil {
				return apiPrice, nil, nil, fmt.Errorf("failed to encode volume price: %w", err)
			}
		default:
			return apiPrice, nil, nil, fmt.Errorf("unsupported tiered price mode: %s", tieredPrice.Mode)
		}

	default:
		return apiPrice, nil, nil, fmt.Errorf("unsupported price type for v3: %s", price.Type())
	}

	return apiPrice, commitments, paymentTerm, nil
}

func ToAPIBillingPriceTiers(tiers []productcatalog.PriceTier) []apiv3.BillingPriceTier {
	result := make([]apiv3.BillingPriceTier, 0, len(tiers))
	for _, t := range tiers {
		tier := apiv3.BillingPriceTier{}

		if t.UpToAmount != nil {
			s := t.UpToAmount.String()
			tier.UpToAmount = &s
		}

		if t.UnitPrice != nil {
			tier.UnitPrice = &apiv3.BillingPriceUnit{
				Type:   apiv3.BillingPriceUnitTypeUnit,
				Amount: t.UnitPrice.Amount.String(),
			}
		}

		if t.FlatPrice != nil {
			tier.FlatPrice = &apiv3.BillingPriceFlat{
				Type:   apiv3.BillingPriceFlatTypeFlat,
				Amount: t.FlatPrice.Amount.String(),
			}
		}

		result = append(result, tier)
	}
	return result
}

func ToAPIBillingSpendCommitments(minAmount, maxAmount *decimal.Decimal) *apiv3.BillingSpendCommitments {
	if minAmount == nil && maxAmount == nil {
		return nil
	}

	c := &apiv3.BillingSpendCommitments{}
	if minAmount != nil {
		s := minAmount.String()
		c.MinimumAmount = &s
	}
	if maxAmount != nil {
		s := maxAmount.String()
		c.MaximumAmount = &s
	}
	return c
}

func ToAPIBillingPricePaymentTerm(t productcatalog.PaymentTermType) (*apiv3.BillingPricePaymentTerm, error) {
	switch t {
	case productcatalog.InArrearsPaymentTerm:
		return lo.ToPtr(apiv3.BillingPricePaymentTermInArrears), nil
	case productcatalog.InAdvancePaymentTerm:
		return lo.ToPtr(apiv3.BillingPricePaymentTermInAdvance), nil
	default:
		return nil, fmt.Errorf("unknown payment term: %v", t)
	}
}

func ToAPIBillingRateCardTaxConfig(tc *productcatalog.TaxConfig) *apiv3.BillingRateCardTaxConfig {
	if tc == nil {
		return nil
	}

	result := &apiv3.BillingRateCardTaxConfig{}

	if tc.Behavior != nil {
		result.Behavior = (*apiv3.BillingTaxBehavior)(tc.Behavior)
	}

	if tc.TaxCodeID != nil {
		result.Code = apiv3.TaxCodeReferenceItem{Id: *tc.TaxCodeID}
	}

	return result
}

func ToAPIBillingRateCardDiscounts(d productcatalog.Discounts) *apiv3.BillingRateCardDiscounts {
	if d.IsEmpty() {
		return nil
	}

	result := &apiv3.BillingRateCardDiscounts{}

	if d.Percentage != nil {
		pct := float32(d.Percentage.Percentage.InexactFloat64())
		result.Percentage = &pct
	}

	if d.Usage != nil {
		s := d.Usage.Quantity.String()
		result.Usage = &s
	}

	return result
}

// FromAPIBillingRateCards converts v3 BillingRateCard slice to domain RateCards.
func FromAPIBillingRateCards(rcs []apiv3.BillingRateCard) (productcatalog.RateCards, error) {
	if len(rcs) == 0 {
		return nil, nil
	}

	result := make(productcatalog.RateCards, 0, len(rcs))
	for _, rc := range rcs {
		domainRC, err := FromAPIBillingRateCard(rc)
		if err != nil {
			return nil, err
		}
		result = append(result, domainRC)
	}
	return result, nil
}

func FromAPIBillingRateCard(rc apiv3.BillingRateCard) (productcatalog.RateCard, error) {
	meta := productcatalog.RateCardMeta{
		Key:         rc.Key,
		Name:        rc.Name,
		Description: rc.Description,
	}

	if rc.Labels != nil {
		md, err := labels.ToMetadata(rc.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to convert labels: %w", err)
		}
		meta.Metadata = md
	}

	if rc.Feature != nil {
		meta.FeatureID = &rc.Feature.Id
	}

	if rc.TaxConfig != nil {
		meta.TaxConfig = FromAPIBillingRateCardTaxConfig(rc.TaxConfig)
	}

	if rc.Discounts != nil {
		discounts, err := FromAPIBillingRateCardDiscounts(rc.Discounts)
		if err != nil {
			return nil, err
		}
		meta.Discounts = discounts
	}

	priceDiscriminator, err := rc.Price.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to determine price type: %w", err)
	}

	if rc.BillingCadence == nil {
		// FlatFeeRateCard
		flatRC := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta,
		}

		switch priceDiscriminator {
		case string(apiv3.BillingPriceFreeTypeFree):
			// nil price
		case string(apiv3.BillingPriceFlatTypeFlat):
			flatAPI, err := rc.Price.AsBillingPriceFlat()
			if err != nil {
				return nil, fmt.Errorf("failed to decode flat price: %w", err)
			}
			flatPrice, paymentTerm, err := FromAPIBillingPriceFlat(flatAPI, rc.PaymentTerm)
			if err != nil {
				return nil, err
			}
			flatPrice.PaymentTerm = paymentTerm
			flatRC.Price = productcatalog.NewPriceFrom(flatPrice)
		default:
			return nil, fmt.Errorf("unsupported price type %q for flat fee rate card (billing_cadence must be set for non-flat prices)", priceDiscriminator)
		}

		return flatRC, nil
	}

	// UsageBasedRateCard
	isoStr := datetime.ISODurationString(*rc.BillingCadence)
	billingCadence, err := isoStr.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse billing_cadence: %w", err)
	}

	usageRC := &productcatalog.UsageBasedRateCard{
		RateCardMeta:   meta,
		BillingCadence: billingCadence,
	}

	switch priceDiscriminator {
	case string(apiv3.BillingPriceFreeTypeFree):
		// nil price

	case string(apiv3.BillingPriceFlatTypeFlat):
		flatAPI, err := rc.Price.AsBillingPriceFlat()
		if err != nil {
			return nil, fmt.Errorf("failed to decode flat price: %w", err)
		}
		flatPrice, paymentTerm, err := FromAPIBillingPriceFlat(flatAPI, rc.PaymentTerm)
		if err != nil {
			return nil, err
		}
		flatPrice.PaymentTerm = paymentTerm
		usageRC.Price = productcatalog.NewPriceFrom(flatPrice)

	case string(apiv3.BillingPriceUnitTypeUnit):
		unitAPI, err := rc.Price.AsBillingPriceUnit()
		if err != nil {
			return nil, fmt.Errorf("failed to decode unit price: %w", err)
		}
		unitPrice, err := FromAPIBillingPriceUnit(unitAPI, rc.Commitments)
		if err != nil {
			return nil, err
		}
		usageRC.Price = productcatalog.NewPriceFrom(unitPrice)

	case string(apiv3.BillingPriceGraduatedTypeGraduated):
		graduatedAPI, err := rc.Price.AsBillingPriceGraduated()
		if err != nil {
			return nil, fmt.Errorf("failed to decode graduated price: %w", err)
		}
		tieredPrice, err := FromAPIBillingPriceGraduated(graduatedAPI, rc.Commitments)
		if err != nil {
			return nil, err
		}
		usageRC.Price = productcatalog.NewPriceFrom(tieredPrice)

	case string(apiv3.BillingPriceVolumeTypeVolume):
		volumeAPI, err := rc.Price.AsBillingPriceVolume()
		if err != nil {
			return nil, fmt.Errorf("failed to decode volume price: %w", err)
		}
		tieredPrice, err := FromAPIBillingPriceVolume(volumeAPI, rc.Commitments)
		if err != nil {
			return nil, err
		}
		usageRC.Price = productcatalog.NewPriceFrom(tieredPrice)

	default:
		return nil, fmt.Errorf("unsupported price type for v3: %s", priceDiscriminator)
	}

	return usageRC, nil
}

func FromAPIBillingPriceFlat(f apiv3.BillingPriceFlat, paymentTermPtr *apiv3.BillingPricePaymentTerm) (productcatalog.FlatPrice, productcatalog.PaymentTermType, error) {
	amount, err := decimal.NewFromString(f.Amount)
	if err != nil {
		return productcatalog.FlatPrice{}, "", fmt.Errorf("failed to parse flat price amount: %w", err)
	}

	paymentTerm := productcatalog.DefaultPaymentTerm
	if paymentTermPtr != nil {
		switch *paymentTermPtr {
		case apiv3.BillingPricePaymentTermInArrears:
			paymentTerm = productcatalog.InArrearsPaymentTerm
		case apiv3.BillingPricePaymentTermInAdvance:
			paymentTerm = productcatalog.InAdvancePaymentTerm
		default:
			return productcatalog.FlatPrice{}, "", fmt.Errorf("unknown payment term: %v", *paymentTermPtr)
		}
	}

	return productcatalog.FlatPrice{Amount: amount}, paymentTerm, nil
}

func FromAPIBillingPriceUnit(u apiv3.BillingPriceUnit, commitments *apiv3.BillingSpendCommitments) (productcatalog.UnitPrice, error) {
	amount, err := decimal.NewFromString(u.Amount)
	if err != nil {
		return productcatalog.UnitPrice{}, fmt.Errorf("failed to parse unit price amount: %w", err)
	}

	up := productcatalog.UnitPrice{Amount: amount}

	if commitments != nil {
		c, err := FromAPIBillingSpendCommitments(commitments)
		if err != nil {
			return up, err
		}
		up.Commitments = c
	}

	return up, nil
}

func FromAPIBillingPriceGraduated(g apiv3.BillingPriceGraduated, commitments *apiv3.BillingSpendCommitments) (productcatalog.TieredPrice, error) {
	tiers, err := FromAPIBillingPriceTiers(g.Tiers)
	if err != nil {
		return productcatalog.TieredPrice{}, err
	}

	tp := productcatalog.TieredPrice{
		Mode:  productcatalog.GraduatedTieredPrice,
		Tiers: tiers,
	}

	if commitments != nil {
		c, err := FromAPIBillingSpendCommitments(commitments)
		if err != nil {
			return tp, err
		}
		tp.Commitments = c
	}

	return tp, nil
}

func FromAPIBillingPriceVolume(v apiv3.BillingPriceVolume, commitments *apiv3.BillingSpendCommitments) (productcatalog.TieredPrice, error) {
	tiers, err := FromAPIBillingPriceTiers(v.Tiers)
	if err != nil {
		return productcatalog.TieredPrice{}, err
	}

	tp := productcatalog.TieredPrice{
		Mode:  productcatalog.VolumeTieredPrice,
		Tiers: tiers,
	}

	if commitments != nil {
		c, err := FromAPIBillingSpendCommitments(commitments)
		if err != nil {
			return tp, err
		}
		tp.Commitments = c
	}

	return tp, nil
}

func FromAPIBillingPriceTiers(tiers []apiv3.BillingPriceTier) ([]productcatalog.PriceTier, error) {
	result := make([]productcatalog.PriceTier, 0, len(tiers))
	for _, t := range tiers {
		tier := productcatalog.PriceTier{}

		if t.UpToAmount != nil {
			d, err := decimal.NewFromString(*t.UpToAmount)
			if err != nil {
				return nil, fmt.Errorf("failed to parse tier up_to_amount: %w", err)
			}
			tier.UpToAmount = &d
		}

		if t.UnitPrice != nil {
			d, err := decimal.NewFromString(t.UnitPrice.Amount)
			if err != nil {
				return nil, fmt.Errorf("failed to parse tier unit price amount: %w", err)
			}
			tier.UnitPrice = &productcatalog.PriceTierUnitPrice{Amount: d}
		}

		if t.FlatPrice != nil {
			d, err := decimal.NewFromString(t.FlatPrice.Amount)
			if err != nil {
				return nil, fmt.Errorf("failed to parse tier flat price amount: %w", err)
			}
			tier.FlatPrice = &productcatalog.PriceTierFlatPrice{Amount: d}
		}

		result = append(result, tier)
	}
	return result, nil
}

func FromAPIBillingSpendCommitments(c *apiv3.BillingSpendCommitments) (productcatalog.Commitments, error) {
	var result productcatalog.Commitments

	if c == nil {
		return result, nil
	}

	if c.MinimumAmount != nil {
		d, err := decimal.NewFromString(*c.MinimumAmount)
		if err != nil {
			return result, fmt.Errorf("failed to parse minimum_amount: %w", err)
		}
		result.MinimumAmount = &d
	}

	if c.MaximumAmount != nil {
		d, err := decimal.NewFromString(*c.MaximumAmount)
		if err != nil {
			return result, fmt.Errorf("failed to parse maximum_amount: %w", err)
		}
		result.MaximumAmount = &d
	}

	return result, nil
}

func FromAPIBillingRateCardTaxConfig(tc *apiv3.BillingRateCardTaxConfig) *productcatalog.TaxConfig {
	if tc == nil {
		return nil
	}

	result := &productcatalog.TaxConfig{}

	if tc.Behavior != nil {
		result.Behavior = (*productcatalog.TaxBehavior)(tc.Behavior)
	}

	if tc.Code.Id != "" {
		result.TaxCodeID = &tc.Code.Id
	}

	return result
}

func FromAPIBillingRateCardDiscounts(d *apiv3.BillingRateCardDiscounts) (productcatalog.Discounts, error) {
	var result productcatalog.Discounts

	if d == nil {
		return result, nil
	}

	if d.Percentage != nil {
		result.Percentage = &productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(float64(*d.Percentage)),
		}
	}

	if d.Usage != nil {
		qty, err := decimal.NewFromString(*d.Usage)
		if err != nil {
			return result, fmt.Errorf("failed to parse usage discount quantity: %w", err)
		}
		result.Usage = &productcatalog.UsageDiscount{Quantity: qty}
	}

	return result, nil
}
