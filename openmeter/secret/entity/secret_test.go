package secretentity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSecretIDValidateRequiresMatchingNamespace(t *testing.T) {
	// given:
	// - a secret identifier whose own namespace differs from its app namespace
	// when:
	// - the secret identifier is validated
	// then:
	// - validation rejects the mixed namespace reference
	secretID := NewSecretID(
		app.AppID{Namespace: "app-namespace", ID: "app-id"},
		"secret-id",
		"secret-key",
	)
	secretID.Namespace = "secret-namespace"

	err := secretID.Validate()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err))
}

func TestUpdateAppSecretInputValidateBindsSecretReference(t *testing.T) {
	// given:
	// - a valid app-owned secret reference
	// when:
	// - an update uses another app or another secret key
	// then:
	// - validation rejects the mixed reference while accepting the exact owner and key
	appID := app.AppID{Namespace: "namespace", ID: "app-id"}
	secretID := NewSecretID(appID, "secret-id", "secret-key")

	for _, testCase := range []struct {
		name    string
		input   UpdateAppSecretInput
		wantErr bool
	}{
		{
			name: "exact reference",
			input: UpdateAppSecretInput{
				AppID:    appID,
				SecretID: secretID,
				Key:      secretID.Key,
				Value:    "new-value",
			},
		},
		{
			name: "different app",
			input: UpdateAppSecretInput{
				AppID: app.AppID{
					Namespace: appID.Namespace,
					ID:        "another-app-id",
				},
				SecretID: secretID,
				Key:      secretID.Key,
				Value:    "new-value",
			},
			wantErr: true,
		},
		{
			name: "different key",
			input: UpdateAppSecretInput{
				AppID:    appID,
				SecretID: secretID,
				Key:      "another-key",
				Value:    "new-value",
			},
			wantErr: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.input.Validate()
			if testCase.wantErr {
				require.Error(t, err)
				require.True(t, models.IsGenericValidationError(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
