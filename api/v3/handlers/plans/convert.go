package plans

import (
	"context"
	"encoding/json"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromAPIPlanSortField(ctx context.Context, field string) (plan.OrderBy, error) {
	switch field {
	case "id":
		return plan.OrderByID, nil
	case "key":
		return plan.OrderByKey, nil
	case "version":
		return plan.OrderByVersion, nil
	case "created_at":
		return plan.OrderByCreatedAt, nil
	case "updated_at":
		return plan.OrderByUpdatedAt, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(
			ctx, field, "id", "key", "version", "created_at", "updated_at",
		)
	}
}

func ToAPIBillingPlan(p plan.Plan) (api.BillingPlan, error) {
	validationIssues, _ := p.AsProductCatalogPlan().ValidationErrors()

	resp := api.BillingPlan{
		BillingCadence:   p.BillingCadence.String(),
		CreatedAt:        p.CreatedAt,
		Currency:         api.BillingCurrencyCode(p.Currency.GetCode().String()),
		DeletedAt:        p.DeletedAt,
		Description:      p.Description,
		EffectiveFrom:    p.EffectiveFrom,
		EffectiveTo:      p.EffectiveTo,
		Id:               p.ID,
		Key:              p.Key,
		Name:             p.Name,
		UpdatedAt:        p.UpdatedAt,
		Version:          p.Version,
		ProRatingEnabled: lo.ToPtr(p.ProRatingConfig.Enabled),
		SettlementMode:   lo.ToPtr(api.BillingSettlementMode(p.SettlementMode)),
		ValidationErrors: ToAPIProductCatalogValidationErrors(validationIssues),
	}

	var status api.BillingPlanStatus
	switch p.Status() {
	case productcatalog.PlanStatusDraft:
		status = api.BillingPlanStatusDraft
	case productcatalog.PlanStatusActive:
		status = api.BillingPlanStatusActive
	case productcatalog.PlanStatusArchived:
		status = api.BillingPlanStatusArchived
	case productcatalog.PlanStatusScheduled:
		status = api.BillingPlanStatusScheduled
	default:
		return resp, fmt.Errorf("invalid PlanStatus: %s", p.Status())
	}

	resp.Status = status

	resp.Phases = make([]api.BillingPlanPhase, 0, len(p.Phases))
	for _, phase := range p.Phases {
		billingPhase, err := ToAPIBillingPlanPhase(phase)
		if err != nil {
			return resp, fmt.Errorf("failed to convert plan phase: %w", err)
		}

		resp.Phases = append(resp.Phases, billingPhase)
	}

	return resp, nil
}

func ToAPIBillingPlanPhase(p plan.Phase) (api.BillingPlanPhase, error) {
	phase := api.BillingPlanPhase{
		Description: p.Description,
		Duration:    (*api.ISO8601Duration)(p.Duration.ISOStringPtrOrNil()),
		Key:         p.Key,
		Name:        p.Name,
		RateCards:   make([]api.BillingRateCard, 0, len(p.RateCards)),
	}

	for _, rc := range p.RateCards {
		billingRC, err := ToAPIBillingRateCard(rc)
		if err != nil {
			return phase, fmt.Errorf("failed to convert rate card %q: %w", rc.Key(), err)
		}

		phase.RateCards = append(phase.RateCards, billingRC)
	}

	return phase, nil
}

func ToAPIBillingRateCard(rc productcatalog.RateCard) (api.BillingRateCard, error) {
	meta := rc.AsMeta()

	result := api.BillingRateCard{
		Key:         meta.Key,
		Name:        meta.Name,
		Description: meta.Description,
		Discounts:   ToAPIBillingRateCardDiscount(meta.Discounts),
		TaxConfig:   ToAPIBillingRateCardTaxConfig(meta.TaxConfig, meta.TaxCode),
	}

	if meta.Currency != nil {
		result.Currency = lo.ToPtr(api.BillingCurrencyCode(meta.Currency.GetCode().String()))
	}

	if meta.FeatureID != nil {
		result.Feature = &api.FeatureReference{
			Id: *meta.FeatureID,
		}
	}

	switch rc.Type() {
	case productcatalog.FlatFeeRateCardType:
		if bc := rc.GetBillingCadence(); bc != nil {
			result.BillingCadence = lo.ToPtr(bc.ISOString().String())
		}

		if meta.Price != nil {
			flatPrice, err := meta.Price.AsFlat()
			if err != nil {
				return result, fmt.Errorf("failed to read flat price: %w", err)
			}

			result.PaymentTerm = lo.ToPtr(api.BillingPricePaymentTerm(flatPrice.PaymentTerm))
		}

	case productcatalog.UsageBasedRateCardType:
		bc := rc.GetBillingCadence()
		if bc == nil {
			return result, fmt.Errorf("usage-based rate card %q missing billing cadence", meta.Key)
		}

		result.BillingCadence = lo.ToPtr(bc.ISOString().String())

		if meta.Price != nil {
			result.Commitments = ToAPIBillingSpendCommitments(meta.Price.GetCommitments())
		}

	default:
		return result, fmt.Errorf("unknown rate card type: %s", rc.Type())
	}

	price, err := ToAPIBillingPrice(meta.Price)
	if err != nil {
		return result, fmt.Errorf("failed to convert price: %w", err)
	}

	result.Price = price

	if meta.EntitlementTemplate != nil {
		ent, err := ToAPIBillingRateCardEntitlement(meta.EntitlementTemplate)
		if err != nil {
			return result, fmt.Errorf("failed to convert entitlement template: %w", err)
		}

		result.Entitlement = ent
	}

	// Prefer a stored unit config; fall back to synthesizing one from a v1
	// dynamic/package price when none is stored. These two sources never coexist on
	// a rate card reachable here: the v3 write path cannot author a package/dynamic
	// price (only free/flat/unit/graduated/volume), and the v1 API has no unit_config
	// field — so a stored unit_config and a package/dynamic price cannot be produced
	// through either API, and there is no double-conversion ambiguity to resolve.
	// (The RateCardMeta.Validate price-type rule is a publish-blocking warning, not a
	// hard write-time reject, so it is not what guarantees this.)
	if meta.UnitConfig != nil {
		result.UnitConfig = lo.ToPtr(ToAPIBillingUnitConfig(*meta.UnitConfig))
	} else {
		unitConfig, err := ToAPIBillingRateCardUnitConfig(meta.Price)
		if err != nil {
			return result, fmt.Errorf("failed to convert unit config: %w", err)
		}

		result.UnitConfig = unitConfig
	}

	return result, nil
}

func ToAPIBillingRateCardEntitlement(t *productcatalog.EntitlementTemplate) (*api.BillingRateCardEntitlement, error) {
	out := &api.BillingRateCardEntitlement{}

	switch t.Type() {
	case entitlement.EntitlementTypeMetered:
		metered, err := t.AsMetered()
		if err != nil {
			return nil, fmt.Errorf("failed to read metered entitlement template: %w", err)
		}

		apiMetered := api.BillingRateCardMeteredEntitlement{
			Type:        "metered",
			IsSoftLimit: lo.ToPtr(metered.IsSoftLimit),
			Limit:       metered.IssueAfterReset,
			UsagePeriod: lo.ToPtr(metered.UsagePeriod.ISOString().String()),
		}

		if err := out.FromBillingRateCardMeteredEntitlement(apiMetered); err != nil {
			return nil, fmt.Errorf("failed to set metered entitlement template: %w", err)
		}

	case entitlement.EntitlementTypeStatic:
		static, err := t.AsStatic()
		if err != nil {
			return nil, fmt.Errorf("failed to read static entitlement template: %w", err)
		}

		// The domain stores the config as a JSON string token wrapping the JSON
		// text (v1 convention, relied on by subscription materialization). Unwrap
		// it so the v3 API returns the raw JSON value. Legacy values that are not
		// string-wrapped are returned as stored so reads never hard-fail.
		var config json.RawMessage
		if len(static.Config) > 0 {
			var text string
			if err := json.Unmarshal(static.Config, &text); err == nil && json.Valid([]byte(text)) {
				config = json.RawMessage(text)
			} else {
				config = static.Config
			}
		}

		if err := out.FromBillingRateCardStaticEntitlement(api.BillingRateCardStaticEntitlement{
			Type:   "static",
			Config: config,
		}); err != nil {
			return nil, fmt.Errorf("failed to set static entitlement template: %w", err)
		}

	case entitlement.EntitlementTypeBoolean:
		if err := out.FromBillingRateCardBooleanEntitlement(api.BillingRateCardBooleanEntitlement{
			Type: "boolean",
		}); err != nil {
			return nil, fmt.Errorf("failed to set boolean entitlement template: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown entitlement template type: %s", t.Type())
	}

	return out, nil
}

// ToAPIBillingRateCardUnitConfig synthesizes a v3 unit config from a v1 dynamic
// or package price. v3 does not surface dynamic or package prices directly;
// instead they are rendered as a unit price paired with a unit config that
// describes the conversion that v1 applied implicitly.
func ToAPIBillingRateCardUnitConfig(p *productcatalog.Price) (*api.BillingUnitConfig, error) {
	if p == nil {
		return nil, nil
	}

	switch p.Type() {
	case productcatalog.DynamicPriceType:
		dynamic, err := p.AsDynamic()
		if err != nil {
			return nil, fmt.Errorf("failed to read dynamic price: %w", err)
		}

		return &api.BillingUnitConfig{
			Operation:        api.BillingUnitConfigOperationMultiply,
			ConversionFactor: dynamic.Multiplier.String(),
		}, nil

	case productcatalog.PackagePriceType:
		pkg, err := p.AsPackage()
		if err != nil {
			return nil, fmt.Errorf("failed to read package price: %w", err)
		}

		return &api.BillingUnitConfig{
			Operation:        api.BillingUnitConfigOperationDivide,
			ConversionFactor: pkg.QuantityPerPackage.String(),
			Rounding:         lo.ToPtr(api.BillingUnitConfigRoundingModeCeiling),
		}, nil

	default:
		return nil, nil
	}
}

// ToAPIBillingUnitConfig maps a stored domain unit config to its API
// representation. Rounding and Precision are omitted when no rounding is applied,
// mirroring the domain semantics where both are inert for the "none" mode.
func ToAPIBillingUnitConfig(uc productcatalog.UnitConfig) api.BillingUnitConfig {
	out := api.BillingUnitConfig{
		Operation:        api.BillingUnitConfigOperation(uc.Operation),
		ConversionFactor: uc.ConversionFactor.String(),
		DisplayUnit:      uc.DisplayUnit,
	}

	if !uc.Rounding.IsNone() {
		out.Rounding = lo.ToPtr(api.BillingUnitConfigRoundingMode(uc.Rounding))
		out.Precision = lo.ToPtr(uc.Precision)
	}

	return out
}

// FromAPIBillingUnitConfig maps the API unit config to the domain type. The enum
// values are identical across the two layers, so operation and rounding are direct
// casts; UnitConfig.Validate (run via RateCardMeta.Validate) rejects unknown enum
// values and a non-positive conversion factor.
func FromAPIBillingUnitConfig(uc api.BillingUnitConfig) (*productcatalog.UnitConfig, error) {
	conversionFactor, err := decimal.NewFromString(uc.ConversionFactor)
	if err != nil {
		return nil, fmt.Errorf("invalid conversion factor: %w", err)
	}

	out := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperation(uc.Operation),
		ConversionFactor: conversionFactor,
		DisplayUnit:      uc.DisplayUnit,
	}

	if uc.Rounding != nil {
		out.Rounding = productcatalog.UnitConfigRoundingMode(*uc.Rounding)
	}

	// Precision is inert without rounding; only carry it when rounding is active so
	// it round-trips consistently with ToAPIBillingUnitConfig, which omits Precision
	// in the "none" case.
	if !out.Rounding.IsNone() {
		out.Precision = lo.FromPtr(uc.Precision)
	}

	return out, nil
}

func ToAPIBillingPrice(p *productcatalog.Price) (api.BillingPrice, error) {
	var result api.BillingPrice

	if p == nil {
		if err := result.FromBillingPriceFree(api.BillingPriceFree{
			Type: api.BillingPriceFreeType("free"),
		}); err != nil {
			return result, fmt.Errorf("failed to set free price: %w", err)
		}

		return result, nil
	}

	switch p.Type() {
	case productcatalog.FlatPriceType:
		flat, err := p.AsFlat()
		if err != nil {
			return result, fmt.Errorf("failed to read flat price: %w", err)
		}

		if err = result.FromBillingPriceFlat(api.BillingPriceFlat{
			Amount: flat.Amount.String(),
			Type:   api.BillingPriceFlatType("flat"),
		}); err != nil {
			return result, fmt.Errorf("failed to set flat price: %w", err)
		}

	case productcatalog.UnitPriceType:
		unit, err := p.AsUnit()
		if err != nil {
			return result, fmt.Errorf("failed to read unit price: %w", err)
		}

		if err = result.FromBillingPriceUnit(api.BillingPriceUnit{
			Amount: unit.Amount.String(),
			Type:   api.BillingPriceUnitType("unit"),
		}); err != nil {
			return result, fmt.Errorf("failed to set unit price: %w", err)
		}

	case productcatalog.TieredPriceType:
		tiered, err := p.AsTiered()
		if err != nil {
			return result, fmt.Errorf("failed to read tiered price: %w", err)
		}

		tiers := ToAPIBillingPriceTiers(tiered.Tiers)

		switch tiered.Mode {
		case productcatalog.GraduatedTieredPrice:
			if err = result.FromBillingPriceGraduated(api.BillingPriceGraduated{
				Tiers: tiers,
				Type:  api.BillingPriceGraduatedType("graduated"),
			}); err != nil {
				return result, fmt.Errorf("failed to set graduated price: %w", err)
			}

		case productcatalog.VolumeTieredPrice:
			if err = result.FromBillingPriceVolume(api.BillingPriceVolume{
				Tiers: tiers,
				Type:  api.BillingPriceVolumeType("volume"),
			}); err != nil {
				return result, fmt.Errorf("failed to set volume price: %w", err)
			}

		default:
			return result, fmt.Errorf("unknown tiered price mode: %s", tiered.Mode)
		}

	case productcatalog.DynamicPriceType:
		// Dynamic prices are surfaced in v3 as a unit price of amount 1; the
		// multiplier is carried separately on the rate card's unit config.
		if err := result.FromBillingPriceUnit(api.BillingPriceUnit{
			Amount: "1",
			Type:   api.BillingPriceUnitType("unit"),
		}); err != nil {
			return result, fmt.Errorf("failed to set unit price for dynamic price: %w", err)
		}

	case productcatalog.PackagePriceType:
		// Package prices are surfaced in v3 as a unit price; the package size
		// is carried separately on the rate card's unit config.
		pkg, err := p.AsPackage()
		if err != nil {
			return result, fmt.Errorf("failed to read package price: %w", err)
		}

		if err = result.FromBillingPriceUnit(api.BillingPriceUnit{
			Amount: pkg.Amount.String(),
			Type:   api.BillingPriceUnitType("unit"),
		}); err != nil {
			return result, fmt.Errorf("failed to set unit price for package price: %w", err)
		}

	default:
		return result, fmt.Errorf("unknown price type: %s", p.Type())
	}

	return result, nil
}

func ToAPIBillingPriceTiers(tiers []productcatalog.PriceTier) []api.BillingPriceTier {
	result := make([]api.BillingPriceTier, 0, len(tiers))

	for _, t := range tiers {
		tier := api.BillingPriceTier{}

		if t.UpToAmount != nil {
			tier.UpToAmount = lo.ToPtr(t.UpToAmount.String())
		}

		if t.FlatPrice != nil {
			tier.FlatPrice = &api.BillingPriceFlat{
				Amount: t.FlatPrice.Amount.String(),
				Type:   api.BillingPriceFlatType("flat"),
			}
		}

		if t.UnitPrice != nil {
			tier.UnitPrice = &api.BillingPriceUnit{
				Amount: t.UnitPrice.Amount.String(),
				Type:   api.BillingPriceUnitType("unit"),
			}
		}

		result = append(result, tier)
	}

	return result
}

func ToAPIBillingRateCardTaxConfig(c *productcatalog.TaxConfig, tc *taxcode.TaxCode) *api.BillingRateCardTaxConfig {
	if c == nil || tc == nil {
		return nil
	}

	result := &api.BillingRateCardTaxConfig{
		Code: api.TaxCodeReference{
			Id: tc.ID,
		},
	}

	if c.Behavior != nil {
		result.Behavior = lo.ToPtr(api.BillingTaxBehavior(*c.Behavior))
	}

	return result
}

func ToAPIBillingRateCardDiscount(d productcatalog.Discounts) *api.BillingRateCardDiscounts {
	if d.Percentage == nil && d.Usage == nil {
		return nil
	}

	result := &api.BillingRateCardDiscounts{}

	if d.Percentage != nil {
		pct := float32(d.Percentage.Percentage.InexactFloat64())
		result.Percentage = &pct
	}

	if d.Usage != nil {
		result.Usage = lo.ToPtr(d.Usage.Quantity.String())
	}

	return result
}

func ToAPIBillingSpendCommitments(c productcatalog.Commitments) *api.BillingSpendCommitments {
	if c.MinimumAmount == nil && c.MaximumAmount == nil {
		return nil
	}

	result := &api.BillingSpendCommitments{}

	if c.MinimumAmount != nil {
		result.MinimumAmount = lo.ToPtr(c.MinimumAmount.String())
	}

	if c.MaximumAmount != nil {
		result.MaximumAmount = lo.ToPtr(c.MaximumAmount.String())
	}

	return result
}

func ToAPIProductCatalogValidationErrors(issues models.ValidationIssues) *[]api.ProductCatalogValidationError {
	if len(issues) == 0 {
		return nil
	}

	result := make([]api.ProductCatalogValidationError, 0, len(issues))
	for _, issue := range issues {
		result = append(result, api.ProductCatalogValidationError{
			Code:    string(issue.Code()),
			Field:   issue.Field().JSONPath(),
			Message: issue.Message(),
		})
	}

	return &result
}

func FromAPIUpsertPlanRequest(ns string, planID string, body api.UpsertPlanRequest) (plan.UpdatePlanInput, error) {
	req := plan.UpdatePlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        planID,
		},
		Name:            &body.Name,
		Description:     body.Description,
		ProRatingConfig: lo.ToPtr(ToProRatingConfig(body.ProRatingEnabled)),
	}

	meta, err := labels.ToMetadata(body.Labels)
	if err != nil {
		return req, fmt.Errorf("failed to convert label metadata: %w", err)
	}

	if body.Labels != nil {
		m := meta
		req.Metadata = &m
	}

	phases := make([]productcatalog.Phase, 0, len(body.Phases))
	for _, phase := range body.Phases {
		p, err := FromAPIBillingPlanPhase(phase)
		if err != nil {
			return req, fmt.Errorf("failed to convert phase: %w", err)
		}

		phases = append(phases, p)
	}

	req.Phases = &phases

	return req, nil
}

