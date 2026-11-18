package config

import "github.com/spf13/viper"

type ProductCatalogConfiguration struct {
	Subscription SubscriptionConfiguration
}

func (c ProductCatalogConfiguration) Validate() error {
	return nil
}

func ConfigureProductCatalog(v *viper.Viper) {
}

type SubscriptionConfiguration struct {
	MultiSubscriptionNamespaces []string
	// BillingStartsAtNamespaces lists the namespaces that can create subscriptions whose billing starts
	// later than the subscription itself.
	BillingStartsAtNamespaces []string
}
