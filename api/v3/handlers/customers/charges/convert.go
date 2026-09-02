package charges

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingcommon"
	"github.com/openmeterio/openmeter/api/v3/handlers/billinginvoices"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingprofiles"
	"github.com/openmeterio/openmeter/api/v3/handlers/customers"
	"github.com/openmeterio/openmeter/api/v3/handlers/features"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/handlers/subscriptions"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// FromAPICustomerChargesSortField validates a v3 (snake_case) customer charges
// sort field, returning it unchanged because the charges adapter matches these
// wire strings directly. Returns a 400 for any unsupported field.
func FromAPICustomerChargesSortField(ctx context.Context, field string) (string, error) {
	switch field {
	case "id", "created_at", "service_period.from", "billing_period.from":
		return field, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(
			ctx, field, "id", "created_at", "service_period.from", "billing_period.from",
		)
	}
}

// ConvertMetadataToLabels converts domain metadata to API labels.
var ConvertMetadataToLabels = labels.FromMetadata[models.Metadata]

// convertFlatFeeChargeToAPI maps the flat-fee branch of a CustomerCharge to
// the API representation. Expanded entities select the full union branches,
// references are used otherwise.
func convertFlatFeeChargeToAPI(charge billingcharges.CustomerCharge, expands meta.Expands) (api.BillingChargeFlatFee, error) {
	flatFee, err := charge.AsFlatFeeCharge()
	if err != nil {
		return api.BillingChargeFlatFee{}, fmt.Errorf("converting flat fee charge: %w", err)
	}

	// The domain status carries detailed sub-states (e.g.
	// "active.realization.processing") that the API enum does not admit; map
	// to the coarse charge status like the usage-based branch does.
	status, err := flatFee.Status.ToMetaChargeStatus()
	if err != nil {
		return api.BillingChargeFlatFee{}, fmt.Errorf("converting flat fee charge status: %w", err)
	}

	intent := flatFee.ChargeBase.Intent.GetEffectiveIntent()

	price := api.BillingPriceFlat{
		Amount: intent.AmountBeforeProration.String(),
		Type:   api.BillingPriceFlatTypeFlat,
	}

	feature, err := convertExpandedChargeFeatureToAPI(flatFee.State.FeatureID, charge.Feature)
	if err != nil {
		return api.BillingChargeFlatFee{}, err
	}

	customer, err := convertChargeCustomerToAPI(flatFee.ChargeBase.Intent.GetCustomerID(), charge.Customer)
	if err != nil {
		return api.BillingChargeFlatFee{}, err
	}

	subscription, err := convertChargeSubscriptionToAPI(flatFee.ChargeBase.Intent.GetSubscription(), charge.Subscription)
	if err != nil {
		return api.BillingChargeFlatFee{}, err
	}

	realizations, err := convertFlatFeeRealizationsToAPI(charge.FlatFeeRealizations, expands)
	if err != nil {
		return api.BillingChargeFlatFee{}, fmt.Errorf("converting realizations: %w", err)
	}

	return api.BillingChargeFlatFee{
		AdvanceAfter:           flatFee.State.AdvanceAfter,
		AmountAfterProration:   ConvertDecimalToCurrencyAmount(flatFee.ChargeBase.State.AmountAfterProration, flatFee.ChargeBase.Intent.GetCurrency().GetCode()),
		BillingPeriod:          ConvertClosedPeriodToAPI(intent.BillingPeriod),
		CreatedAt:              flatFee.ChargeBase.ManagedResource.ManagedModel.CreatedAt,
		Currency:               api.CurrencyCode(flatFee.ChargeBase.Intent.GetCurrency().GetCode()),
		Customer:               customer,
		DeletedAt:              flatFee.ChargeBase.ManagedResource.ManagedModel.DeletedAt,
		Description:            intent.Description,
		Discounts:              convertFlatFeeDiscounts(intent.PercentageDiscounts),
		Feature:                feature,
		FullServicePeriod:      ConvertClosedPeriodToAPI(intent.FullServicePeriod),
		Id:                     flatFee.ChargeBase.ManagedResource.ID,
		InvoiceAt:              intent.InvoiceAt,
		Labels:                 ConvertMetadataToLabels(intent.Metadata),
		LifecycleController:    ConvertLifecycleControllerToAPI(intent.ManagedBy, WithManualOverride(flatFee.ChargeBase.Intent.HasOverrideLayer())),
		Name:                   intent.Name,
		PaymentTerm:            api.BillingPricePaymentTerm(intent.PaymentTerm),
		Price:                  price,
		ProrationConfiguration: ConvertProRatingConfigToAPI(intent.ProRating),
		Realizations:           realizations,
		ServicePeriod:          ConvertClosedPeriodToAPI(intent.ServicePeriod),
		SettlementMode:         api.BillingSettlementMode(flatFee.ChargeBase.Intent.GetSettlementMode()),
		Status:                 api.BillingChargeStatus(status),
		Subscription:           subscription,
		SystemIntent:           toAPIBillingChargeFlatFeeSystemIntent(flatFee.ChargeBase.Intent),
		TaxConfig:              convertTaxCodeConfigToAPI(intent.TaxConfig),
		Type:                   api.BillingChargeFlatFeeTypeFlatFee,
		UniqueReferenceId:      flatFee.ChargeBase.Intent.GetUniqueReferenceID(),
		UpdatedAt:              flatFee.ChargeBase.ManagedResource.ManagedModel.UpdatedAt,
	}, nil
}

