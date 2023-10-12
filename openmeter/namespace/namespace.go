// Package namespace adds a concept of tenancy to OpenMeter allowing to segment clients.
package namespace

import (
	"github.com/openmeterio/openmeter/internal/namespace"
)

// Manager is responsible for managing namespaces in different components.
type Manager = namespace.Manager

type ManagerConfig = namespace.ManagerConfig

func NewManager(config ManagerConfig) (*Manager, error) {
	return namespace.NewManager(config)
}

// Handler is responsible for creating a namespace in a given component.
//
// An empty name means a default namespace is supposed to be created.
// The concept of a default namespace is implementation specific.
//
// The behavior for trying to create a namespace that already exists is unspecified at the moment.
type Handler = namespace.Handler