func FromAPICreatePlanRequest(ns string, body api.CreatePlanRequest) (plan.CreatePlanInput, error) {
	meta, err := labels.ToMetadata(body.Labels)
	if err != nil {
		return plan.CreatePlanInput{}, fmt.Errorf("failed to convert label metadata: %w", err)
	}

	req := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ns,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:             body.Key,
				Name:            body.Name,
				Description:     body.Description,
				Metadata:        meta,
				ProRatingConfig: ToProRatingConfig(body.ProRatingEnabled),
			},
		},
	}

	req.Currency = currencyx.Code(body.Currency)

	billingCadence, err := datetime.ISODurationString(body.BillingCadence).Parse()
	if err != nil {
		return req, fmt.Errorf("invalid billing cadence: %w", err)
	}

	req.BillingCadence = billingCadence

	if len(body.Phases) > 0 {
		req.Phases = make([]productcatalog.Phase, 0, len(body.Phases))

		for _, phase := range body.Phases {
			p, err := FromAPIBillingPlanPhase(phase)
			if err != nil {
				return req, fmt.Errorf("failed to convert phase: %w", err)
			}

			req.Phases = append(req.Phases, p)
		}
	}

	return req, nil
}

func ToProRatingConfig(enabled *bool) productcatalog.ProRatingConfig {
	if enabled == nil || *enabled {
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}

	return productcatalog.ProRatingConfig{
		Enabled: false,
	}
}

