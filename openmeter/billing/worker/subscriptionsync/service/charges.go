package service

import (
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func (s *Service) chargeFromSubscriptionRateCard(subs subscription.SubscriptionView, item subscriptionItemWithPeriods, currency currencyx.Calculator) (*charges.Charge, error) {
	switch item.Spec.RateCard.AsMeta().Price.Type() {
	case productcatalog.FlatPriceType:
		return s.chargeFromFlatPrice(subs, item, currency)
	default:
		return s.chargeFromUsageBasedPrice(subs, item, currency)
	}
}

func (s *Service) chargeFromUsageBasedPrice(subs subscription.SubscriptionView, item subscriptionItemWithPeriods, currency currencyx.Calculator) (*charges.Charge, error) {
	gatheringLine, err := s.lineFromSubscriptionRateCard(subs, item, currency)
	if err != nil {
		return nil, err
	}

	if gatheringLine == nil {
		return nil, nil
	}

	intent := charges.NewIntent(
		charges.IntentMeta{
			Metadata:          gatheringLine.Metadata,
			Annotations:       gatheringLine.Annotations,
			ManagedBy:         gatheringLine.ManagedBy,
			CustomerID:        subs.Customer.ID,
			Currency:          subs.Spec.Currency,
			ServicePeriod:     gatheringLine.ServicePeriod,
			FullServicePeriod: item.FullServicePeriod.ToClosedPeriod(),
			BillingPeriod:     item.BillingPeriod.ToClosedPeriod(),
			InvoiceAt:         gatheringLine.InvoiceAt,
			TaxConfig:         gatheringLine.TaxConfig,
			UniqueReferenceID: gatheringLine.ChildUniqueReferenceID,
			Subscription: &charges.SubscriptionReference{
				SubscriptionID: subs.Subscription.ID,
				PhaseID:        item.PhaseID,
				ItemID:         item.SubscriptionItem.ID,
			},
		},
		charges.UsageBasedIntent{
			Price:      gatheringLine.Price,
			Discounts:  lo.EmptyableToPtr(item.Spec.RateCard.AsMeta().Discounts),
			FeatureKey: gatheringLine.FeatureKey,
		},
	)

	charge := &charges.Charge{
		ManagedResource: gatheringLine.ManagedResource,
		Intent:          intent,
	}

	if err := charge.Validate(); err != nil {
		return nil, err
	}

	return charge, nil
}

func (s *Service) chargeFromFlatPrice(subs subscription.SubscriptionView, item subscriptionItemWithPeriods, currency currencyx.Calculator) (*charges.Charge, error) {
	gatheringLine, err := s.lineFromSubscriptionRateCard(subs, item, currency)
	if err != nil {
		return nil, err
	}

	if gatheringLine == nil {
		return nil, nil
	}

	itemMeta := item.Spec.RateCard.AsMeta()

	price, err := itemMeta.Price.AsFlat()
	if err != nil {
		return nil, err
	}

	paymentTerm := price.PaymentTerm
	if paymentTerm == "" {
		paymentTerm = productcatalog.DefaultPaymentTerm
	}

	var percentageDiscounts *productcatalog.PercentageDiscount
	if itemMeta.Discounts.Percentage != nil {
		percentageDiscounts = lo.ToPtr(itemMeta.Discounts.Percentage.Clone())
	}

	prorationConfig := productcatalog.ProRatingConfig{
		Enabled: false,
	}

	if !item.ServicePeriod.IsEmpty() && s.shouldProrate(item, subs) {
		prorationConfig = productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}

	linePrice, err := gatheringLine.Price.AsFlat()
	if err != nil {
		return nil, err
	}

	intent := charges.NewIntent(
		charges.IntentMeta{
			Metadata:          gatheringLine.Metadata,
			Annotations:       gatheringLine.Annotations,
			ManagedBy:         gatheringLine.ManagedBy,
			CustomerID:        subs.Customer.ID,
			Currency:          subs.Spec.Currency,
			ServicePeriod:     gatheringLine.ServicePeriod,
			FullServicePeriod: item.FullServicePeriod.ToClosedPeriod(),
			BillingPeriod:     item.BillingPeriod.ToClosedPeriod(),
			InvoiceAt:         gatheringLine.InvoiceAt,
			TaxConfig:         gatheringLine.TaxConfig,
			UniqueReferenceID: gatheringLine.ChildUniqueReferenceID,
			Subscription: &charges.SubscriptionReference{
				SubscriptionID: subs.Subscription.ID,
				PhaseID:        item.PhaseID,
				ItemID:         item.SubscriptionItem.ID,
			},
		},
		charges.FlatFeeIntent{
			AmountBeforeProration: price.Amount,
			AmountAfterProration:  linePrice.Amount,
			PaymentTerm:           paymentTerm,
			PercentageDiscounts:   percentageDiscounts,
			ProRating:             prorationConfig,
			FeatureKey:            gatheringLine.FeatureKey,
		},
	)

	charge := &charges.Charge{
		ManagedResource: gatheringLine.ManagedResource,
		Intent:          intent,
	}

	if err := intent.Validate(); err != nil {
		return nil, err
	}

	return charge, nil
}

func upsertWithRealizations(upsert upsertChargeAndAssociateLinesPatch, linesByLineID map[string]billing.LineWithInvoiceHeader) (upsertChargeAndAssociateLinesPatch, error) {
	for _, lineID := range upsert.LinesIDsToAssociate {
		line, ok := linesByLineID[lineID.ID]
		if !ok {
			return upsert, fmt.Errorf("line %s not found", lineID.ID)
		}

		// Gathering invoice lines are not realizations of the charge intent so we skip them
		if line.Invoice.AsInvoice().Type() != billing.InvoiceTypeStandard {
			continue
		}

		stdInvoice, err := line.Invoice.AsInvoice().AsStandardInvoice()
		if err != nil {
			return upsert, fmt.Errorf("converting invoice to standard invoice: %w", err)
		}

		stdLine, err := line.Line.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return upsert, fmt.Errorf("converting line to standard line: %w", err)
		}

		meteredServicePeriodQuantity := lo.FromPtr(lo.CoalesceOrEmpty(stdLine.UsageBased.MeteredQuantity, stdLine.UsageBased.Quantity))
		meteredPreServicePeriodQuantity := lo.FromPtr(lo.CoalesceOrEmpty(stdLine.UsageBased.MeteredPreLinePeriodQuantity, stdLine.UsageBased.PreLinePeriodQuantity))

		status := charges.StandardInvoiceRealizationStatusAuthorized
		if stdInvoice.Status.IsFinal() {
			status = charges.StandardInvoiceRealizationStatusSettled
		} else {
			shortStatus := stdInvoice.Status.ShortStatus()
			if slices.Contains(billing.StandardInvoiceMutableStatusCategories, billing.StandardInvoiceStatusCategory(shortStatus)) {
				status = charges.StandardInvoiceRealizationStatusDraft
			}
		}

		upsert.Charge.Realizations.StandardInvoice = append(upsert.Charge.Realizations.StandardInvoice, charges.StandardInvoiceRealization{
			LineID:                          stdLine.ID,
			ServicePeriod:                   stdLine.Period.ToClosedPeriod(),
			Status:                          status,
			Totals:                          stdLine.Totals,
			MeteredServicePeriodQuantity:    meteredServicePeriodQuantity,
			MeteredPreServicePeriodQuantity: meteredPreServicePeriodQuantity,
		})

	}

	return upsert, nil
}
