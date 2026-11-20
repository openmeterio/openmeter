package httpdriver

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/driver"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	plandriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// MapAPISubscriptionEditOperationToPatch maps a v1 edit operation to a patch. Legacy stripe.code
// values on added items are resolved to Tax Codes through the resolver, which may create them.
func MapAPISubscriptionEditOperationToPatch(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, apiPatch apiv1models.SubscriptionEditOperation) (subscription.Patch, error) {
	disc, err := apiPatch.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to get discriminator: %w", err)
	}

	switch disc {
	case string(apiv1models.EditSubscriptionAddItemOpAddItem):
		apiP, err := apiPatch.AsEditSubscriptionAddItem()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionAddItem: %w", err)
		}

		// Let's parse and validate value.
		// Fortunately TypeSpec to OpenAPI generation is utterly logical and consistent, so we have to work with a structurally identical but differently named type.
		planRC, err := productcataloghttp.AsRateCard(ctx, resolver, namespace, apiP.RateCard)
		if err != nil {
			return nil, fmt.Errorf("failed to cast to RateCard: %w", err)
		}

		sPRC := &plansubscription.RateCard{
			PhaseKey: apiP.PhaseKey,
			RateCard: planRC,
		}

		p := patch.PatchAddItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  planRC.Key(),
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput:     sPRC.ToCreateSubscriptionItemPlanInput(),
					CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
				},
			},
		}

		return p, nil
	case string(apiv1models.EditSubscriptionRemoveItemOpRemoveItem):
		apiP, err := apiPatch.AsEditSubscriptionRemoveItem()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionRemoveItem: %w", err)
		}

		p := patch.PatchRemoveItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  apiP.ItemKey,
		}

		return p, nil
	case string(apiv1models.EditSubscriptionAddPhaseOpAddPhase):
		apiP, err := apiPatch.AsEditSubscriptionAddPhase()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionAddPhase: %w", err)
		}

		var sa datetime.ISODuration
		if apiP.Phase.StartAfter != nil {
			saStr := datetime.ISODurationString(*apiP.Phase.StartAfter)
			sa, err = saStr.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse start after: %w", err)
			}
		}

		var dur *datetime.ISODuration

		if apiP.Phase.Duration != nil {
			dS := datetime.ISODurationString(*apiP.Phase.Duration)
			d, err := dS.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse duration: %w", err)
			}

			dur = &d
		}

		p := patch.PatchAddPhase{
			PhaseKey: apiP.Phase.Key,
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				Duration: dur,
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:    apiP.Phase.Key,
					StartAfter:  sa,
					Name:        apiP.Phase.Name,
					Description: apiP.Phase.Description,
				},
				CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
			},
		}

		return p, nil
	case string(apiv1models.EditSubscriptionRemovePhaseOpRemovePhase):
		apiP, err := apiPatch.AsEditSubscriptionRemovePhase()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionRemovePhase: %w", err)
		}

		var shift subscription.RemoveSubscriptionPhaseShifting

		switch apiP.Shift {
		case apiv1models.RemovePhaseShiftingNext:
			shift = subscription.RemoveSubscriptionPhaseShiftNext
		case apiv1models.RemovePhaseShiftingPrev:
			shift = subscription.RemoveSubscriptionPhaseShiftPrev
		default:
			return nil, fmt.Errorf("unknown shift value: %s", apiP.Shift)
		}

		p := patch.PatchRemovePhase{
			PhaseKey: apiP.PhaseKey,
			RemoveInput: subscription.RemoveSubscriptionPhaseInput{
				Shift: shift,
			},
		}

		return p, nil
	case string(apiv1models.EditSubscriptionStretchPhaseOpStretchPhase):
		apiP, err := apiPatch.AsEditSubscriptionStretchPhase()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionStretchPhase: %w", err)
		}

		durStr := datetime.ISODurationString(apiP.ExtendBy)
		d, err := durStr.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration: %w", err)
		}

		p := patch.PatchStretchPhase{
			PhaseKey: apiP.PhaseKey,
			Duration: d,
		}

		return p, nil
	case string(apiv1models.EditSubscriptionUnscheduleEditOpUnscheduleEdit):
		p := patch.PatchUnscheduleEdit{}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown discriminator: %s", disc)
	}
}

