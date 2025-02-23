package adapter

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/portal"
)

type Config struct {
	Secret string
	Expire time.Duration
}

func (c Config) Validate() error {
	if c.Secret == "" {
		return errors.New("secret must not be empty")
	}

	if c.Expire == 0 {
		return errors.New("expire must not be 0")
	}

	return nil
}

func New(config Config) (portal.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		secret: []byte(config.Secret),
		expire: config.Expire,
	}, nil
}

var _ portal.Service = (*adapter)(nil)

type adapter struct {
	secret []byte
	expire time.Duration
}