// convertUsageBasedChargeToAPI maps the usage-based branch of a CustomerCharge
// to the API representation. Expanded entities select the full union branches,
// references are used otherwise.
func convertUsageBasedChargeToAPI(charge billingcharges.CustomerCharge, expands meta.Expands) (api.BillingChargeUsageBased, error) {
	usageBasedFee, err := charge.AsUsageBasedCharge()
	if err != nil {
		return api.BillingChargeUsageBased{}, fmt.Errorf("converting usage based charge: %w", err)
	}

	status, err := ConvertUsageBasedStatusToAPI(usageBasedFee.ChargeBase.Status)
	if err != nil {
		return api.BillingChargeUsageBased{}, fmt.Errorf("converting usage based charge status: %w", err)
	}

	intent := usageBasedFee.ChargeBase.Intent.GetEffectiveIntent()

	ratingConfiguration, err := toAPIBillingUsageBasedRatingConfiguration(intent.IntentMutableFields)
	if err != nil {
		return api.BillingChargeUsageBased{}, fmt.Errorf("converting rating configuration: %w", err)
	}

	systemIntent, err := toAPIBillingChargeUsageBasedSystemIntent(usageBasedFee.ChargeBase.Intent)
	if err != nil {
		return api.BillingChargeUsageBased{}, fmt.Errorf("converting system intent: %w", err)
	}

	feature, err := convertExpandedChargeFeatureToAPI(&usageBasedFee.State.FeatureID, charge.Feature)
	if err != nil {
		return api.BillingChargeUsageBased{}, err
	}

	if feature == nil {
		return api.BillingChargeUsageBased{}, fmt.Errorf("feature reference is required for usage-based charges")
	}

	customer, err := convertChargeCustomerToAPI(usageBasedFee.ChargeBase.Intent.GetCustomerID(), charge.Customer)
	if err != nil {
		return api.BillingChargeUsageBased{}, err
	}

	subscription, err := convertChargeSubscriptionToAPI(usageBasedFee.ChargeBase.Intent.GetSubscription(), charge.Subscription)
	if err != nil {
		return api.BillingChargeUsageBased{}, err
	}

	realizations, err := convertUsageBasedRealizationsToAPI(charge.UsageBasedRealizations, expands)
	if err != nil {
		return api.BillingChargeUsageBased{}, fmt.Errorf("converting realizations: %w", err)
	}

	// The contract promises the charge-level usage under the real_time_usage
	// expand; the domain carries the live cumulative read alongside the rated
	// realtime totals.
	var usage *api.Numeric
	if usageBasedFee.Expands.RealtimeQuantity != nil {
		usage = lo.ToPtr(usageBasedFee.Expands.RealtimeQuantity.String())
	}

	return api.BillingChargeUsageBased{
		AdvanceAfter:        usageBasedFee.State.AdvanceAfter,
		BillingPeriod:       ConvertClosedPeriodToAPI(intent.BillingPeriod),
		Commitments:         ratingConfiguration.Commitments,
		CreatedAt:           usageBasedFee.ChargeBase.ManagedResource.ManagedModel.CreatedAt,
		Currency:            api.CurrencyCode(usageBasedFee.ChargeBase.Intent.GetCurrency().GetCode()),
		Customer:            customer,
		DeletedAt:           usageBasedFee.ChargeBase.ManagedResource.ManagedModel.DeletedAt,
		Description:         intent.Description,
		Discounts:           convertUsageBasedDiscounts(intent.Discounts),
		Feature:             lo.FromPtr(feature),
		FullServicePeriod:   ConvertClosedPeriodToAPI(intent.FullServicePeriod),
		Id:                  usageBasedFee.ChargeBase.ManagedResource.ID,
		InvoiceAt:           intent.InvoiceAt,
		Labels:              ConvertMetadataToLabels(intent.Metadata),
		LifecycleController: ConvertLifecycleControllerToAPI(intent.ManagedBy, WithManualOverride(usageBasedFee.ChargeBase.Intent.HasOverrideLayer())),
		Name:                intent.Name,
		Price:               ratingConfiguration.Price,
		Realizations:        realizations,
		ServicePeriod:       ConvertClosedPeriodToAPI(intent.ServicePeriod),
		SettlementMode:      api.BillingSettlementMode(usageBasedFee.ChargeBase.Intent.GetSettlementMode()),
		Status:              lo.FromPtr(status),
		Subscription:        subscription,
		SystemIntent:        systemIntent,
		TaxConfig:           convertTaxCodeConfigToAPI(intent.TaxConfig),
		Totals:              convertUsageBasedChargeTotals(usageBasedFee),
		Type:                api.BillingChargeUsageBasedTypeUsageBased,
		UnitConfig:          ratingConfiguration.UnitConfig,
		UniqueReferenceId:   usageBasedFee.ChargeBase.Intent.GetUniqueReferenceID(),
		UpdatedAt:           usageBasedFee.ChargeBase.ManagedResource.ManagedModel.UpdatedAt,
		Usage:               usage,
	}, nil
}

func toAPIBillingChargeFlatFeeSystemIntent(intent flatfee.OverridableIntent) *api.BillingChargeFlatFeeSystemIntent {
	if !intent.HasOverrideLayer() || intent.GetBaseManagedBy() == billing.ManuallyManagedLine {
		return nil
	}

	baseIntent := intent.GetBaseIntent()

	return &api.BillingChargeFlatFeeSystemIntent{
		AmountBeforeProration:  ConvertDecimalToCurrencyAmount(baseIntent.AmountBeforeProration, baseIntent.Currency.GetCode()),
		BillingPeriod:          ConvertClosedPeriodToAPI(baseIntent.BillingPeriod),
		DeletedAt:              baseIntent.IntentDeletedAt,
		Description:            baseIntent.Description,
		Discounts:              convertFlatFeeDiscounts(baseIntent.PercentageDiscounts),
		FullServicePeriod:      ConvertClosedPeriodToAPI(baseIntent.FullServicePeriod),
		InvoiceAt:              baseIntent.InvoiceAt,
		Labels:                 ConvertMetadataToLabels(baseIntent.Metadata),
		Name:                   baseIntent.Name,
		PaymentTerm:            api.BillingPricePaymentTerm(baseIntent.PaymentTerm),
		ProrationConfiguration: ConvertProRatingConfigToAPI(baseIntent.ProRating),
		ServicePeriod:          ConvertClosedPeriodToAPI(baseIntent.ServicePeriod),
	}
}