func FromAPIBillingPlanPhase(p api.BillingPlanPhase) (productcatalog.Phase, error) {
	meta, labelErr := labels.ToMetadata(p.Labels)

	if labelErr != nil {
		return productcatalog.Phase{}, fmt.Errorf("failed to convert label metadata: %w", labelErr)
	}

	phase := productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
			Metadata:    meta,
		},
	}

	var err error

	phase.Duration, err = (*datetime.ISODurationString)(p.Duration).ParsePtrOrNil()
	if err != nil {
		return phase, fmt.Errorf("invalid duration: %w", err)
	}

	if len(p.RateCards) > 0 {
		phase.RateCards = make(productcatalog.RateCards, 0, len(p.RateCards))

		for _, rc := range p.RateCards {
			rateCard, err := FromAPIBillingRateCard(rc)
			if err != nil {
				return phase, fmt.Errorf("failed to convert rate card %q: %w", rc.Key, err)
			}

			phase.RateCards = append(phase.RateCards, rateCard)
		}
	}

	return phase, nil
}

func FromAPIBillingRateCard(rc api.BillingRateCard) (productcatalog.RateCard, error) {
	priceType, err := rc.Price.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read price type: %w", err)
	}

	labelMeta, err := labels.ToMetadata(rc.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to convert metadata: %w", err)
	}

	meta := productcatalog.RateCardMeta{
		Key:         rc.Key,
		Name:        rc.Name,
		Description: rc.Description,
		Metadata:    labelMeta,
	}

	if rc.Currency != nil {
		meta.Currency = currencyx.Code(*rc.Currency)
	}

	if rc.Feature != nil {
		meta.FeatureID = &rc.Feature.Id
	}

	if rc.TaxConfig != nil {
		meta.TaxConfig = FromAPIBillingRateCardTaxConfig(*rc.TaxConfig)
	}

	if rc.Discounts != nil {
		discounts, err := FromAPIBillingRateCardDiscounts(*rc.Discounts)
		if err != nil {
			return nil, fmt.Errorf("failed to convert discounts: %w", err)
		}

		meta.Discounts = discounts
	}

	// The billing cadence doubles as the default usage period for metered
	// entitlement templates, so parse it once up front.
	var billingCadence *datetime.ISODuration
	if rc.BillingCadence != nil {
		bc, err := datetime.ISODurationString(*rc.BillingCadence).Parse()
		if err != nil {
			return nil, fmt.Errorf("invalid billing cadence: %w", err)
		}

		billingCadence = &bc
	}

	if rc.Entitlement != nil {
		tmpl, err := FromAPIBillingRateCardEntitlement(*rc.Entitlement, billingCadence)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entitlement template: %w", err)
		}

		meta.EntitlementTemplate = tmpl
	}

	// Set the unit config up front: meta is copied by value into both the flat and
	// usage-based rate cards below, so it must be populated before the switch. We do
	// not branch on price type here; the price-type restriction lives in
	// RateCardMeta.Validate as a publish-blocking warning, so an invalid combination
	// (e.g. unit_config on a flat price) is mapped through and surfaces as a
	// validation issue on the draft rather than being rejected at create/update.
	if rc.UnitConfig != nil {
		unitConfig, err := FromAPIBillingUnitConfig(*rc.UnitConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unit config: %w", err)
		}

		meta.UnitConfig = unitConfig
	}

	switch priceType {
	case "free", "flat":
		price, err := FromAPIBillingPrice(rc.Price, rc.PaymentTerm)
		if err != nil {
			return nil, fmt.Errorf("failed to convert price: %w", err)
		}

		meta.Price = price

		flatRC := &productcatalog.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: billingCadence,
		}

		return flatRC, nil

	case "unit", "graduated", "volume":
		if billingCadence == nil {
			return nil, fmt.Errorf("billing cadence is required for usage-based rate card %q", rc.Key)
		}

		bc := *billingCadence

		price, err := FromAPIBillingPriceWithCommitments(rc.Price, rc.Commitments)
		if err != nil {
			return nil, fmt.Errorf("failed to convert price: %w", err)
		}

		meta.Price = price

		return &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta,
			BillingCadence: bc,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported price type: %s", priceType)
	}
}

