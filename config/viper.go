package config

import (
	"github.com/mitchellh/mapstructure"
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
