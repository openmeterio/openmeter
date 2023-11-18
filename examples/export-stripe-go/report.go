package main

import (
	"context"
	"fmt"
	"time"

	openmeter "github.com/openmeterio/openmeter/api/client/go"
	stripe "github.com/stripe/stripe-go/v74"
	subscription "github.com/stripe/stripe-go/v74/subscription"
	usagerecord "github.com/stripe/stripe-go/v74/usagerecord"
)

type Report struct {
	ctx       context.Context
	openmeter *openmeter.Client
	from      time.Time
	to        time.Time
}

func NewReport(ctx context.Context, om *openmeter.Client, duration time.Duration) Report {
	// We round down period to closest windows as OpenMeter aggregates usage in windows.
	// Usage occuring between rounded down date and now will be attributed to the next billing period.
	to := time.Now().Truncate(duration)
	from := to.Add(time.Duration(-1) * duration)

	report := Report{
		ctx:       context.Background(),
		openmeter: om,
		from:      from,
		to:        to,
	}

	return report
}

func (r *Report) Run() error {
	status := "active"
	i := subscription.List(&stripe.SubscriptionListParams{
		Status: &status,
	})
	if i.Err() != nil {
		return i.Err()
	}
	for i.Next() {
		s := i.Subscription()

		// Skip subscriptions that started after `to`.
		if s.CurrentPeriodStart > r.to.Unix() {
			continue
		}

		err := r.reportUsage(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Report) reportUsage(subscription *stripe.Subscription) error {
	for _, item := range subscription.Items.Data {
		// Skip subscription item if it's not metered
		if item.Price.Recurring.UsageType != "metered" {
			return nil
		}

		meterSlug := item.Metadata["om_meter_id"]
		// Skip subscription item if doesn't have corresponding OpenMeter meter
		if meterSlug == "" {
			return nil
		}

		// Query usage from OpenMeter for billing period
		resp, err := r.openmeter.QueryMeter(r.ctx, meterSlug, &openmeter.QueryMeterParams{
			// If you don't report usage events via Stripe Customer ID you need to map Stripe IDs to your internal ID
			Subject: &subscription.Customer.ID,
			From:    &r.from,
			To:      &r.to,
		})
		if err != nil {
			return err
		}
		payload, err := openmeter.ParseQueryMeterResponse(resp)
		if err != nil {
			return err
		}

		// TODO (pmarton): switch to OpenMeter aggregate API
		var total float64 = 0
		for _, value := range payload.JSON200.Data {
			total += value.Value
		}

		// Debug log
		fmt.Printf(
			"stripe_customer: %s, stripe_price: %s, meter: %s, total_usage: %f, from: %s, to: %s\n",
			subscription.Customer.ID,
			item.Price.ID,
			meterSlug,
			total,
			r.from.String(),
			r.to.String(),
		)

		// Report usage to Stripe
		// We use action=set so even if this webhook get called multiple time
		// we still end up with only one usage record for the same period.
		// We report on `CurrentPeriodStart` because it's going to be the
		// same between Webhook calls and we add one as Stripe doesn't allow
		// to report usage on the exact start and end time of the subscription.
		recordParams := &stripe.UsageRecordParams{
			Action:           stripe.String("set"),
			Quantity:         stripe.Int64(int64(total)),
			SubscriptionItem: stripe.String(item.ID),
			Timestamp:        stripe.Int64(subscription.CurrentPeriodStart + 1),
		}
		_, err = usagerecord.New(recordParams)
		if err != nil {
			return err
		}
	}

	return nil
}