func FromAPIBillingRateCardEntitlement(e api.BillingRateCardEntitlement, billingCadence *datetime.ISODuration) (*productcatalog.EntitlementTemplate, error) {
	disc, err := e.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read entitlement type: %w", err)
	}

	switch disc {
	case "metered":
		metered, err := e.AsBillingRateCardMeteredEntitlement()
		if err != nil {
			return nil, fmt.Errorf("failed to read metered entitlement template: %w", err)
		}

		// usage_period defaults to the rate card billing cadence when omitted.
		var usagePeriod datetime.ISODuration
		if metered.UsagePeriod != nil {
			usagePeriod, err = datetime.ISODurationString(*metered.UsagePeriod).Parse()
			if err != nil {
				return nil, models.NewGenericValidationError(fmt.Errorf("invalid usage period: %w", err))
			}
		}

		if usagePeriod.IsZero() {
			if billingCadence == nil || billingCadence.IsZero() {
				return nil, models.NewGenericValidationError(
					fmt.Errorf("metered entitlement requires usage_period when it cannot be inferred from billing_cadence"),
				)
			}

			usagePeriod = *billingCadence
		}

		tmpl := productcatalog.MeteredEntitlementTemplate{
			IsSoftLimit:     lo.FromPtr(metered.IsSoftLimit),
			IssueAfterReset: metered.Limit,
			UsagePeriod:     usagePeriod,
		}

		return productcatalog.NewEntitlementTemplateFrom(tmpl), nil

	case "static":
		// Extract the config's raw JSON bytes from the union so client values
		// survive untouched (no float64 round-trip), then wrap them in a JSON
		// string token: the domain-wide convention (shared with v1) stores the
		// config as JSON-encoded text, which subscription materialization
		// unwraps when instantiating the entitlement.
		rawEnt, err := e.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to read static entitlement template: %w", err)
		}

		var static struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.Unmarshal(rawEnt, &static); err != nil {
			return nil, fmt.Errorf("failed to read static entitlement template: %w", err)
		}

		if len(static.Config) == 0 {
			static.Config = json.RawMessage("null")
		}

		token, err := json.Marshal(string(static.Config))
		if err != nil {
			return nil, fmt.Errorf("failed to encode static entitlement config: %w", err)
		}

		return productcatalog.NewEntitlementTemplateFrom(productcatalog.StaticEntitlementTemplate{
			Config: token,
		}), nil

	case "boolean":
		return productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}), nil

	default:
		return nil, fmt.Errorf("unsupported entitlement type: %s", disc)
	}
}

