package models

import "fmt"

type NamespacedKey struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

func (k NamespacedKey) Validate() error {
	if k.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if k.Key == "" {
		return fmt.Errorf("key is required")
	}

	return nil
}