func toAPIBillingChargeUsageBasedSystemIntent(intent usagebased.OverridableIntent) (*api.BillingChargeUsageBasedSystemIntent, error) {
	if !intent.HasOverrideLayer() || intent.GetBaseManagedBy() == billing.ManuallyManagedLine {
		return nil, nil
	}

	baseIntent := intent.GetBaseIntent()

	ratingConfiguration, err := toAPIBillingUsageBasedRatingConfiguration(baseIntent.IntentMutableFields)
	if err != nil {
		return nil, fmt.Errorf("converting rating configuration: %w", err)
	}

	return &api.BillingChargeUsageBasedSystemIntent{
		BillingPeriod:     ConvertClosedPeriodToAPI(baseIntent.BillingPeriod),
		Commitments:       ratingConfiguration.Commitments,
		DeletedAt:         baseIntent.IntentDeletedAt,
		Description:       baseIntent.Description,
		Discounts:         convertUsageBasedDiscounts(baseIntent.Discounts),
		FullServicePeriod: ConvertClosedPeriodToAPI(baseIntent.FullServicePeriod),
		InvoiceAt:         baseIntent.InvoiceAt,
		Labels:            ConvertMetadataToLabels(baseIntent.Metadata),
		Name:              baseIntent.Name,
		Price:             ratingConfiguration.Price,
		ServicePeriod:     ConvertClosedPeriodToAPI(baseIntent.ServicePeriod),
		UnitConfig:        ratingConfiguration.UnitConfig,
	}, nil
}

func convertFeatureIDToReference(id *string) *api.FeatureReference {
	if id == nil {
		return nil
	}
	return &api.FeatureReference{
		Id: *id,
	}
}

func convertSubscriptionToReference(source *meta.SubscriptionReference) (*api.SubscriptionOrReference, error) {
	if source == nil {
		return nil, nil
	}
	var result api.SubscriptionOrReference
	if err := result.FromBillingSubscriptionReference(ConvertSubscriptionRefToAPI(*source)); err != nil {
		return nil, fmt.Errorf("converting subscription reference: %w", err)
	}
	return &result, nil
}

func convertChargeSubscriptionToAPI(source *meta.SubscriptionReference, expanded *subscription.Subscription) (*api.SubscriptionOrReference, error) {
	if source == nil {
		return nil, nil
	}

	if expanded != nil {
		var out api.SubscriptionOrReference

		sub := subscriptions.ToAPIBillingSubscriptionBase(*expanded)

		if err := out.FromBillingSubscription(sub); err != nil {
			return nil, fmt.Errorf("setting subscription union: %w", err)
		}

		return &out, nil
	}

	return convertSubscriptionToReference(source)
}

func convertChargeCustomerToAPI(id string, expanded *customer.Customer) (api.CustomerOrReference, error) {
	if expanded != nil {
		var out api.CustomerOrReference
		if err := out.FromBillingCustomer(customers.ToAPIBillingCustomer(*expanded)); err != nil {
			return out, fmt.Errorf("setting customer union: %w", err)
		}

		return out, nil
	}

	return convertCustomerIDToReference(id)
}

func convertExpandedChargeFeatureToAPI(featureID *string, expanded *feature.Feature) (*api.FeatureOrReference, error) {
	var feature api.FeatureOrReference

	if expanded != nil {
		apiFeature, err := features.ConvertFeatureToAPI(*expanded)
		if err != nil {
			return nil, fmt.Errorf("converting feature: %w", err)
		}

		if err := feature.FromFeature(apiFeature); err != nil {
			return nil, fmt.Errorf("feature mapping failed: %w", err)
		}
	} else if ref := convertFeatureIDToReference(featureID); ref != nil {
		if err := feature.FromFeatureReference(*ref); err != nil {
			return nil, fmt.Errorf("feature mapping failed: %w", err)
		}
	} else {
		return nil, nil
	}

	return &feature, nil
}

func convertFlatFeeRealizationsToAPI(realizations []billingcharges.CustomerChargeFlatFeeRealization, expands meta.Expands) ([]api.BillingChargeRealization, error) {
	out := make([]api.BillingChargeRealization, 0, len(realizations))

	for _, resolved := range realizations {
		if resolved.Run == nil {
			// The outstanding entry is not a persisted run, so it has no ID,
			// invoice, line, or payment. Flat fees are not metered, so no entry
			// carries usage.
			out = append(out, api.BillingChargeRealization{
				ServicePeriod: ConvertClosedPeriodToAPI(resolved.ServicePeriod),
				Type:          api.BillingChargeRealizationTypeOutstanding,
			})
			continue
		}

		run := resolved.Run

		realizationType, err := convertFlatFeeRealizationTypeToAPI(run.Type, resolved.Voided)
		if err != nil {
			return nil, fmt.Errorf("realization run %s: %w", run.ID.ID, err)
		}

		invoice, err := convertRealizationInvoiceToAPI(run.InvoiceID, resolved.Invoice)
		if err != nil {
			return nil, fmt.Errorf("realization run %s: %w", run.ID.ID, err)
		}

		detailedLines, err := convertFlatFeeDetailedLinesToAPI(run.DetailedLines)
		if err != nil {
			return nil, fmt.Errorf("realization run %s: %w", run.ID.ID, err)
		}

		realization := api.BillingChargeRealization{
			DetailedLines: detailedLines,
			Id:            lo.ToPtr(run.ID.ID),
			Invoice:       invoice,
			LineId:        run.LineID,
			Payment:       convertRealizationPaymentToAPI(run.Payment),
			ServicePeriod: ConvertClosedPeriodToAPI(resolved.ServicePeriod),
			Type:          realizationType,
		}

		if expands.Has(meta.ExpandRealizationTotals) {
			realization.Totals = lo.ToPtr(ToAPIBillingTotals(run.Totals))
		}

		out = append(out, realization)
	}

	return out, nil
}