func FromAPIBillingPrice(p api.BillingPrice, paymentTerm *api.BillingPricePaymentTerm) (*productcatalog.Price, error) {
	disc, err := p.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read price type: %w", err)
	}

	switch disc {
	case "free":
		return nil, nil

	case "flat":
		flat, err := p.AsBillingPriceFlat()
		if err != nil {
			return nil, fmt.Errorf("failed to read flat price: %w", err)
		}

		amount, err := decimal.NewFromString(flat.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid flat price amount: %w", err)
		}

		term := productcatalog.DefaultPaymentTerm
		if paymentTerm != nil {
			term = productcatalog.PaymentTermType(*paymentTerm)
		}

		return productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      amount,
			PaymentTerm: term,
		}), nil

	default:
		return nil, fmt.Errorf("toBillingPrice does not handle price type %q", disc)
	}
}

func FromAPIBillingPriceWithCommitments(p api.BillingPrice, commitments *api.BillingSpendCommitments) (*productcatalog.Price, error) {
	disc, err := p.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read price type: %w", err)
	}

	c, err := ParseCommitments(commitments)
	if err != nil {
		return nil, err
	}

	switch disc {
	case "unit":
		unit, err := p.AsBillingPriceUnit()
		if err != nil {
			return nil, fmt.Errorf("failed to read unit price: %w", err)
		}

		amount, err := decimal.NewFromString(unit.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid unit price amount: %w", err)
		}

		return productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount:      amount,
			Commitments: c,
		}), nil

	case "graduated":
		grad, err := p.AsBillingPriceGraduated()
		if err != nil {
			return nil, fmt.Errorf("failed to read graduated price: %w", err)
		}

		tiers, err := FromAPIBillingPriceTiers(grad.Tiers)
		if err != nil {
			return nil, fmt.Errorf("failed to convert graduated tiers: %w", err)
		}

		return productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode:        productcatalog.GraduatedTieredPrice,
			Tiers:       tiers,
			Commitments: c,
		}), nil

	case "volume":
		vol, err := p.AsBillingPriceVolume()
		if err != nil {
			return nil, fmt.Errorf("failed to read volume price: %w", err)
		}

		tiers, err := FromAPIBillingPriceTiers(vol.Tiers)
		if err != nil {
			return nil, fmt.Errorf("failed to convert volume tiers: %w", err)
		}

		return productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode:        productcatalog.VolumeTieredPrice,
			Tiers:       tiers,
			Commitments: c,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported usage-based price type: %s", disc)
	}
}

