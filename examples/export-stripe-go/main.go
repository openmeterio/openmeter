package main

import (
	"log"
	"net/http"
	"os"

	openmeter "github.com/openmeterio/openmeter/api"
	stripe "github.com/stripe/stripe-go/v74"
)

// Stripe Key.
// Go to https://dashboard.stripe.com/test/apikeys to obtain yours
var stripeKey = os.Getenv("STRIPE_KEY") // sk_test_...""

// Stripe Webhook Secret
// Replace this endpoint secret with your endpoint's unique secret
// If you are testing with the CLI, find the secret by running 'stripe listen'
// If you are using an endpoint defined with the API or dashboard, look in your webhook settings
// at https://dashboard.stripe.com/webhooks
var endpointSecret = os.Getenv("STRIPE_WEBHOOK_SECRET") // "whsec_...."

func main() {
	stripe.Key = stripeKey

	// Setup Stripe test product, price and customer
	err := SetupStripe()
	if err != nil {
		panic(err)
	}

	// Initialize OpenMeter client
	om, err := openmeter.NewClient("http://localhost:8888")
	if err != nil {
		panic(err.Error())
	}

	// Initialize server
	server := Server{
		endpointSecret: endpointSecret,
		openmeter:      om,
	}

	// Run server
	http.HandleFunc("/webhook", server.handleWebhook)
	addr := "localhost:4242"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
