package config

import (
	"errors"
	"fmt"
	"net/url"

	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

type SvixConfiguration notificationwebhook.SvixConfig

func (c SvixConfiguration) Validate() error {
	if c.APIToken == "" {
		return errors.New("no API token provided")
	}

	if c.ServerURL != "" {
		if _, err := url.Parse(c.ServerURL); err != nil {
			return fmt.Errorf("invalid server URL: %w", err)
		}
	}

	return nil
}
