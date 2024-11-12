package config

import "github.com/spf13/viper"

type ProductCatalogConfiguration struct {
	Enabled bool
}

func (c ProductCatalogConfiguration) Validate() error {
	return nil
}

func ConfigureProductCatalog(v *viper.Viper) {
	v.SetDefault("productcatalog.enabled", false)
}
