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
}