func convertUsageBasedRealizationsToAPI(realizations []billingcharges.CustomerChargeUsageBasedRealization, expands meta.Expands) ([]api.BillingChargeRealization, error) {
	out := make([]api.BillingChargeRealization, 0, len(realizations))

	for _, resolved := range realizations {
		if resolved.Run == nil {
			// The outstanding entry is not a persisted run, so it has no ID,
			// invoice, line, or payment. Its quantity is the not-yet-booked
			// remainder of the live read under the real_time_usage expand, zero
			// otherwise.
			out = append(out, api.BillingChargeRealization{
				ServicePeriod: ConvertClosedPeriodToAPI(resolved.ServicePeriod),
				Type:          api.BillingChargeRealizationTypeOutstanding,
				Usage:         lo.ToPtr(resolved.Quantity.String()),
			})
			continue
		}

		run := resolved.Run

		realizationType, err := convertUsageBasedRealizationTypeToAPI(run.Type, resolved.Voided)
		if err != nil {
			return nil, fmt.Errorf("realization run %s: %w", run.ID.ID, err)
		}

		invoice, err := convertRealizationInvoiceToAPI(run.InvoiceID, resolved.Invoice)
		if err != nil {
			return nil, fmt.Errorf("realization run %s: %w", run.ID.ID, err)
		}

		detailedLines, err := convertUsageBasedDetailedLinesToAPI(run.DetailedLines)
		if err != nil {
			return nil, fmt.Errorf("realization run %s: %w", run.ID.ID, err)
		}

		realization := api.BillingChargeRealization{
			DetailedLines: detailedLines,
			Id:            lo.ToPtr(run.ID.ID),
			Invoice:       invoice,
			LineId:        run.LineID,
			Payment:       convertRealizationPaymentToAPI(run.Payment),
			ServicePeriod: ConvertClosedPeriodToAPI(resolved.ServicePeriod),
			Type:          realizationType,
			Usage:         lo.ToPtr(resolved.Quantity.String()),
		}

		if expands.Has(meta.ExpandRealizationTotals) {
			realization.Totals = lo.ToPtr(ToAPIBillingTotals(run.Totals))
		}

		out = append(out, realization)
	}

	return out, nil
}

// convertFlatFeeRealizationTypeToAPI maps a flat fee run type to the API enum.
func convertFlatFeeRealizationTypeToAPI(t flatfee.RealizationRunType, voided bool) (api.BillingChargeRealizationType, error) {
	if voided {
		return api.BillingChargeRealizationTypeVoided, nil
	}

	switch t {
	case flatfee.RealizationRunTypeFinalRealization:
		return api.BillingChargeRealizationTypeFinalRealization, nil
	default:
		return "", fmt.Errorf("unsupported flat fee realization run type: %s", t)
	}
}

// convertUsageBasedRealizationTypeToAPI maps a usage-based run type to the API
// enum.
func convertUsageBasedRealizationTypeToAPI(t usagebased.RealizationRunType, voided bool) (api.BillingChargeRealizationType, error) {
	if voided {
		return api.BillingChargeRealizationTypeVoided, nil
	}

	switch t {
	case usagebased.RealizationRunTypeFinalRealization:
		return api.BillingChargeRealizationTypeFinalRealization, nil
	case usagebased.RealizationRunTypePartialInvoice:
		return api.BillingChargeRealizationTypePartialInvoice, nil
	default:
		return "", fmt.Errorf("unsupported usage based realization run type: %s", t)
	}
}

func convertRealizationInvoiceToAPI(invoiceID *string, expanded *billing.StandardInvoice) (*api.ChargeRealizationInvoiceOrReference, error) {
	if invoiceID == nil {
		return nil, nil
	}

	if expanded != nil {
		apiInvoice, err := billinginvoices.ToAPIChargeRealizationInvoice(*expanded)
		if err != nil {
			return nil, fmt.Errorf("converting invoice: %w", err)
		}

		var out api.ChargeRealizationInvoiceOrReference
		if err := out.FromBillingChargeRealizationInvoice(apiInvoice); err != nil {
			return nil, fmt.Errorf("setting invoice union: %w", err)
		}

		return &out, nil
	}

	return convertInvoiceIDToReference(invoiceID)
}

func convertInvoiceIDToReference(invoiceID *string) (*api.ChargeRealizationInvoiceOrReference, error) {
	if invoiceID == nil {
		return nil, nil
	}

	var invoice api.ChargeRealizationInvoiceOrReference
	if err := invoice.FromChargeRealizationInvoiceReference(api.ChargeRealizationInvoiceReference{Id: *invoiceID}); err != nil {
		return nil, fmt.Errorf("converting invoice reference: %w", err)
	}

	return &invoice, nil
}

