package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

func (s *service) TriggerPeriodicRealization(ctx context.Context, input charges.ChargeID) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		// TODO: at this point we should rather just fetch the header to validate if it should even receive the
		// realization event.
		charge, err := s.adapter.GetChargeByID(ctx, input)
		if err != nil {
			return err
		}

		if charge.Status != charges.ChargeStatusActive {
			// TODO: charge lifecycle management (e.g change state when fully settled etc.)
			return fmt.Errorf("charge is not active: %s", charge.Status)
		}

		if charge.Intent.SettlementMode == productcatalog.InvoiceOnlySettlementMode {
			return fmt.Errorf("Periodic realization is only supported for credit based settlement modes [chargeId=%s]", charge.ID)
		}

		if charge.Intent.IntentType != charges.IntentTypeUsageBased {
			return fmt.Errorf("Periodic realization is only supported for usage based charges [chargeId=%s intentType=%s]", charge.ID, charge.Intent.IntentType)
		}

		// Given realization is a kind of progressive billing, we need to make sure that we are supporting progressive
		// billing for the charge (or we only realize gains at the end of the service period)

		asOf := clock.Now().In(time.UTC).Truncate(24 * time.Hour)

		// If we already have a realization for this period we skip the realization
		for _, realization := range charge.Realizations.Credit {
			// TODO: maybe contains?! it doesn't matter too much but still
			if realization.ServicePeriod.To.Equal(asOf) {
				return nil
			}
		}

		priceAccessor, err := newChargePriceAccessor(charge)
		if err != nil {
			return err
		}

		featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, charge.Namespace, []string{priceAccessor.GetFeatureKey()})
		if err != nil {
			return err
		}

		chargeBillablePeriod, err := lineservice.ResolveBillablePeriod[chargePriceAccessor](
			lineservice.ResolveBillablePeriodInput[chargePriceAccessor]{
				AsOf:               asOf,
				ProgressiveBilling: true,
				Line:               priceAccessor,
				FeatureMeters:      featureMeters,
			})
		if err != nil {
			return err
		}

		if chargeBillablePeriod == nil {
			// TODO: log a warning that the charge is not billable
			return nil
		}

		// Let's calculate the total value of the UBP line for the service period
		// NOTE: this means that we are already considering future events, but it's not a
		// big deal, as we are going to bill those during finalization either ways.
		// TODO: let's think a bit more about this

		gatheringLine, err := chargeIntentToGatheringLine(charge)
		if err != nil {
			return err
		}

		standardLine := gatheringLine.AsStandardLine()
		if err != nil {
			return err
		}

		// TODO: given we are calculating in memory it's fine, but let's doublecheck if we even care about the
		// invoice id in the line service.
		standardLine.InvoiceID = charge.ID

		// TODO: we should filter by stored_at < asOf
		quantity, err := s.getUsageBasedChargeQuantity(ctx, charge, featureMeters)
		if err != nil {
			return err
		}

		standardLine.UsageBased.Quantity = &quantity
		standardLine.UsageBased.MeteredQuantity = &quantity
		standardLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
		standardLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)

		lineSvc, err := lineservice.FromEntity(&standardLine, featureMeters)
		if err != nil {
			return err
		}

		err = lineSvc.CalculateDetailedLines()
		if err != nil {
			return err
		}

		err = lineSvc.UpdateTotals()
		if err != nil {
			return err
		}

		creditRealizations, err := s.handler.OnRealizeUsageBasedCreditChargePeriodically(ctx, charges.UsageBasedRealizationInput{
			Charge:       charge,
			AsOf:         asOf,
			CurrentUsage: lineSvc.ToEntity(),
		})
		if err != nil {
			return err
		}

		return s.createCreditRealization(ctx, charge, creditRealizations)
	})
}

func newChargePriceAccessor(charge charges.Charge) (chargePriceAccessor, error) {
	switch charge.Intent.IntentType {
	case charges.IntentTypeFlatFee:
		flat, err := charge.Intent.GetFlatFeeIntent()
		if err != nil {
			return chargePriceAccessor{}, err
		}
		return chargePriceAccessor{
			featureKey:    flat.FeatureKey,
			servicePeriod: charge.Intent.ServicePeriod,
			price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      flat.AmountAfterProration,
				PaymentTerm: flat.PaymentTerm,
			}),
			invoiceAt: charge.Intent.InvoiceAt,
			id:        charge.ID,
		}, nil
	case charges.IntentTypeUsageBased:
		usage, err := charge.Intent.GetUsageBasedIntent()
		if err != nil {
			return chargePriceAccessor{}, err
		}
		return chargePriceAccessor{
			featureKey:    usage.FeatureKey,
			servicePeriod: charge.Intent.ServicePeriod,
			price:         &usage.Price,
			invoiceAt:     charge.Intent.InvoiceAt,
			id:            charge.ID,
		}, nil
	default:
		return chargePriceAccessor{}, fmt.Errorf("invalid intent type: %s", charge.Intent.IntentType)
	}
}

type chargePriceAccessor struct {
	id            string
	featureKey    string
	servicePeriod timeutil.ClosedPeriod
	price         *productcatalog.Price
	invoiceAt     time.Time
}

func (c chargePriceAccessor) GetPrice() *productcatalog.Price {
	return c.price
}

func (c chargePriceAccessor) GetServicePeriod() timeutil.ClosedPeriod {
	return c.servicePeriod
}

func (c chargePriceAccessor) GetFeatureKey() string {
	return c.featureKey
}

func (c chargePriceAccessor) GetID() string {
	return c.id
}

func (c chargePriceAccessor) GetInvoiceAt() time.Time {
	return c.invoiceAt
}

func (c chargePriceAccessor) GetSplitLineGroupID() *string {
	return nil
}
