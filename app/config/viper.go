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

// ViperKeyPrefixer is a helper to prepend prefix to a key name.
type ViperKeyPrefixer func(s string) string

const delimiter = "."

// NewViperKeyPrefixer returns a new ViperKeyPrefixer which prepends a dot delimited prefix calculated by concatenating provided
// prefixes in the order they appear in prefixes list.
//
//	prefixer := NewViperKeyPrefixer("a", "b")
//	s := prefixer("c")
//	fmt.Println(s) // -> "a.b.c"
func NewViperKeyPrefixer(prefixes ...string) ViperKeyPrefixer {
	var prefix string

	for _, p := range prefixes {
		if p == "" {
			continue
		}

		if prefix == "" {
			prefix = p
		} else {
			prefix += delimiter + p
		}
	}

	if prefix == "" {
		return func(s string) string { return s }
	} else {
		return func(s string) string { return prefix + delimiter + s }
	}
}
