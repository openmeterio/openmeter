package models

import (
	"fmt"
)

type NamespacedID struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
}

func (i NamespacedID) Validate() error {
	if i.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if i.ID == "" {
		return fmt.Errorf("id is required")
	}

	return nil
}