func MapSubscriptionToAPI(sub subscription.Subscription) apiv1models.Subscription {
	var ref *apiv1models.PlanReference

	if sub.PlanRef != nil {
		ref = &apiv1models.PlanReference{
			Id:      sub.PlanRef.Id,
			Key:     sub.PlanRef.Key,
			Version: sub.PlanRef.Version,
		}
	}

	var annotations *apiv1models.Annotations
	if len(sub.Annotations) > 0 {
		annotations = lo.ToPtr(apiv1models.Annotations(sub.Annotations))
	}

	costBasisPins := make([]apiv1models.SubscriptionCostBasisPin, 0, len(sub.CostBasisPins))
	for _, pin := range sub.CostBasisPins {
		costBasisPins = append(costBasisPins, apiv1models.SubscriptionCostBasisPin{
			CustomCurrencyId: pin.CustomCurrencyID,
			InvoiceCurrency:  pin.InvoiceCurrency.String(),
			CostBasisId:      pin.CostBasis.ID,
		})
	}

	return apiv1models.Subscription{
		Id:         sub.ID,
		ActiveFrom: sub.ActiveFrom,
		ActiveTo:   sub.ActiveTo,
		CustomerId: sub.CustomerId,
		Currency:   sub.InvoiceCurrency.String(),
		CostBasisMode: apiv1models.SubscriptionCostBasisMode(
			sub.CostBasisMode.OrDefault(),
		),
		CostBasisPins: costBasisPins,
		Description:   sub.Description,
		Name:          sub.Name,
		Status:        apiv1models.SubscriptionStatus(sub.GetStatusAt(clock.Now())),
		Plan:          ref,
		Metadata:      lo.EmptyableToPtr(apiv1models.Metadata(sub.Metadata)),
		Annotations:   annotations,
		CreatedAt:     sub.CreatedAt,
		UpdatedAt:     sub.UpdatedAt,
		DeletedAt:     sub.DeletedAt,
		Alignment: &apiv1models.Alignment{
			BillablesMustAlign: lo.ToPtr(true), // TODO(OM-1249): this field is deprecated on API, we'll get rid of it eventually
		},
		BillingCadence:  sub.BillingCadence.String(),
		BillingAnchor:   sub.BillingAnchor,
		BillingStartsAt: sub.BillingStartsAt,
		ProRatingConfig: &apiv1models.ProRatingConfig{
			Enabled: sub.ProRatingConfig.Enabled,
			Mode:    apiv1models.ProRatingMode(sub.ProRatingConfig.Mode),
		},
		SettlementMode: apiv1models.BillingSettlementMode(sub.SettlementMode),
	}
}

