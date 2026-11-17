package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRTEConfiguration(t *testing.T) {
	v := viper.New()
	ConfigureRTE(v)
	var cfg RTEConfiguration
	require.NoError(t, v.UnmarshalKey("rte", &cfg))
	require.False(t, cfg.Enabled)
	require.Equal(t, 10, cfg.ShadowSamplePercent)
	require.NoError(t, (RTEConfiguration{}).Validate())

	cfg.Enabled = true
	for _, sample := range []int{1, 10, 100} {
		cfg.ShadowSamplePercent = sample
		require.NoError(t, cfg.Validate())
	}

	for _, sample := range []int{-1, 0, 101} {
		cfg.ShadowSamplePercent = sample
		require.ErrorContains(t, cfg.Validate(), "shadow sample percent")
	}
}
