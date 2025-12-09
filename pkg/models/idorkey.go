package models

import (
	"errors"
	"fmt"
)

// NamespacedIDOrKey represents a namespaced id or key (when we don't know if a specific entity is either a key or an id)
// If the type of entity is known please use the ref package or the NamespacedID or the NamespacedKey types
type NamespacedIDOrKey struct {
	Namespace string `json:"namespace"`
	IDOrKey   string `json:"idOrKey"`
}

func (n NamespacedIDOrKey) Validate() error {
	var errs []error

	if n.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if n.IDOrKey == "" {
		errs = append(errs, fmt.Errorf("idOrKey is required"))
	}

	return errors.Join(errs...)
}
