package main

import (
	"context"
	"fmt"
	"os"
	"time"

	openmeter "github.com/openmeterio/openmeter/api/client/go"
	stripe "github.com/stripe/stripe-go/v74"
)

// Stripe Key.
// Go to https://dashboard.stripe.com/test/apikeys to obtain yours
var stripeKey = os.Getenv("STRIPE_KEY") // sk_test_...""

func main() {
	stripe.Key = stripeKey

	if len(os.Args) == 1 {
		fmt.Printf("provide argument: setup or report\n")
		return
	}
	mode := os.Args[1]

	// Setup Stripe test product, price and customer
	if mode == "setup" {
		err := SetupStripe()
		if err != nil {
			panic(err)
		}
	} else if mode == "report" {
		// Initialize OpenMeter client
		om, err := openmeter.NewClient("http://localhost:8888")
		if err != nil {
			panic(err.Error())
		}

		// Report usage
		reportingFrequency := time.Second // change it in real app
		report := NewReport(context.Background(), om, reportingFrequency)
		err = report.Run()
		if err != nil {
			panic(err.Error())
		}
	} else {
		fmt.Printf("Unknown mode: %s try: setup or report\n", mode)
	}

	fmt.Println("done")
	os.Exit(0)
}