func convertRealizationPaymentToAPI(p *payment.Invoiced) *api.BillingChargeRealizationPayment {
	if p == nil {
		return nil
	}

	return &api.BillingChargeRealizationPayment{
		Status: api.BillingChargeRealizationPaymentStatus(p.Status),
	}
}

func convertFlatFeeDetailedLinesToAPI(source mo.Option[flatfee.DetailedLines]) (*[]api.BillingChargeRealizationDetailedLine, error) {
	if !source.IsPresent() {
		return nil, nil
	}

	lines := make([]api.BillingChargeRealizationDetailedLine, 0, len(source.OrEmpty()))
	for _, line := range source.OrEmpty() {
		var out api.BillingChargeRealizationDetailedLine
		if err := out.FromBillingChargeRealizationDetailedLineFlatFee(api.BillingChargeRealizationDetailedLineFlatFee{
			AmountDiscounts: convertRealizationAmountDiscountsToAPI(line.AmountDiscounts),
			Category:        api.BillingChargeRealizationDetailedLineCategory(line.Category),
			CreatedAt:       line.CreatedAt,
			CreditsApplied:  convertRealizationCreditsAppliedToAPI(line.CreditsApplied),
			DeletedAt:       line.DeletedAt,
			Id:              line.ID,
			ServicePeriod:   ConvertClosedPeriodToAPI(line.ServicePeriod),
			Totals:          ToAPIBillingTotals(line.Totals),
			Type:            api.BillingChargeRealizationDetailedLineFlatFeeTypeFlatFee,
			UnitPrice:       line.PerUnitAmount.String(),
			UpdatedAt:       line.UpdatedAt,
		}); err != nil {
			return nil, fmt.Errorf("setting flat fee detailed line union: %w", err)
		}

		lines = append(lines, out)
	}

	return &lines, nil
}

func convertUsageBasedDetailedLinesToAPI(source mo.Option[usagebased.DetailedLines]) (*[]api.BillingChargeRealizationDetailedLine, error) {
	if !source.IsPresent() {
		return nil, nil
	}

	lines := make([]api.BillingChargeRealizationDetailedLine, 0, len(source.OrEmpty()))
	for _, line := range source.OrEmpty() {
		var out api.BillingChargeRealizationDetailedLine
		if err := out.FromBillingChargeRealizationDetailedLineUsageBased(api.BillingChargeRealizationDetailedLineUsageBased{
			AmountDiscounts: convertRealizationAmountDiscountsToAPI(line.AmountDiscounts),
			Category:        api.BillingChargeRealizationDetailedLineCategory(line.Category),
			CorrectsRunId:   line.CorrectsRunID,
			CreatedAt:       line.CreatedAt,
			CreditsApplied:  convertRealizationCreditsAppliedToAPI(line.CreditsApplied),
			DeletedAt:       line.DeletedAt,
			Id:              line.ID,
			Quantity:        line.Quantity.String(),
			ServicePeriod:   ConvertClosedPeriodToAPI(line.ServicePeriod),
			Totals:          ToAPIBillingTotals(line.Totals),
			Type:            api.BillingChargeRealizationDetailedLineUsageBasedTypeUsageBased,
			UnitPrice:       line.PerUnitAmount.String(),
			UpdatedAt:       line.UpdatedAt,
		}); err != nil {
			return nil, fmt.Errorf("setting usage based detailed line union: %w", err)
		}

		lines = append(lines, out)
	}

	return &lines, nil
}

func convertRealizationCreditsAppliedToAPI(source creditsapplied.CreditsApplied) *[]api.BillingChargeRealizationDetailedLineCreditApplied {
	if len(source) == 0 {
		return nil
	}

	credits := lo.Map(source, func(credit creditsapplied.CreditApplied, _ int) api.BillingChargeRealizationDetailedLineCreditApplied {
		out := api.BillingChargeRealizationDetailedLineCreditApplied{
			Amount:              credit.Amount.String(),
			CreditRealizationId: credit.CreditRealizationID,
		}

		if credit.Description != "" {
			out.Description = lo.ToPtr(credit.Description)
		}

		return out
	})

	return &credits
}

// convertRealizationAmountDiscountsToAPI maps the signed discount snapshots of a
// detailed line. Negative amounts reverse a previously realized discount.
func convertRealizationAmountDiscountsToAPI(source chargedetailedline.AmountDiscounts) []api.BillingChargeRealizationAmountDiscount {
	discounts := make([]api.BillingChargeRealizationAmountDiscount, 0, len(source))

	for _, discount := range source {
		out := api.BillingChargeRealizationAmountDiscount{
			Amount:                 discount.Amount.String(),
			ChildUniqueReferenceId: discount.ChildUniqueReferenceID,
			Description:            discount.Description,
			Reason:                 string(discount.Reason.Type()),
		}

		if !discount.RoundingAmount.IsZero() {
			out.RoundingAmount = lo.ToPtr(discount.RoundingAmount.String())
		}

		discounts = append(discounts, out)
	}

	return discounts
}

// convertChargeToAPI dispatches on charge type and maps to the API union
// type. Expanded entities are carried by the CustomerCharge; expands only
// gates the emission of realization totals.
func convertChargeToAPI(charge billingcharges.CustomerCharge, expands meta.Expands) (api.BillingCharge, error) {
	var out api.BillingCharge

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		apiFF, err := convertFlatFeeChargeToAPI(charge, expands)
		if err != nil {
			return out, err
		}
		if err := out.FromBillingChargeFlatFee(apiFF); err != nil {
			return out, fmt.Errorf("setting flat fee charge union: %w", err)
		}

	case meta.ChargeTypeUsageBased:
		apiUB, err := convertUsageBasedChargeToAPI(charge, expands)
		if err != nil {
			return out, err
		}
		if err := out.FromBillingChargeUsageBased(apiUB); err != nil {
			return out, fmt.Errorf("setting usage based charge union: %w", err)
		}

	case meta.ChargeTypeCreditPurchase:
		// Credit purchases are excluded at the query level (ChargeTypes filter) and
		// should never reach this path. Return an error as a defensive measure.
		return out, fmt.Errorf("credit purchase charges are not supported in the charges API")

	default:
		return out, fmt.Errorf("unsupported charge type: %s", charge.Type())
	}

	return out, nil
}