// MapSubscriptionItemToAPI maps an item to its v1 shape. taxCodes must hold every Tax Code the
// items reference: a missing entry is an error.
func MapSubscriptionItemToAPI(item subscription.SubscriptionItemView, taxCodes map[string]*taxcode.TaxCode) (apiv1models.SubscriptionItem, error) {
	var included *apiv1models.SubscriptionItemIncluded

	if item.Feature != nil {
		feature, err := productcatalogdriver.MapFeatureToResponse(*item.Feature)
		if err != nil {
			return apiv1models.SubscriptionItem{}, fmt.Errorf("failed to map feature to API: %w", err)
		}

		included = &apiv1models.SubscriptionItemIncluded{
			Feature: feature,
		}
	}

	if included != nil && item.Entitlement != nil {
		apiEnt, err := entitlementdriver.Parser.ToAPIGeneric(&item.Entitlement.Entitlement)
		if err != nil {
			return apiv1models.SubscriptionItem{}, err
		}

		included.Entitlement = apiEnt
	}

	var tx *apiv1models.TaxConfig

	if item.SubscriptionItem.RateCard.AsMeta().TaxConfig != nil {
		txv, err := productcataloghttp.ToAPITaxConfig(*item.SubscriptionItem.RateCard.AsMeta().TaxConfig, taxCodes)
		if err != nil {
			return apiv1models.SubscriptionItem{}, err
		}

		tx = &txv
	}

	var pr *apiv1models.RateCardUsageBasedPrice

	if item.SubscriptionItem.RateCard.AsMeta().Price != nil {
		prc, err := productcataloghttp.FromRateCardUsageBasedPrice(*item.SubscriptionItem.RateCard.AsMeta().Price)
		if err != nil {
			return apiv1models.SubscriptionItem{}, err
		}

		pr = &prc
	}

	var featureKey *string
	if feature := item.SubscriptionItem.RateCard.AsMeta().Feature; feature != nil {
		featureKey = feature.Key
	}

	result := apiv1models.SubscriptionItem{
		ActiveFrom:     item.SubscriptionItem.ActiveFrom,
		ActiveTo:       item.SubscriptionItem.ActiveTo,
		BillingCadence: (*string)(item.SubscriptionItem.RateCard.GetBillingCadence().ISOStringPtrOrNil()),
		CreatedAt:      item.SubscriptionItem.CreatedAt,
		DeletedAt:      item.SubscriptionItem.DeletedAt,
		Description:    item.SubscriptionItem.Description,
		Id:             item.SubscriptionItem.ID,
		Included:       included,
		Key:            item.SubscriptionItem.Key,
		FeatureKey:     featureKey,
		Metadata:       lo.EmptyableToPtr(apiv1models.Metadata(item.SubscriptionItem.Metadata)),
		Name:           item.SubscriptionItem.Name,
		Price:          pr,
		Discounts:      productcataloghttp.FromDiscounts(item.SubscriptionItem.RateCard.AsMeta().Discounts),
		TaxConfig:      tx,
		UpdatedAt:      item.SubscriptionItem.UpdatedAt,
	}

	if currency := item.SubscriptionItem.RateCard.AsMeta().Currency; currency != nil {
		result.Currency = lo.ToPtr(currency.GetCode().String())
	}

	return result, nil
}

func MapAPITimingToTiming(apiTiming apiv1models.SubscriptionTiming) (subscription.Timing, error) {
	var res subscription.Timing

	t, err := apiTiming.AsSubscriptionTiming1()
	if err != nil {
		e, err := apiTiming.AsSubscriptionTimingEnum()
		if err != nil {
			return res, fmt.Errorf("failed to cast to SubscriptionChangeTiming: %w", err)
		}

		res.Enum = lo.ToPtr(subscription.TimingEnum(e))
	} else {
		res.Custom = &t
	}

	return res, nil
}

