package config

import "github.com/spf13/viper"

type SchemaRegistryConfiguration struct {
	URL      string
	Username string
	Password string
}

// configureSchemaRegistry configures some defaults in the Viper instance.
func configureSchemaRegistry(v *viper.Viper) {
	v.SetDefault("schemaRegistry.url", "")
	v.SetDefault("schemaRegistry.username", "")
	v.SetDefault("schemaRegistry.password", "")
}
