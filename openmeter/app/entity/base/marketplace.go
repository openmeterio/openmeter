package appentitybase

import (
	"errors"
	"fmt"
)

type MarketplaceListing struct {
	Type         AppType      `json:"type"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	IconURL      string       `json:"iconUrl"`
	Capabilities []Capability `json:"capabilities"`
}

func (p MarketplaceListing) Validate() error {
	if p.Type == "" {
		return errors.New("type is required")
	}

	if p.Name == "" {
		return errors.New("name is required")
	}

	if p.Description == "" {
		return errors.New("description is required")
	}

	if p.IconURL == "" {
		return errors.New("icon url is required")
	}

	for i, capability := range p.Capabilities {
		if err := capability.Validate(); err != nil {
			return fmt.Errorf("error validating capability a position %d: %w", i, err)
		}
	}

	return nil
}

type Capability struct {
	Type        CapabilityType `json:"type"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
}

func (c Capability) Validate() error {
	if c.Key == "" {
		return errors.New("key is required")
	}

	if c.Name == "" {
		return errors.New("name is required")
	}

	if c.Description == "" {
		return errors.New("description is required")
	}

	return nil
}