// We map the items as follows:
// - for the current phase, the API will only return the active item for each key
// - for past phases, the API will return the last item for each key
// - for future phases, the API will return the first version
func MapSubscriptionPhaseToAPI(subView subscription.SubscriptionView, phaseView subscription.SubscriptionPhaseView, taxCodes map[string]*taxcode.TaxCode) (apiv1models.SubscriptionPhaseExpanded, error) {
	var endOfPhase *time.Time
	if dur, err := subView.Spec.GetPhaseCadence(phaseView.SubscriptionPhase.Key); err == nil {
		endOfPhase = dur.ActiveTo
	}

	now := clock.Now()
	currPhase, currExists := subView.Spec.GetCurrentPhaseAt(now)

	flatItems := lo.Flatten(lo.Values(phaseView.ItemsByKey))
	apiItems := make([]apiv1models.SubscriptionItem, 0, len(flatItems))

	apiItemTimelines := make(map[string][]apiv1models.SubscriptionItem)

	var relativePhaseTime string

	if currExists && currPhase.PhaseKey == phaseView.SubscriptionPhase.Key {
		relativePhaseTime = "current"
	} else if phaseView.SubscriptionPhase.ActiveFrom.After(now) {
		relativePhaseTime = "future"
	} else {
		relativePhaseTime = "past"
	}

	for key, items := range phaseView.ItemsByKey {
		// Let's add the items to the timeline
		timeline, err := slicesx.MapWithErr(items, func(item subscription.SubscriptionItemView) (apiv1models.SubscriptionItem, error) {
			apiItem, err := MapSubscriptionItemToAPI(item, taxCodes)
			if err != nil {
				return apiv1models.SubscriptionItem{}, err
			}

			return apiItem, nil
		})
		if err != nil {
			return apiv1models.SubscriptionPhaseExpanded{}, err
		}

		apiItemTimelines[key] = timeline

		// Then let's add the items to the flat list
		switch relativePhaseTime {
		// If this is the current phase
		case "current":
			// Let's find if there's a current item, if so add that to the output
			curr := slices.IndexFunc(items, func(i subscription.SubscriptionItemView) bool {
				return i.SubscriptionItem.IsActiveAt(now)
			})

			if curr != -1 {
				apiItem, err := MapSubscriptionItemToAPI(items[curr], taxCodes)
				if err != nil {
					return apiv1models.SubscriptionPhaseExpanded{}, err
				}

				apiItems = append(apiItems, apiItem)
			}

			continue
		// If this is a future phase lets add the first item
		case "future":
			if len(items) > 0 {
				apiItem, err := MapSubscriptionItemToAPI(items[0], taxCodes)
				if err != nil {
					return apiv1models.SubscriptionPhaseExpanded{}, err
				}

				apiItems = append(apiItems, apiItem)
			}

			continue
		// If this is a past phase
		case "past":
			// Let's find the last item
			if len(items) > 0 {
				apiItem, err := MapSubscriptionItemToAPI(items[len(items)-1], taxCodes)
				if err != nil {
					return apiv1models.SubscriptionPhaseExpanded{}, err
				}

				apiItems = append(apiItems, apiItem)
			}

			continue
		default:
			return apiv1models.SubscriptionPhaseExpanded{}, fmt.Errorf("no logical branch enetered: %s", relativePhaseTime)
		}
	}

	return apiv1models.SubscriptionPhaseExpanded{
		ActiveFrom:    phaseView.SubscriptionPhase.ActiveFrom,
		ActiveTo:      endOfPhase,
		CreatedAt:     phaseView.SubscriptionPhase.CreatedAt,
		UpdatedAt:     phaseView.SubscriptionPhase.UpdatedAt,
		DeletedAt:     phaseView.SubscriptionPhase.DeletedAt,
		Description:   phaseView.SubscriptionPhase.Description,
		Discounts:     nil, // TODO: add discounts
		Id:            phaseView.SubscriptionPhase.ID,
		Items:         apiItems,
		ItemTimelines: apiItemTimelines,
		Key:           phaseView.SubscriptionPhase.Key,
		Metadata:      lo.EmptyableToPtr(apiv1models.Metadata(phaseView.SubscriptionPhase.Metadata)),
		Name:          phaseView.SubscriptionPhase.Name,
	}, nil
}

