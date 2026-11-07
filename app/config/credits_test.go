package config

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestCreditsWriterConfiguration(t *testing.T) {
	v := viper.New()
	SetViperDefaults(v, pflag.NewFlagSet("test", pflag.ContinueOnError))
	CreditsWriterConfiguration{Revision: "embedded", Version: "embedded-version"}.SetDefaults(v, "credits")

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader("credits:\n  writer:\n    revision: from-yaml\n")))
	t.Setenv("CREDITS_WRITER_VERSION", "from-env")

	var conf Configuration
	require.NoError(t, v.Unmarshal(&conf, viper.DecodeHook(DecodeHook())))
	require.Equal(t, "from-yaml", conf.Credits.Writer.CommitSHA())
	require.Equal(t, "from-env", conf.Credits.Writer.Version)
}
