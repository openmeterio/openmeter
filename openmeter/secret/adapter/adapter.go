package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/secret"
)

func New() secret.Adapter {
	return &adapter{}
}

var _ secret.Adapter = (*adapter)(nil)

type adapter struct{}
