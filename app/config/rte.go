package config

import "github.com/spf13/viper"

type RTEConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
}

func ConfigureRTE(v *viper.Viper) {
	v.SetDefault("rte.dlq.topic", "om_sys.rte_dlq")
	v.SetDefault("rte.consumerGroupName", "om_rte")

	ConfigureConsumer(v, "rte")
}
