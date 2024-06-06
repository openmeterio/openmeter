package config

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/sagikazarmark/mapstructurex"
)

func DecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		mapstructurex.MapDecoderHookFunc(),
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)
}