// convertUsageBasedChargeTotals aggregates booked totals from persisted realization runs.
func convertUsageBasedChargeTotals(charge usagebased.Charge) api.BillingChargeTotals {
	out := api.BillingChargeTotals{
		Booked: ToAPIBillingTotals(charge.Realizations.Sum()),
	}

	if charge.Expands.RealtimeUsage != nil {
		out.Realtime = lo.ToPtr(ToAPIBillingTotals(*charge.Expands.RealtimeUsage))
	}

	return out
}

// ToAPIBillingTotals maps a domain totals.Totals to the API BillingTotals type.
// Shared with the invoice handlers through billingcommon; the alias keeps the
// exported surface and in-package call sites unchanged.
var ToAPIBillingTotals = billingcommon.ToAPIBillingTotals

type apiUsageBasedRatingConfiguration struct {
	Price       api.BillingPriceUsageBased
	Commitments *api.BillingSpendCommitments
	UnitConfig  *api.BillingUnitConfig
}

// toAPIBillingUsageBasedRatingConfiguration maps the complete rating snapshot.
// Stored unit config takes precedence; legacy dynamic and package prices
// synthesize the same v3 unit-price representation used by plans and
// subscriptions. Commitments remain attached to the projected price semantics.
func toAPIBillingUsageBasedRatingConfiguration(intent usagebased.IntentMutableFields) (apiUsageBasedRatingConfiguration, error) {
	price, err := plans.ToAPIBillingPriceUsageBased(&intent.Price)
	if err != nil {
		return apiUsageBasedRatingConfiguration{}, fmt.Errorf("converting price: %w", err)
	}

	result := apiUsageBasedRatingConfiguration{
		Price:       price,
		Commitments: plans.ToAPIBillingSpendCommitments(intent.Price.GetCommitments()),
	}

	if intent.UnitConfig != nil {
		result.UnitConfig = lo.ToPtr(plans.ToAPIBillingUnitConfig(*intent.UnitConfig))

		return result, nil
	}

	legacyUnitConfig, err := plans.ToAPIBillingUnitConfigFromLegacyPrice(&intent.Price)
	if err != nil {
		return apiUsageBasedRatingConfiguration{}, fmt.Errorf("converting unit config: %w", err)
	}
	result.UnitConfig = legacyUnitConfig

	return result, nil
}

// convertFlatFeeDiscounts maps the optional percentage discount to the anonymous API struct.
func convertFlatFeeDiscounts(pd *billing.PercentageDiscount) *api.BillingChargeFlatFeeDiscounts {
	if pd == nil {
		return nil
	}
	pct := float32(pd.Percentage.InexactFloat64())
	return &api.BillingChargeFlatFeeDiscounts{Percentage: &pct}
}

