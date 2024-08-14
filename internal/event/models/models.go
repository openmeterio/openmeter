package models

import "errors"

type FeatureKeyAndID struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

func (f FeatureKeyAndID) Validate() error {
	if f.Key == "" {
		return errors.New("key is required")
	}

	if f.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type NamespaceID struct {
	ID string `json:"id"`
}

func (i NamespaceID) Validate() error {
	if i.ID == "" {
		return errors.New("namespace-id is required")
	}

	return nil
}
