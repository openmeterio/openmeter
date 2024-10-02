package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/secret"
)

func New() (secret.Adapter, error) {
	return &adapter{}, nil
}

var _ secret.Adapter = (*adapter)(nil)

type adapter struct{}