// convertUsageBasedDiscounts maps usage-based discounts to the API type.
func convertUsageBasedDiscounts(d billing.Discounts) *api.BillingRateCardDiscounts {
	if d.Percentage == nil && d.Usage == nil {
		return nil
	}
	result := &api.BillingRateCardDiscounts{}
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

// ConvertUsageBasedStatusToAPI maps usage-based substates to their top-level API status.
// For example, "active.final_realization.started" maps to "active".
func ConvertUsageBasedStatusToAPI(status usagebased.Status) (*api.BillingChargeStatus, error) {
	s, err := status.ToMetaChargeStatus()
	if err != nil {
		return nil, fmt.Errorf("converting usage-based status to charge status: %w", err)
	}
	return lo.ToPtr(api.BillingChargeStatus(s)), nil
}

// ConvertClosedPeriodToAPI maps a domain ClosedPeriod to the API type.
var ConvertClosedPeriodToAPI = billingcommon.ConvertClosedPeriodToAPI

// ConvertDecimalToCurrencyAmount wraps a decimal amount and currency in a CurrencyAmount.
func ConvertDecimalToCurrencyAmount(d alpacadecimal.Decimal, currency currencyx.Code) api.ChargesCurrencyAmount {
	return api.ChargesCurrencyAmount{Amount: d.String(), Currency: api.BillingCurrencyCode(currency)}
}

func convertCustomerIDToReference(id string) (api.CustomerOrReference, error) {
	var customer api.CustomerOrReference
	if err := customer.FromCustomerReference(api.CustomerReference{Id: id}); err != nil {
		return customer, fmt.Errorf("converting customer reference: %w", err)
	}

	return customer, nil
}

// ConvertProRatingConfigToAPI maps a ProRatingConfig to the API proration configuration.
func ConvertProRatingConfigToAPI(c productcatalog.ProRatingConfig) api.BillingRateCardProrationConfiguration {
	if !c.Enabled {
		return api.BillingRateCardProrationConfiguration{
			Mode: api.BillingRateCardProrationModeNoProration,
		}
	}
	return api.BillingRateCardProrationConfiguration{
		Mode: api.BillingRateCardProrationMode(c.Mode),
	}
}

// ConvertSubscriptionRefToAPI maps a SubscriptionReference to the API type.
var ConvertSubscriptionRefToAPI = billingcommon.ConvertSubscriptionRefToAPI

// LifecycleControllerOption configures lifecycle controller conversion.
type LifecycleControllerOption = billingcommon.LifecycleControllerOption

// WithManualOverride marks the API lifecycle controller manual when a charge
// override exists even if the base intent remains subscription-owned for sync.
var WithManualOverride = billingcommon.WithManualOverride

// ConvertLifecycleControllerToAPI maps the internal lifecycle owner to the public
// lifecycle controller.
var ConvertLifecycleControllerToAPI = billingcommon.ConvertLifecycleControllerToAPI

// convertTaxCodeConfigToAPI maps a TaxCodeConfig (Behavior + TaxCodeID) to the API type.
func convertTaxCodeConfigToAPI(cfg productcatalog.TaxCodeConfig) *api.BillingTaxConfig {
	if lo.IsEmpty(cfg) {
		return nil
	}

	out := &api.BillingTaxConfig{
		Behavior: (*api.BillingTaxBehavior)(cfg.Behavior),
	}

	if cfg.TaxCodeID != "" {
		out.TaxCode = &api.TaxCodeReference{Id: cfg.TaxCodeID}
		out.TaxCodeId = lo.ToPtr(cfg.TaxCodeID)
	}

	return out
}

// convertAPIChargesExpand maps an API expand token to its service-side
// equivalent. The API's `realization.detailed_lines` maps onto the existing
// detailed-lines expand, which the adapters resolve while loading runs.
func convertAPIChargesExpand(e api.BillingChargesExpand) (meta.Expand, error) {
	switch e {
	case api.BillingChargesExpandRealTimeUsage:
		return meta.ExpandRealtimeUsage, nil
	case api.BillingChargesExpandCustomer:
		return meta.ExpandCustomer, nil
	case api.BillingChargesExpandFeature:
		return meta.ExpandFeature, nil
	case api.BillingChargesExpandSubscription:
		return meta.ExpandSubscription, nil
	case api.BillingChargesExpandRealizationInvoice:
		return meta.ExpandRealizationInvoice, nil
	case api.BillingChargesExpandRealizationTotals:
		return meta.ExpandRealizationTotals, nil
	case api.BillingChargesExpandRealizationDetailedLines:
		return meta.ExpandDetailedLines, nil
	default:
		return "", fmt.Errorf("unsupported expand: %s", e)
	}
}

func fromAPICreateChargeFlatFeeRequest(namespace, customerID string, flatFee api.CreateChargeFlatFeeRequest) (billingcharges.CreateCustomerChargeInput, error) {
	var zero billingcharges.CreateCustomerChargeInput

	taxConfig, err := billingprofiles.FromAPIBillingTaxConfig(flatFee.TaxConfig)
	if err != nil {
		return zero, fmt.Errorf("invalid tax config: %w", err)
	}

	if flatFee.AmountBeforeProration.Currency != flatFee.Currency {
		return zero, models.NewGenericValidationError(errors.New("currency mismatch, amount_before_proration.currency should match currency field"))
	}

	amountBeforeProration, err := alpacadecimal.NewFromString(flatFee.AmountBeforeProration.Amount)
	if err != nil {
		return zero, models.NewGenericValidationError(errors.New("invalid amount_before_proration.amount, must be a valid decimal string"))
	}

	var metadata models.Metadata
	if flatFee.Labels != nil {
		metadata = models.Metadata(*flatFee.Labels)
	}

	var discount *billing.PercentageDiscount
	if flatFee.Discounts != nil && flatFee.Discounts.Percentage != nil {
		discount = billing.Discounts{
			Percentage: &billing.PercentageDiscount{
				PercentageDiscount: productcatalog.PercentageDiscount{
					Percentage: models.NewPercentage(float64(lo.FromPtr(flatFee.Discounts.Percentage))),
				},
			},
		}.UpsertCorrelationIDs().Percentage
	}

	var proRating productcatalog.ProRatingConfig
	if flatFee.ProrationConfiguration.Mode == api.BillingRateCardProrationModeProratePrices {
		proRating = productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	} else {
		proRating = productcatalog.ProRatingConfig{
			Enabled: false,
		}
	}

	var featureID *string
	if flatFee.Feature != nil {
		featureID = lo.ToPtr(flatFee.Feature.Id)
	}

	var costBasis *costbasis.Intent
	if flatFee.CostBasis != nil {
		costBasisType, err := flatFee.CostBasis.Discriminator()
		if err != nil {
			return zero, fmt.Errorf("invalid cost basis: %w", err)
		}

		var currencyCode api.CurrencyCode
		var intentBuilder costbasis.NewIntentFromFieldsInput

		switch costBasisType {
		// TODO fix poor naming
		case string(api.BillingChargeCostBasisDynamicTypeDynamic):
			dynamic, err := flatFee.CostBasis.AsBillingChargeCostBasisDynamic()
			if err != nil {
				return zero, fmt.Errorf("invalid fiat currency: %w", err)
			}
			intentBuilder.Mode = costbasis.ModeDynamic
			currencyCode = dynamic.FiatCurrency
		case string(api.BillingChargeCostBasisManualTypeManual):
			manual, err := flatFee.CostBasis.AsBillingChargeCostBasisManual()
			if err != nil {
				return zero, fmt.Errorf("invalid fiat currency: %w", err)
			}

			intentBuilder.Mode = costbasis.ModeManual
			currencyCode = manual.FiatCurrency

			rate, err := alpacadecimal.NewFromString(manual.Rate)
			if err != nil {
				return zero, fmt.Errorf("invalid rate: %w", err)
			}
			intentBuilder.Rate = &rate
		case string(api.BillingChargeCostBasisPinnedTypePinned):
			pinned, err := flatFee.CostBasis.AsBillingChargeCostBasisPinned()
			if err != nil {
				return zero, fmt.Errorf("invalid fiat currency: %w", err)
			}
			intentBuilder.Mode = costbasis.ModePinned
			intentBuilder.CurrencyCostBasisID = &pinned.CostBasisId
			currencyCode = pinned.FiatCurrency
		}

		intentBuilder.FiatCurrency, err = currencyx.NewFiatCurrency(currencyCode)
		if err != nil {
			return zero, fmt.Errorf("invalid fiat currency: %w", err)
		}

		intent, err := costbasis.NewIntentFromFields(intentBuilder)
		if err != nil {
			return zero, fmt.Errorf("invalid cost basis: %w", err)
		}
		costBasis = &intent
	}

	return billingcharges.CreateCustomerChargeInput{
		Namespace:         namespace,
		CustomerID:        customerID,
		CurrencyCode:      currencyx.Code(flatFee.Currency),
		CostBasis:         costBasis,
		TaxConfig:         productcatalog.TaxCodeConfigFrom(taxConfig),
		UniqueReferenceID: flatFee.UniqueReferenceId,
		FlatFee: &billingcharges.CreateCustomerChargeFlatFeeInput{
			IntentMutableFields: flatfee.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              flatFee.Name,
					Description:       flatFee.Description,
					Metadata:          metadata,
					ServicePeriod:     timeutil.ClosedPeriod(flatFee.ServicePeriod),
					FullServicePeriod: timeutil.ClosedPeriod(lo.FromPtrOr(flatFee.FullServicePeriod, flatFee.ServicePeriod)),
					BillingPeriod:     timeutil.ClosedPeriod(lo.FromPtrOr(flatFee.BillingPeriod, flatFee.ServicePeriod)),
				},
				InvoiceAt:             flatFee.InvoiceAt,
				PaymentTerm:           productcatalog.PaymentTermType(flatFee.PaymentTerm),
				PercentageDiscounts:   discount,
				ProRating:             proRating,
				AmountBeforeProration: amountBeforeProration,
			},
			FeatureID:      featureID,
			SettlementMode: productcatalog.SettlementMode(flatFee.SettlementMode),
		},
	}, nil
}

