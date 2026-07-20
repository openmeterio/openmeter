package config

import "testing"

func TestCreditsConfigurationIsCustomCurrencyEnabled(t *testing.T) {
	tests := []struct {
		name           string
		creditsEnabled bool
		customCurrency bool
		expected       bool
	}{
		{name: "both disabled"},
		{name: "credits disabled", customCurrency: true},
		{name: "custom currency disabled", creditsEnabled: true},
		{name: "both enabled", creditsEnabled: true, customCurrency: true, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CreditsConfiguration{
				Enabled:        tt.creditsEnabled,
				CustomCurrency: tt.customCurrency,
			}

			if actual := config.IsCustomCurrencyEnabled(); actual != tt.expected {
				t.Fatalf("expected custom currency enabled to be %t, got %t", tt.expected, actual)
			}
		})
	}
}
