// Copyright © 2023 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"

	stripe "github.com/stripe/stripe-go/v74"
	stripeCustomer "github.com/stripe/stripe-go/v74/customer"
	stripePrice "github.com/stripe/stripe-go/v74/price"
	stripeProduct "github.com/stripe/stripe-go/v74/product"
	stripeSubscription "github.com/stripe/stripe-go/v74/subscription"
)

// Meter in your config, we use it to map our price to this meter
var meterId = "m1"

func SetupStripe() error {
	// Create a Stripe Product
	product, err := stripeProduct.New(&stripe.ProductParams{
		Name: stripe.String("Execution Duration"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Stripe product created: https://dashboard.stripe.com/test/products/%s\n", product.ID)

	// Create a metered Stripe Price
	price, err := stripePrice.New(&stripe.PriceParams{
		Product: &product.ID,
		// The meter ID this price belongs to
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Recurring: &stripe.PriceRecurringParams{
			Interval:  stripe.String(string(stripe.PlanIntervalMonth)),
			UsageType: stripe.String(string(stripe.PlanUsageTypeMetered)),
		},
		BillingScheme: stripe.String(string(stripe.PlanBillingSchemePerUnit)),
		UnitAmount:    stripe.Int64(10), // cents
	})
	if err != nil {
		return err
	}
	fmt.Printf("Stripe price created: https://dashboard.stripe.com/test/prices/%s\n", price.ID)

	// Create a Stripe customer
	customer, err := stripeCustomer.New(&stripe.CustomerParams{
		Name: stripe.String("My Awesome Customer"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Stripe customer created: https://dashboard.stripe.com/test/customers/%s\n", customer.ID)

	// Start a new Stripe subscription for customer with the price created above
	subscription, err := stripeSubscription.New(&stripe.SubscriptionParams{
		Customer: &customer.ID,
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: &price.ID,
				Metadata: map[string]string{
					"om_meter_id": meterId,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("Stripe subscription created: https://dashboard.stripe.com/test/subscriptions/%s\n", subscription.ID)

	return nil
}