func fromAPICreateChargeUsageBasedRequest(namespace, customerID string, usageBasedFee api.CreateChargeUsageBasedRequest) (billingcharges.CreateCustomerChargeInput, error) {
	var zero billingcharges.CreateCustomerChargeInput

	taxConfig, err := billingprofiles.FromAPIBillingTaxConfig(usageBasedFee.TaxConfig)
	if err != nil {
		return zero, fmt.Errorf("invalid tax config: %w", err)
	}

	var metadata models.Metadata
	if usageBasedFee.Labels != nil {
		metadata = models.Metadata(*usageBasedFee.Labels)
	}

	var discounts billing.Discounts
	if usageBasedFee.Discounts != nil {
		if usageBasedFee.Discounts.Percentage != nil {
			discounts.Percentage = &billing.PercentageDiscount{
				PercentageDiscount: productcatalog.PercentageDiscount{
					Percentage: models.NewPercentage(float64(lo.FromPtr(usageBasedFee.Discounts.Percentage))),
				},
			}
		}
		if usageBasedFee.Discounts.Usage != nil {
			quantity, err := alpacadecimal.NewFromString(lo.FromPtr(usageBasedFee.Discounts.Usage))
			if err != nil {
				return zero, fmt.Errorf("invalid usage discount quantity: %w", err)
			}
			discounts.Usage = &billing.UsageDiscount{
				UsageDiscount: productcatalog.UsageDiscount{
					Quantity: quantity,
				},
			}
		}
		discounts = discounts.UpsertCorrelationIDs()
	}

	price, err := plans.FromAPIBillingPriceUsageBased(usageBasedFee.Price, usageBasedFee.Commitments)
	if err != nil {
		return zero, fmt.Errorf("invalid price: %w", err)
	}

	var unitConfig *productcatalog.UnitConfig
	if usageBasedFee.UnitConfig != nil {
		unitConfig, err = plans.FromAPIBillingUnitConfig(*usageBasedFee.UnitConfig)
		if err != nil {
			return zero, fmt.Errorf("invalid unit config: %w", err)
		}
	}

	return billingcharges.CreateCustomerChargeInput{
		Namespace:         namespace,
		CustomerID:        customerID,
		CurrencyCode:      currencyx.Code(usageBasedFee.Currency),
		TaxConfig:         productcatalog.TaxCodeConfigFrom(taxConfig),
		UniqueReferenceID: usageBasedFee.UniqueReferenceId,
		UsageBased: &billingcharges.CreateCustomerChargeUsageBasedInput{
			IntentMutableFields: usagebased.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              usageBasedFee.Name,
					Description:       usageBasedFee.Description,
					Metadata:          metadata,
					ServicePeriod:     timeutil.ClosedPeriod(usageBasedFee.ServicePeriod),
					FullServicePeriod: timeutil.ClosedPeriod(lo.FromPtrOr(usageBasedFee.FullServicePeriod, usageBasedFee.ServicePeriod)),
					BillingPeriod:     timeutil.ClosedPeriod(lo.FromPtrOr(usageBasedFee.BillingPeriod, usageBasedFee.ServicePeriod)),
				},
				InvoiceAt:  usageBasedFee.InvoiceAt,
				Price:      *price,
				Discounts:  discounts,
				UnitConfig: unitConfig,
			},
			FeatureID:      usageBasedFee.Feature.Id,
			SettlementMode: productcatalog.SettlementMode(usageBasedFee.SettlementMode),
		},
	}, nil
}
