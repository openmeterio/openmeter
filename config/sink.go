package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type SinkConfiguration struct {
	MinCommitCount   int
	MaxCommitWait    time.Duration
	NamespaceRefetch time.Duration
}

func (c SinkConfiguration) Validate() error {
	if c.MinCommitCount < 1 {
		return errors.New("MinCommitCount must be greater than 0")
	}

	if c.MaxCommitWait < 1 {
		return errors.New("MaxCommitWait must be greater than 0")
	}

	if c.NamespaceRefetch < 1 {
		return errors.New("NamespaceRefetch must be greater than 0")
	}

	return nil
}

// Configure configures some defaults in the Viper instance.
func configureSink(v *viper.Viper) {
	v.SetDefault("sink.minCommitCount", 500)
	v.SetDefault("sink.maxCommitWait", "5s")
	v.SetDefault("sink.namespaceRefetch", "15s")
}