func ParseCommitments(c *api.BillingSpendCommitments) (productcatalog.Commitments, error) {
	if c == nil {
		return productcatalog.Commitments{}, nil
	}

	result := productcatalog.Commitments{}

	if c.MinimumAmount != nil {
		min, err := decimal.NewFromString(*c.MinimumAmount)
		if err != nil {
			return result, fmt.Errorf("invalid minimum amount: %w", err)
		}

		result.MinimumAmount = &min
	}

	if c.MaximumAmount != nil {
		max, err := decimal.NewFromString(*c.MaximumAmount)
		if err != nil {
			return result, fmt.Errorf("invalid maximum amount: %w", err)
		}

		result.MaximumAmount = &max
	}

	return result, nil
}

func FromAPIBillingPriceTiers(tiers []api.BillingPriceTier) ([]productcatalog.PriceTier, error) {
	result := make([]productcatalog.PriceTier, 0, len(tiers))

	for _, t := range tiers {
		tier := productcatalog.PriceTier{}

		if t.UpToAmount != nil {
			amount, err := decimal.NewFromString(*t.UpToAmount)
			if err != nil {
				return nil, fmt.Errorf("invalid tier up-to amount: %w", err)
			}

			tier.UpToAmount = &amount
		}

		if t.FlatPrice != nil {
			amount, err := decimal.NewFromString(t.FlatPrice.Amount)
			if err != nil {
				return nil, fmt.Errorf("invalid tier flat price amount: %w", err)
			}

			tier.FlatPrice = &productcatalog.PriceTierFlatPrice{Amount: amount}
		}

		if t.UnitPrice != nil {
			amount, err := decimal.NewFromString(t.UnitPrice.Amount)
			if err != nil {
				return nil, fmt.Errorf("invalid tier unit price amount: %w", err)
			}

			tier.UnitPrice = &productcatalog.PriceTierUnitPrice{Amount: amount}
		}

		result = append(result, tier)
	}

	return result, nil
}

func FromAPIBillingRateCardTaxConfig(tc api.BillingRateCardTaxConfig) *productcatalog.TaxConfig {
	result := &productcatalog.TaxConfig{
		TaxCodeID: &tc.Code.Id,
	}

	if tc.Behavior != nil {
		result.Behavior = lo.ToPtr(productcatalog.TaxBehavior(*tc.Behavior))
	}

	return result
}

func FromAPIBillingRateCardDiscounts(d api.BillingRateCardDiscounts) (productcatalog.Discounts, error) {
	result := productcatalog.Discounts{}

	if d.Percentage != nil {
		result.Percentage = &productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(float64(*d.Percentage)),
		}
	}

	if d.Usage != nil {
		qty, err := decimal.NewFromString(*d.Usage)
		if err != nil {
			return result, fmt.Errorf("invalid usage discount quantity: %w", err)
		}

		result.Usage = &productcatalog.UsageDiscount{
			Quantity: qty,
		}
	}

	return result, nil
}
