// Package namespace adds a concept of tenancy to OpenMeter allowing to segment clients.
package namespace

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Manager is responsible for managing namespaces in different components.
type Manager struct {
	config ManagerConfig
}

type ManagerConfig struct {
	MeterRepository   meter.Repository
	DefaultNamespace  string
	DisableManagement bool
	Handlers          []Handler
}

func NewManager(config ManagerConfig) (*Manager, error) {
	if config.MeterRepository == nil {
		return nil, errors.New("meter repository is required")
	}
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
	DeleteNamespace(ctx context.Context, name string, meters []models.Meter) error
}

// CreateNamespace orchestrates namespace creation across different components.
func (m Manager) CreateNamespace(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("cannot create empty namespace")
	}

	return m.createNamespace(ctx, name)
}

// DeleteNamespace orchestrates namespace creation across different components.
func (m Manager) DeleteNamespace(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("cannot delete empty namespace")
	}

	if name == m.config.DefaultNamespace {
		return errors.New("cannot delete default namespace")
	}

	return m.deleteNamespace(ctx, name)
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

func (m Manager) IsManagementDisabled() bool {
	return m.config.DisableManagement
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

// TODO: introduce some resiliency (eg. retries or rollbacks in case a component fails to delete a namespace).
func (m Manager) deleteNamespace(ctx context.Context, name string) error {
	var errs []error

	// Retrieve meters belonging to the namespace
	meters, err := m.config.MeterRepository.ListMeters(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to list meters: %w", err)
	}

	for _, handler := range m.config.Handlers {
		err := handler.DeleteNamespace(ctx, name, meters)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
