package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	openmeter "github.com/openmeterio/openmeter/api"
	stripe "github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/usagerecord"
	"github.com/stripe/stripe-go/v74/webhook"
)

type Server struct {
	endpointSecret string
	openmeter      *openmeter.Client
}

// See: https://stripe.com/docs/webhooks/quickstart
func (s *Server) handleWebhook(w http.ResponseWriter, req *http.Request) {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}
	err = json.Unmarshal(payload, &event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Webhook error while parsing basic request. %v\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signatureHeader := req.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, signatureHeader, s.endpointSecret)
	if err != nil {
		http.Error(w, fmt.Sprintf("Webhook signature verification failed. %v", err), http.StatusBadRequest)
		return
	}
	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "customer.subscription.updated":
		var subscription stripe.Subscription
		err = json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = s.reportUsage(req.Context(), &subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error usage report: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) reportUsage(ctx context.Context, subscription *stripe.Subscription) error {
	for _, item := range subscription.Items.Data {
		// Skip subscription item if it's not metered
		if item.Price.Recurring.UsageType != "metered" {
			return nil
		}

		meterId := item.Metadata["om_meter_id"]
		// Skip subscription item if doesn't have corresponding OpenMeter meter
		if meterId == "" {
			return nil
		}

		// We round down period to closest windows as OpenMeter aggregates usage in windows.
		// Usage occuring between rounded down CurrentPeriodEnd and CurrentPeriodEnd will be attributed to the next billing period.
		periodStart := time.Unix(subscription.CurrentPeriodStart, 0).Truncate(time.Hour)
		periodEnd := time.Unix(subscription.CurrentPeriodEnd, 0).Truncate(time.Hour)

		// Query usage from OpenMeter for billing period
		resp, err := s.openmeter.GetValuesByMeterId(ctx, meterId, &openmeter.GetValuesByMeterIdParams{
			// If you don't report usage events via Stripe Customer ID you need to map Stripe IDs to your internal ID
			Subject: &subscription.Customer.ID,
			From:    &periodStart,
			To:      &periodEnd,
		})
		if err != nil {
			return err
		}
		payload, err := openmeter.ParseGetValuesByMeterIdResponse(resp)
		if err != nil {
			return err
		}

		// TODO (pmarton): switch to OpenMeter aggregate API
		var total float32 = 0
		for _, value := range *payload.JSON200.Values {
			total += *value.Value
		}

		// Debug log
		fmt.Printf(
			"stripe_customer: %s, stripe_price: %s, meter: %s, total_usage: %f, from: %s, to: %s",
			subscription.Customer.ID,
			item.Price.ID,
			meterId,
			total,
			periodStart.String(),
			periodEnd.String(),
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
