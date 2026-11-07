package config

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestMetadataConfiguration(t *testing.T) {
	v := viper.New()
	SetViperDefaultsWithMetadata(v, pflag.NewFlagSet("test", pflag.ContinueOnError), MetadataConfiguration{Revision: "embedded", Version: "embedded-version"})

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader("metadata:\n  revision: from-yaml\n")))
	t.Setenv("METADATA_VERSION", "from-env")

	var conf Configuration
	require.NoError(t, v.Unmarshal(&conf, viper.DecodeHook(DecodeHook())))
	require.Equal(t, "from-yaml", conf.Metadata.Revision)
	require.Equal(t, "from-env", conf.Metadata.Version)
}