// MapSubscriptionViewToAPI maps a view to its v1 shape. taxCodes must hold every Tax Code
// the view's items reference (see taxCodesForView): a missing entry is an error.
func MapSubscriptionViewToAPI(view subscription.SubscriptionView, taxCodes map[string]*taxcode.TaxCode) (apiv1models.SubscriptionExpanded, error) {
	apiSub := MapSubscriptionToAPI(view.Subscription)
	alg := apiv1models.SubscriptionAlignment{
		BillablesMustAlign: lo.ToPtr(true), // this field is deprecated on API, we'll get rid of it eventually
	}

	{
		period, err := view.Spec.GetAlignedBillingPeriodAt(clock.Now())
		if err != nil {
			if !subscription.IsErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(err) {
				return apiv1models.SubscriptionExpanded{}, err
			}
		} else {
			alg.CurrentAlignedBillingPeriod = &apiv1models.Period{
				From: period.From,
				To:   period.To,
			}
		}
	}

	base := apiv1models.SubscriptionExpanded{
		ActiveFrom:      apiSub.ActiveFrom,
		ActiveTo:        apiSub.ActiveTo,
		CreatedAt:       apiSub.CreatedAt,
		Currency:        apiSub.Currency,
		CostBasisMode:   apiSub.CostBasisMode,
		CostBasisPins:   apiSub.CostBasisPins,
		CustomerId:      apiSub.CustomerId,
		DeletedAt:       apiSub.DeletedAt,
		Description:     apiSub.Description,
		Id:              apiSub.Id,
		Metadata:        apiSub.Metadata,
		Annotations:     apiSub.Annotations,
		ProRatingConfig: apiSub.ProRatingConfig,
		BillingAnchor:   apiSub.BillingAnchor,
		BillingStartsAt: apiSub.BillingStartsAt,
		BillingCadence:  apiSub.BillingCadence,
		Name:            apiSub.Name,
		Phases:          nil,
		Plan:            apiSub.Plan,
		UpdatedAt:       apiSub.UpdatedAt,
		Status:          apiSub.Status,
		Alignment:       &alg,
		SettlementMode:  apiSub.SettlementMode,
	}

	phases := make([]apiv1models.SubscriptionPhaseExpanded, 0, len(view.Phases))
	for _, phase := range view.Phases {
		phaseAPI, err := MapSubscriptionPhaseToAPI(view, phase, taxCodes)
		if err != nil {
			return base, err
		}

		phases = append(phases, phaseAPI)
	}

	base.Phases = phases

	return base, nil
}

// CustomPlanToCreatePlanRequest maps a v1 custom plan to a plan create request. Legacy stripe.code
// values are resolved to Tax Codes through resolver, which may create them.
func CustomPlanToCreatePlanRequest(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, a apiv1models.CustomPlanInput, namespace string) (plandriver.CreatePlanRequest, error) {
	var err error

	settlementMode := productcatalog.SettlementMode(lo.FromPtr(a.SettlementMode))
	if settlementMode == "" {
		settlementMode = productcatalog.CreditThenInvoiceSettlementMode
	}

	req := plandriver.CreatePlanRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            a.Name,
				Description:     a.Description,
				Metadata:        lo.FromPtr(a.Metadata),
				ProRatingConfig: asProRatingConfig(a.ProRatingConfig),
				SettlementMode:  settlementMode,
			},
			Phases: nil,
		},
	}

	currencyCode := currencyx.FiatCode(a.Currency)
	if err = currencyCode.Validate(); err != nil {
		return req, fmt.Errorf("invalid CurrencyCode: %w", err)
	}

	req.Currency = currencies.NewCurrencyReference(currencyx.Code(currencyCode))

	req.PlanMeta.BillingCadence, err = datetime.ISODurationString(a.BillingCadence).Parse()
	if err != nil {
		return req, fmt.Errorf("invalid BillingCadence: %w", err)
	}

	if len(a.Phases) > 0 {
		req.Phases = make([]productcatalog.Phase, 0, len(a.Phases))

		for _, phase := range a.Phases {
			planPhase, err := plandriver.AsPlanPhase(ctx, resolver, namespace, phase)
			if err != nil {
				return req, fmt.Errorf("failed to cast PlanPhase: %w", err)
			}

			req.Phases = append(req.Phases, planPhase)
		}
	}

	return req, nil
}

// asProRatingConfig converts API ProRatingConfig to domain ProRatingConfig
func asProRatingConfig(p *apiv1models.ProRatingConfig) productcatalog.ProRatingConfig {
	if p == nil {
		// Return default configuration when not provided
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}

	return productcatalog.ProRatingConfig{
		Enabled: p.Enabled,
		Mode:    productcatalog.ProRatingMode(p.Mode),
	}
}
