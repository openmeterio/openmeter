package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	openmeter "github.com/openmeterio/openmeter/api"
	stripe "github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/usagerecord"
	"github.com/stripe/stripe-go/v74/webhook"
)

// Stripe Key.
var stripeKey = "sk_test_secret"

// Stripe Webhook Secret
// Replace this endpoint secret with your endpoint's unique secret
// If you are testing with the CLI, find the secret by running 'stripe listen'
// If you are using an endpoint defined with the API or dashboard, look in your webhook settings
// at https://dashboard.stripe.com/webhooks
var endpointSecret = "whsec_test_secret"

// Map Stripe Price(s) to OpenMeter meter(s)
var stripePriceIdToMeterId = map[string]string{
	"price_xxx": "requests",
}

func main() {
	om, err := openmeter.NewClient("http://localhost:8888")
	if err != nil {
		panic(err.Error())
	}

	stripe.Key = stripeKey
	server := Server{
		openmeter: om,
	}

	http.HandleFunc("/webhook", server.handleWebhook)
	addr := "localhost:4242"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type Server struct {
	openmeter *openmeter.Client
}

// See: https://stripe.com/docs/webhooks/quickstart
func (s *Server) handleWebhook(w http.ResponseWriter, req *http.Request) {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := io.ReadAll(req.Body)
	// handle err

	event := stripe.Event{}
	err = json.Unmarshal(payload, &event)
	// handle err

	signatureHeader := req.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		http.Error(w, fmt.Sprintf("Webhook signature verification failed. %v", err), http.StatusBadRequest)
		return
	}
	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "customer.subscription.updated":
		var subscription stripe.Subscription
		err = json.Unmarshal(event.Data.Raw, &subscription)
		// handle err

		s.reportUsage(req.Context(), &subscription)
	default:
		// log unhandled event type
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) reportUsage(ctx context.Context, subscription *stripe.Subscription) {
	for _, item := range subscription.Items.Data {
		// Skip invoice item item if didn't have corresponding OpenMeter meter
		if stripePriceIdToMeterId[item.Price.ID] == "" {
			return
		}
		// Or not metered
		if item.Price.Recurring.UsageType != "metered" {
			return
		}

		// Query total usage from OpenMeter for billing period
		meterId := stripePriceIdToMeterId[item.Price.ID]

		// TODO: finish after filters implemented
		// queryParams := &GetTotalParams{
		// 	Meter: meterId,
		// 	// If you don't report usage events via Stripe Customer ID you need to map Stripe IDs to your internal ID
		// 	Consumer: subscription.Customer.ID,
		// 	From:     time.Unix(subscription.CurrentPeriodStart, 0),
		// 	To:       time.Unix(subscription.CurrentPeriodEnd, 0),
		// }
		_, err := s.openmeter.GetValuesByMeterId(ctx, meterId)
		if err != nil {
			// handle err
		}
		var total int64 = 2000000

		// Report usage to Stripe
		// We use action=set so even if this webhook get called multiple time
		// we still end up with only one usage record for the same period.
		// We report on `CurrentPeriodStart` because it's going to be the
		// same between Webhook calls and we add one as Stripe doesn't allow
		// to report usage on the exact start and end time of the subscription.
		recordParams := &stripe.UsageRecordParams{
			Action:           stripe.String("set"),
			Quantity:         stripe.Int64(total),
			SubscriptionItem: stripe.String(item.ID),
			Timestamp:        stripe.Int64(subscription.CurrentPeriodStart + 1),
		}
		_, err = usagerecord.New(recordParams)
		// handle err
	}
}
