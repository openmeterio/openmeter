// Package namespace adds a concept of tenancy to OpenMeter allowing to segment clients.
package namespace

import (
	"context"
	"errors"
)

// Manager is responsible for managing namespaces in different components.
type Manager struct {
	config ManagerConfig
}

type ManagerConfig struct {
	DefaultNamespace string
	Handlers         []Handler
}

func NewManager(config ManagerConfig) (*Manager, error) {
	if config.DefaultNamespace == "" {
		return nil, errors.New("default namespace is required")
	}

	manager := Manager{
		config: config,
	}

	return &manager, nil
}

// Handler is responsible for creating a namespace in a given component.
//
// An empty name means a default namespace is supposed to be created.
// The concept of a default namespace is implementation specific.
//
// The behavior for trying to create a namespace that already exists is unspecified at the moment.
type Handler interface {
	CreateNamespace(ctx context.Context, name string) error
}

// CreateNamespace orchestrates namespace creation across different components.
func (m Manager) CreateNamespace(ctx context.Context, name string) error {
	// TODO: validate name

	return m.createNamespace(ctx, name)
}

// CreateDefaultNamespace orchestrates the creation of a default namespace.
//
// The concept of a default namespace is implementation specific.
func (m Manager) CreateDefaultNamespace(ctx context.Context) error {
	return m.createNamespace(ctx, m.config.DefaultNamespace)
}

func (m Manager) GetDefaultNamespace() string {
	return m.config.DefaultNamespace
}

// TODO: introduce some resiliency (eg. retries or rollbacks in case a component fails to create a namespace).
func (m Manager) createNamespace(ctx context.Context, name string) error {
	var errs []error

	for _, handler := range m.config.Handlers {
		err := handler.CreateNamespace(ctx, name)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
