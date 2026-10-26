package svix

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	svix "github.com/svix/svix-webhooks/v2/go"
	svixutils "github.com/svix/svix-webhooks/v2/go/utils"

	"github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/providers/svix/testutils"
	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("requires a live Svix server")
	}

	env := testutils.NewTestEnv(t)

	provider, err := New(Config{
		Client: env.Client,
		Logger: env.Logger,
		Tracer: env.Tracer,
	})
	require.NoError(t, err, "failed to create svix provider")
	require.NotNil(t, provider, "provider should not be nil")

	t.Run("Namespace", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			t.Parallel()

			// Given a unique namespace with cleanup independent of test cancellation.
			namespace := v4.ProviderNamespace{
				ID:   ulid.Make().String(),
				Name: "Namespace integration test",
				Metadata: models.Metadata{
					"environment": "initial",
					"obsolete":    "value",
				},
			}

			ctx := t.Context()

			t.Cleanup(func() {
				// The test context is canceled before cleanup runs.
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				defer cancel()

				err := provider.DeleteNamespace(cleanupCtx, v4.DeleteNamespaceInput{ID: namespace.ID})
				require.NoError(t, err, "failed to clean up namespace")
			})

			input := v4.CreateNamespaceInput{
				ProviderNamespace: namespace,
				IdempotencyKey:    new(ulid.Make().String()),
			}

			// When the namespace is created.
			created, err := provider.CreateNamespace(ctx, input)
			require.NoError(t, err)
			require.NotNil(t, created)

			// Then provider data identifies the Svix application, not its Notification UID.
			applicationID := created.ProviderData["svix.application_id"]
			require.NotEmpty(t, applicationID)
			require.NotEqual(t, namespace.ID, applicationID)

			providerData := v4.ProviderData{"svix.application_id": applicationID}

			namespace.ProviderData = providerData

			AssertNamespace(t, created, namespace)

			createdStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
			require.NoError(t, err)
			AssertNamespace(t, createdStored, namespace)
			require.Equal(t, created, createdStored)

			createdApplication, err := env.Client.Application().Get(t.Context(), namespace.ProviderData["svix.application_id"])
			require.NoError(t, err)
			AssertNamespaceWithApplication(t, createdApplication, namespace)

			t.Run("RetryWithSameIdempotencyKey", func(t *testing.T) {
				// Given the exact request that created the namespace.
				// When creation is retried with the same caller-supplied key.
				repeated, err := provider.CreateNamespace(t.Context(), input)
				require.NoError(t, err)

				// Then both identities, metadata, and provider data remain unchanged.
				AssertNamespace(t, repeated, namespace)

				repeatedStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
				require.NoError(t, err)
				AssertNamespace(t, repeatedStored, namespace)
				require.Equal(t, repeated, repeatedStored)

				repeatedApplication, err := env.Client.Application().Get(t.Context(), namespace.ProviderData["svix.application_id"])
				require.NoError(t, err)
				AssertNamespaceWithApplication(t, repeatedApplication, namespace)
			})

			t.Run("GetExistingWithoutOverwriting", func(t *testing.T) {
				// Given the existing UID with different requested attributes and no caller key.
				input := v4.CreateNamespaceInput{
					ProviderNamespace: v4.ProviderNamespace{
						ID:   namespace.ID,
						Name: "Must not overwrite the existing name",
						Metadata: models.Metadata{
							"unexpected": "value",
						},
					},
				}

				// When creation recovers the application by UID.
				existing, err := provider.CreateNamespace(t.Context(), input)
				require.NoError(t, err)

				// Then the existing state and provider data are returned without mutation.
				AssertNamespace(t, existing, namespace)

				existingStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
				require.NoError(t, err)
				AssertNamespace(t, existingStored, namespace)
				require.Equal(t, existing, existingStored)

				existingApplication, err := env.Client.Application().Get(t.Context(), namespace.ProviderData["svix.application_id"])
				require.NoError(t, err)
				AssertNamespaceWithApplication(t, existingApplication, namespace)
			})

			t.Run("Update", func(t *testing.T) {
				t.Run("PartialUpdate", func(t *testing.T) {
					tests := []struct {
						name           string
						input          v4.UpdateNamespaceInput
						modifyExpected func(*v4.ProviderNamespace)
					}{
						{
							name: "NoChanges",
						},
						{
							name:  "NameOnly",
							input: v4.UpdateNamespaceInput{Name: new("Renamed")},
							modifyExpected: func(expected *v4.ProviderNamespace) {
								expected.Name = "Renamed"
							},
						},
						{
							name:  "MetadataOnly",
							input: v4.UpdateNamespaceInput{Metadata: new(models.Metadata{"replacement": "value"})},
							modifyExpected: func(expected *v4.ProviderNamespace) {
								expected.Metadata = models.Metadata{"replacement": "value"}
							},
						},
					}

					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							// Given the same complete state, independent of sibling updates.
							_, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(namespace))
							require.NoError(t, err)

							input := tt.input
							input.ID = namespace.ID
							input.ProviderData = providerData

							expected := namespace
							if tt.modifyExpected != nil {
								tt.modifyExpected(&expected)
							}

							// When only the selected fields are supplied.
							updated, err := provider.UpdateNamespace(t.Context(), input)
							require.NoError(t, err)

							// Then omitted fields and both identities remain unchanged.
							AssertNamespace(t, updated, expected)

							stored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
							require.NoError(t, err)
							AssertNamespace(t, stored, expected)

							application, err := env.Client.Application().Get(t.Context(), applicationID)
							require.NoError(t, err)
							AssertNamespaceWithApplication(t, application, expected)
						})
					}
				})

				t.Run("ReplaceMetadata", func(t *testing.T) {
					// Given nonempty metadata, independently of earlier sibling tests.
					seeded, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(namespace))
					require.NoError(t, err)
					AssertNamespace(t, seeded, namespace)

					seededStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
					require.NoError(t, err)
					AssertNamespace(t, seededStored, namespace)
					require.Equal(t, seeded, seededStored)

					seededApplication, err := env.Client.Application().Get(t.Context(), namespace.ProviderData["svix.application_id"])
					require.NoError(t, err)
					AssertNamespaceWithApplication(t, seededApplication, namespace)

					input := v4.ProviderNamespace{
						ID:   namespace.ID,
						Name: "Updated namespace",
						Metadata: models.Metadata{
							"environment": "updated",
							"new":         "value",
						},
						ProviderData: providerData,
					}

					// When the namespace's name and complete metadata are updated.
					updated, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(input))
					require.NoError(t, err)

					// Then obsolete metadata is removed while provider data is preserved.
					AssertNamespace(t, updated, input)

					updatedStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: input.ID})
					require.NoError(t, err)
					AssertNamespace(t, updatedStored, input)
					require.Equal(t, updated, updatedStored)

					updatedApplication, err := env.Client.Application().Get(t.Context(), input.ProviderData["svix.application_id"])
					require.NoError(t, err)
					AssertNamespaceWithApplication(t, updatedApplication, input)
				})

				t.Run("ClearMetadataWithNil", func(t *testing.T) {
					// Given nonempty metadata on the same namespace.
					seeded, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(namespace))
					require.NoError(t, err)
					AssertNamespace(t, seeded, namespace)

					seededStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
					require.NoError(t, err)
					AssertNamespace(t, seededStored, namespace)
					require.Equal(t, seeded, seededStored)

					seededApplication, err := env.Client.Application().Get(t.Context(), namespace.ProviderData["svix.application_id"])
					require.NoError(t, err)
					AssertNamespaceWithApplication(t, seededApplication, namespace)

					input := v4.ProviderNamespace{
						ID:           namespace.ID,
						Name:         namespace.Name,
						Metadata:     nil,
						ProviderData: providerData,
					}

					// When metadata is explicitly replaced with a nil map.
					updated, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(input))
					require.NoError(t, err)

					// Then user metadata is cleared, but provider data is retained.
					AssertNamespace(t, updated, input)

					updatedStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: input.ID})
					require.NoError(t, err)
					AssertNamespace(t, updatedStored, input)
					require.Equal(t, updated, updatedStored)

					updatedApplication, err := env.Client.Application().Get(t.Context(), input.ProviderData["svix.application_id"])
					require.NoError(t, err)
					AssertNamespaceWithApplication(t, updatedApplication, input)
				})

				t.Run("ClearMetadataWithEmptyMap", func(t *testing.T) {
					// Given nonempty metadata, even if the nil-clearing sibling ran.
					seeded, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(namespace))
					require.NoError(t, err)
					AssertNamespace(t, seeded, namespace)

					seededStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
					require.NoError(t, err)
					AssertNamespace(t, seededStored, namespace)
					require.Equal(t, seeded, seededStored)

					seededApplication, err := env.Client.Application().Get(t.Context(), namespace.ProviderData["svix.application_id"])
					require.NoError(t, err)
					AssertNamespaceWithApplication(t, seededApplication, namespace)

					input := v4.ProviderNamespace{
						ID:           namespace.ID,
						Name:         namespace.Name,
						Metadata:     models.Metadata{},
						ProviderData: providerData,
					}

					// When metadata is explicitly set to an empty map.
					updated, err := provider.UpdateNamespace(t.Context(), ToUpdateNamespaceInput(input))
					require.NoError(t, err)

					// Then user metadata is cleared, but provider data is retained.
					AssertNamespace(t, updated, input)

					updatedStored, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: input.ID})
					require.NoError(t, err)
					AssertNamespace(t, updatedStored, input)
					require.Equal(t, updated, updatedStored)

					updatedApplication, err := env.Client.Application().Get(t.Context(), input.ProviderData["svix.application_id"])
					require.NoError(t, err)
					AssertNamespaceWithApplication(t, updatedApplication, input)
				})
			})

			t.Run("Delete", func(t *testing.T) {
				// Given the namespace created by the parent, whether or not Update ran.
				input := v4.DeleteNamespaceInput{ID: namespace.ID}

				// When the namespace is deleted.
				err := provider.DeleteNamespace(t.Context(), input)
				require.NoError(t, err)

				// Then it is no longer retrievable and no provider data is returned.
				deleted, err := provider.GetNamespace(t.Context(), v4.GetNamespaceInput{ID: namespace.ID})
				require.Error(t, err)
				require.Nil(t, deleted)

				var providerErr *v4.ProviderError
				require.ErrorAs(t, err, &providerErr)
				require.Equal(t, v4.ProviderErrorCodeResourceNotFound, providerErr.Code)

				t.Run("UpdateMissing", func(t *testing.T) {
					tests := []struct {
						name  string
						input v4.UpdateNamespaceInput
					}{
						{name: "NoChanges", input: v4.UpdateNamespaceInput{ID: namespace.ID}},
						{name: "NameOnly", input: v4.UpdateNamespaceInput{ID: namespace.ID, Name: new("Must not recreate")}},
					}

					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							// Given a deleted namespace, when an update is attempted.
							updated, err := provider.UpdateNamespace(t.Context(), tt.input)

							// Then even an empty update reports absence rather than success.
							require.Nil(t, updated)

							var providerErr *v4.ProviderError
							require.ErrorAs(t, err, &providerErr)
							require.Equal(t, v4.ProviderErrorCodeResourceNotFound, providerErr.Code)
						})
					}
				})

				t.Run("DeleteAgain", func(t *testing.T) {
					// Given the namespace already deleted by the parent.
					// When deletion is retried, then it still succeeds.
					err := provider.DeleteNamespace(t.Context(), input)
					require.NoError(t, err)
				})
			})
		})

		t.Run("Channel", func(t *testing.T) {
			t.Parallel()

			namespaceID := ulid.Make().String()

			eventTypes := []v4.EventType{
				{
					Name:    "test." + strings.ToLower(namespaceID) + ".created",
					Version: 1,
				},
				{
					Name:    "test." + strings.ToLower(namespaceID) + ".updated",
					Version: 1,
				},
			}

			// Register only test-owned event types, before namespace cleanup is registered.
			// Cleanup removes the namespace/endpoints before removing these types.
			for _, eventType := range eventTypes {
				_, err := env.Client.EventType().Create(t.Context(), svix.EventTypeIn{
					Name:        eventType.String(),
					Description: "Channel integration test",
				}, nil)
				require.NoError(t, err)

				ctx := t.Context()

				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
					defer cancel()

					err := env.Client.EventType().Delete(
						cleanupCtx,
						eventType.String(),
						&svix.EventTypeDeleteOptions{Expunge: new(true)},
					)
					require.NoError(t, err, "failed to clean up event type")
				})
			}

			ctx := t.Context()

			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				defer cancel()

				err := provider.DeleteNamespace(cleanupCtx, v4.DeleteNamespaceInput{ID: namespaceID})
				require.NoError(t, err, "failed to clean up channel namespace")
			})

			_, err := provider.CreateNamespace(t.Context(), v4.CreateNamespaceInput{
				ProviderNamespace: v4.ProviderNamespace{
					ID:   namespaceID,
					Name: "Channel integration test",
				},
			})
			require.NoError(t, err)

			receiver := testutils.NewWebhookServer(t, env.WebhookHost)

			initialSecret, err := webhooksecret.NewSigningSecretWithDefaultSize()
			require.NoError(t, err)

			t.Run("Create", func(t *testing.T) {
				// Given a provisioned namespace and a webhook configuration.
				input := v4.CreateChannelInput[v4.WebhookConfig]{
					Namespace: namespaceID,
					ChannelID: ulid.Make().String(),
					Name:      "Initial channel",
					Config: v4.WebhookConfig{
						URL:           receiver.URL + "/initial",
						SigningSecret: initialSecret,
						CustomHeaders: map[string]string{
							"x-test-environment": "initial",
							"x-test-obsolete":    "value",
						},
					},
					EventTypes:     eventTypes[:1],
					IdempotencyKey: new(ulid.Make().String()),
				}

				// When the channel is created.
				created, err := provider.CreateChannel(t.Context(), input)
				require.NoError(t, err)
				require.NotNil(t, created)

				// Then provider data identifies the endpoint independently of its UID.
				endpointID := created.ProviderData["svix.endpoint_id"]
				require.NotEmpty(t, endpointID)
				require.NotEqual(t, input.ChannelID, endpointID)

				expected := v4.CreateChannelInput[v4.WebhookConfig]{
					Namespace:  input.Namespace,
					ChannelID:  input.ChannelID,
					Name:       input.Name,
					Config:     input.Config,
					EventTypes: input.EventTypes,
					ProviderData: v4.ProviderData{
						"svix.endpoint_id": endpointID,
					},
				}

				expectedResult := v4.ChannelOperationResult{
					State:        v4.ChannelOperationStateInSync,
					ProviderData: expected.ProviderData,
				}

				AssertChannel(t, created, expectedResult)

				createdStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
					Namespace:    expected.Namespace,
					ChannelID:    expected.ChannelID,
					ProviderData: expected.ProviderData,
				})
				require.NoError(t, err)
				AssertChannel(t, createdStored, expectedResult)

				createdEndpoint, err := env.Client.Endpoint().Get(t.Context(), expected.Namespace, endpointID)
				require.NoError(t, err)
				AssertChannelWithEndpoint(t, createdEndpoint, expected)

				createdHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), expected.Namespace, endpointID)
				require.NoError(t, err)
				require.NotNil(t, createdHeaders)
				require.Empty(t, createdHeaders.Sensitive)
				require.True(t, maps.Equal(expected.Config.CustomHeaders, createdHeaders.Headers),
					"expected headers %v, got %v", expected.Config.CustomHeaders, createdHeaders.Headers)

				createdSecret, err := env.Client.Endpoint().GetSecret(t.Context(), expected.Namespace, endpointID)
				require.NoError(t, err)
				require.NotNil(t, createdSecret)
				require.NotEmpty(t, createdSecret.Key)
				require.NotEmpty(t, expected.Config.SigningSecret)
				require.Equal(
					t,
					strings.TrimPrefix(expected.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
					strings.TrimPrefix(createdSecret.Key, webhooksecret.SigningSecretPrefix),
				)

				t.Run("RetryWithSameIdempotencyKey", func(t *testing.T) {
					// Given the original creation request.
					// When it is retried with the same key.
					repeated, err := provider.CreateChannel(t.Context(), input)
					require.NoError(t, err)

					// Then the same endpoint, provider data, and secret are returned.
					AssertChannel(t, repeated, expectedResult)

					repeatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
						Namespace:    expected.Namespace,
						ChannelID:    expected.ChannelID,
						ProviderData: expected.ProviderData,
					})
					require.NoError(t, err)
					AssertChannel(t, repeatedStored, expectedResult)

					repeatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), expected.Namespace, endpointID)
					require.NoError(t, err)
					AssertChannelWithEndpoint(t, repeatedEndpoint, expected)

					repeatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), expected.Namespace, endpointID)
					require.NoError(t, err)
					require.NotNil(t, repeatedHeaders)
					require.Empty(t, repeatedHeaders.Sensitive)
					require.True(t, maps.Equal(expected.Config.CustomHeaders, repeatedHeaders.Headers),
						"expected headers %v, got %v", expected.Config.CustomHeaders, repeatedHeaders.Headers)

					repeatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), expected.Namespace, endpointID)
					require.NoError(t, err)
					require.NotNil(t, repeatedSecret)
					require.NotEmpty(t, repeatedSecret.Key)
					require.NotEmpty(t, expected.Config.SigningSecret)
					require.Equal(
						t,
						strings.TrimPrefix(expected.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
						strings.TrimPrefix(repeatedSecret.Key, webhooksecret.SigningSecretPrefix),
					)
				})

				t.Run("RetryWithoutIdempotencyKey", func(t *testing.T) {
					// Given the existing endpoint UID without a caller-supplied key.
					retryInput := input
					retryInput.IdempotencyKey = nil

					// When creation recovers through the UID-conflict path.
					repeated, err := provider.CreateChannel(t.Context(), retryInput)
					require.NoError(t, err)

					// Then no replacement endpoint or secret is created.
					AssertChannel(t, repeated, expectedResult)

					repeatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
						Namespace:    expected.Namespace,
						ChannelID:    expected.ChannelID,
						ProviderData: expected.ProviderData,
					})
					require.NoError(t, err)
					AssertChannel(t, repeatedStored, expectedResult)

					repeatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), expected.Namespace, endpointID)
					require.NoError(t, err)
					AssertChannelWithEndpoint(t, repeatedEndpoint, expected)

					repeatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), expected.Namespace, endpointID)
					require.NoError(t, err)
					require.NotNil(t, repeatedHeaders)
					require.Empty(t, repeatedHeaders.Sensitive)
					require.True(t, maps.Equal(expected.Config.CustomHeaders, repeatedHeaders.Headers),
						"expected headers %v, got %v", expected.Config.CustomHeaders, repeatedHeaders.Headers)

					repeatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), expected.Namespace, endpointID)
					require.NoError(t, err)
					require.NotNil(t, repeatedSecret)
					require.NotEmpty(t, repeatedSecret.Key)
					require.NotEmpty(t, expected.Config.SigningSecret)
					require.Equal(
						t,
						strings.TrimPrefix(expected.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
						strings.TrimPrefix(repeatedSecret.Key, webhooksecret.SigningSecretPrefix),
					)
				})

				t.Run("Update", func(t *testing.T) {
					t.Run("PartialUpdate", func(t *testing.T) {
						rotatedSecret, err := webhooksecret.NewSigningSecretWithDefaultSize()
						require.NoError(t, err)

						tests := []struct {
							name              string
							input             v4.UpdateChannelInput[v4.WebhookConfigUpdate]
							initiallyDisabled bool
							modifyExpected    func(*v4.CreateChannelInput[v4.WebhookConfig])
						}{
							{
								name: "NoChanges",
							},
							{
								name:  "NameOnly",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Name: new("Renamed")},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Name = "Renamed"
								},
							},
							{
								name:  "URLOnly",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Config: v4.WebhookConfigUpdate{URL: new(receiver.URL + "/partial")}},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Config.URL = receiver.URL + "/partial"
								},
							},
							{
								name:  "HeadersOnly",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string{"x-test-new": "partial"})}},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Config.CustomHeaders = map[string]string{"x-test-new": "partial"}
								},
							},
							{
								name:  "ClearHeadersWithNil",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string)}},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Config.CustomHeaders = nil
								},
							},
							{
								name:  "ClearHeadersWithEmptyMap",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string{})}},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Config.CustomHeaders = map[string]string{}
								},
							},
							{
								name:  "SecretOnly",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Config: v4.WebhookConfigUpdate{SigningSecret: &rotatedSecret}},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Config.SigningSecret = rotatedSecret
								},
							},
							{
								name:  "Disable",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Disabled: new(true)},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Disabled = true
								},
							},
							{
								name:              "Enable",
								initiallyDisabled: true,
								input:             v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Disabled: new(false)},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.Disabled = false
								},
							},
							{
								name:  "EventTypesOnly",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{EventTypes: new(eventTypes[1:])},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.EventTypes = eventTypes[1:]
								},
							},
							{
								name:  "ClearEventTypesWithNil",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{EventTypes: new([]v4.EventType)},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.EventTypes = nil
								},
							},
							{
								name:  "ClearEventTypesWithEmptySlice",
								input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{EventTypes: new([]v4.EventType{})},
								modifyExpected: func(want *v4.CreateChannelInput[v4.WebhookConfig]) {
									want.EventTypes = []v4.EventType{}
								},
							},
						}

						for _, tt := range tests {
							t.Run(tt.name, func(t *testing.T) {
								// Given complete known state, independent of preceding siblings.
								seed := expected
								seed.Disabled = tt.initiallyDisabled

								_, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(seed))
								require.NoError(t, err)

								input := tt.input
								input.Namespace = expected.Namespace
								input.ChannelID = expected.ChannelID
								input.ProviderData = expected.ProviderData

								want := seed
								if tt.modifyExpected != nil {
									tt.modifyExpected(&want)
								}

								// When only the selected fields are supplied.
								updated, err := provider.UpdateChannel(t.Context(), input)
								require.NoError(t, err)

								// Then omitted settings, identity, and provider data survive.
								AssertChannel(t, updated, expectedResult)

								endpoint, err := env.Client.Endpoint().Get(t.Context(), want.Namespace, endpointID)
								require.NoError(t, err)
								AssertChannelWithEndpoint(t, endpoint, want)

								headers, err := env.Client.Endpoint().GetHeaders(t.Context(), want.Namespace, endpointID)
								require.NoError(t, err)
								require.NotNil(t, headers)
								require.True(t, maps.Equal(want.Config.CustomHeaders, headers.Headers))

								secret, err := env.Client.Endpoint().GetSecret(t.Context(), want.Namespace, endpointID)
								require.NoError(t, err)
								require.NotNil(t, secret)
								require.Equal(t,
									strings.TrimPrefix(want.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
									strings.TrimPrefix(secret.Key, webhooksecret.SigningSecretPrefix),
								)
							})
						}
					})

					t.Run("PreserveLegacyRouting", func(t *testing.T) {
						// Given a legacy Rule-ID filter on the existing endpoint.
						_, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(expected))
						require.NoError(t, err)

						_, err = env.Client.Endpoint().Patch(t.Context(), namespaceID, endpointID, svix.EndpointPatch{
							Channels: svixutils.NewNullable([]string{"legacy-rule"}),
						})
						require.NoError(t, err)

						// When only the name changes, then both routing filters survive.
						updated, err := provider.UpdateChannel(t.Context(), v4.UpdateChannelInput[v4.WebhookConfigUpdate]{
							Namespace: namespaceID,
							ChannelID: expected.ChannelID,
							Name:      new("Renamed legacy endpoint"),
						})
						require.NoError(t, err)
						AssertChannel(t, updated, expectedResult)

						endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, endpointID)
						require.NoError(t, err)
						require.NotNil(t, endpoint)
						require.Equal(t, "Renamed legacy endpoint", endpoint.Description)
						require.Equal(t, []string{"legacy-rule"}, endpoint.Channels)
						require.Equal(t, []string{eventTypes[0].String()}, endpoint.EventTypes)

						// When event-type routing is explicitly updated, legacy routing is removed.
						updated, err = provider.UpdateChannel(t.Context(), v4.UpdateChannelInput[v4.WebhookConfigUpdate]{
							Namespace:  namespaceID,
							ChannelID:  expected.ChannelID,
							EventTypes: new(eventTypes[1:]),
						})
						require.NoError(t, err)
						AssertChannel(t, updated, expectedResult)

						want := expected
						want.Name = "Renamed legacy endpoint"
						want.EventTypes = eventTypes[1:]

						endpoint, err = env.Client.Endpoint().Get(t.Context(), namespaceID, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, endpoint, want)
					})

					t.Run("ReplaceConfiguration", func(t *testing.T) {
						// Given the existing channel with its original configuration.
						input := expected
						input.Name = "Updated channel"
						input.Disabled = true
						input.EventTypes = eventTypes[1:]
						input.Config = v4.WebhookConfig{
							URL: receiver.URL + "/updated",
							CustomHeaders: map[string]string{
								"x-test-environment": "updated",
								"x-test-new":         "value",
							},
						}

						// When configuration is replaced without supplying a secret.
						updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(input))
						require.NoError(t, err)

						// Then configuration changes while endpoint identity and the secret are preserved.
						want := input
						want.Config.SigningSecret = initialSecret

						AssertChannel(t, updated, expectedResult)

						updatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    want.Namespace,
							ChannelID:    want.ChannelID,
							ProviderData: want.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, updatedStored, expectedResult)

						updatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), want.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, updatedEndpoint, want)

						updatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), want.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedHeaders)
						require.Empty(t, updatedHeaders.Sensitive)
						require.True(t, maps.Equal(want.Config.CustomHeaders, updatedHeaders.Headers),
							"expected headers %v, got %v", want.Config.CustomHeaders, updatedHeaders.Headers)

						updatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), want.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedSecret)
						require.NotEmpty(t, updatedSecret.Key)
						require.NotEmpty(t, want.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(want.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(updatedSecret.Key, webhooksecret.SigningSecretPrefix),
						)
					})

					t.Run("ClearHeadersWithNil", func(t *testing.T) {
						// Given nonempty headers, independently of earlier siblings.
						seeded, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(expected))
						require.NoError(t, err)
						AssertChannel(t, seeded, expectedResult)

						seededStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    expected.Namespace,
							ChannelID:    expected.ChannelID,
							ProviderData: expected.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, seededStored, expectedResult)

						seededEndpoint, err := env.Client.Endpoint().Get(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, seededEndpoint, expected)

						seededHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, seededHeaders)
						require.Empty(t, seededHeaders.Sensitive)
						require.True(t, maps.Equal(expected.Config.CustomHeaders, seededHeaders.Headers),
							"expected headers %v, got %v", expected.Config.CustomHeaders, seededHeaders.Headers)

						seededSecret, err := env.Client.Endpoint().GetSecret(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, seededSecret)
						require.NotEmpty(t, seededSecret.Key)
						require.NotEmpty(t, expected.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(expected.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(seededSecret.Key, webhooksecret.SigningSecretPrefix),
						)

						input := expected
						input.Config.CustomHeaders = nil

						// When headers are explicitly replaced with a nil map.
						updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(input))
						require.NoError(t, err)

						// Then headers are cleared without changing identity or secret.
						AssertChannel(t, updated, expectedResult)

						updatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    input.Namespace,
							ChannelID:    input.ChannelID,
							ProviderData: input.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, updatedStored, expectedResult)

						updatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, updatedEndpoint, input)

						updatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedHeaders)
						require.Empty(t, updatedHeaders.Sensitive)
						require.True(t, maps.Equal(input.Config.CustomHeaders, updatedHeaders.Headers),
							"expected headers %v, got %v", input.Config.CustomHeaders, updatedHeaders.Headers)

						updatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedSecret)
						require.NotEmpty(t, updatedSecret.Key)
						require.NotEmpty(t, input.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(input.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(updatedSecret.Key, webhooksecret.SigningSecretPrefix),
						)
					})

					t.Run("ClearHeadersWithEmptyMap", func(t *testing.T) {
						// Given nonempty headers, even if the nil-clearing sibling ran.
						seeded, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(expected))
						require.NoError(t, err)
						AssertChannel(t, seeded, expectedResult)

						seededStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    expected.Namespace,
							ChannelID:    expected.ChannelID,
							ProviderData: expected.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, seededStored, expectedResult)

						seededEndpoint, err := env.Client.Endpoint().Get(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, seededEndpoint, expected)

						seededHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, seededHeaders)
						require.Empty(t, seededHeaders.Sensitive)
						require.True(t, maps.Equal(expected.Config.CustomHeaders, seededHeaders.Headers),
							"expected headers %v, got %v", expected.Config.CustomHeaders, seededHeaders.Headers)

						seededSecret, err := env.Client.Endpoint().GetSecret(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, seededSecret)
						require.NotEmpty(t, seededSecret.Key)
						require.NotEmpty(t, expected.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(expected.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(seededSecret.Key, webhooksecret.SigningSecretPrefix),
						)

						input := expected
						input.Config.CustomHeaders = map[string]string{}

						// When headers are explicitly replaced with an empty map.
						updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(input))
						require.NoError(t, err)

						// Then headers are cleared without changing identity or secret.
						AssertChannel(t, updated, expectedResult)

						updatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    input.Namespace,
							ChannelID:    input.ChannelID,
							ProviderData: input.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, updatedStored, expectedResult)

						updatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, updatedEndpoint, input)

						updatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedHeaders)
						require.Empty(t, updatedHeaders.Sensitive)
						require.True(t, maps.Equal(input.Config.CustomHeaders, updatedHeaders.Headers),
							"expected headers %v, got %v", input.Config.CustomHeaders, updatedHeaders.Headers)

						updatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedSecret)
						require.NotEmpty(t, updatedSecret.Key)
						require.NotEmpty(t, input.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(input.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(updatedSecret.Key, webhooksecret.SigningSecretPrefix),
						)
					})

					t.Run("RotateSecret", func(t *testing.T) {
						// Given the original secret on the existing endpoint.
						seeded, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(expected))
						require.NoError(t, err)
						AssertChannel(t, seeded, expectedResult)

						seededStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    expected.Namespace,
							ChannelID:    expected.ChannelID,
							ProviderData: expected.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, seededStored, expectedResult)

						seededEndpoint, err := env.Client.Endpoint().Get(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, seededEndpoint, expected)

						seededHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, seededHeaders)
						require.Empty(t, seededHeaders.Sensitive)
						require.True(t, maps.Equal(expected.Config.CustomHeaders, seededHeaders.Headers),
							"expected headers %v, got %v", expected.Config.CustomHeaders, seededHeaders.Headers)

						seededSecret, err := env.Client.Endpoint().GetSecret(t.Context(), expected.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, seededSecret)
						require.NotEmpty(t, seededSecret.Key)
						require.NotEmpty(t, expected.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(expected.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(seededSecret.Key, webhooksecret.SigningSecretPrefix),
						)

						rotatedSecret, err := webhooksecret.NewSigningSecretWithDefaultSize()
						require.NoError(t, err)
						require.NotEqual(t, initialSecret, rotatedSecret)

						input := expected
						input.Config.SigningSecret = rotatedSecret

						// When a different signing secret is supplied.
						updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(input))
						require.NoError(t, err)

						// Then the secret changes while provider data stays unchanged.
						AssertChannel(t, updated, expectedResult)

						updatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
							Namespace:    input.Namespace,
							ChannelID:    input.ChannelID,
							ProviderData: input.ProviderData,
						})
						require.NoError(t, err)
						AssertChannel(t, updatedStored, expectedResult)

						updatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, updatedEndpoint, input)

						updatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedHeaders)
						require.Empty(t, updatedHeaders.Sensitive)
						require.True(t, maps.Equal(input.Config.CustomHeaders, updatedHeaders.Headers),
							"expected headers %v, got %v", input.Config.CustomHeaders, updatedHeaders.Headers)

						updatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), input.Namespace, endpointID)
						require.NoError(t, err)
						require.NotNil(t, updatedSecret)
						require.NotEmpty(t, updatedSecret.Key)
						require.NotEmpty(t, input.Config.SigningSecret)
						require.Equal(
							t,
							strings.TrimPrefix(input.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
							strings.TrimPrefix(updatedSecret.Key, webhooksecret.SigningSecretPrefix),
						)

						t.Run("RetryWithSameSecret", func(t *testing.T) {
							// Given the already-applied rotation.
							// When the same update is retried.
							repeated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(input))
							require.NoError(t, err)

							// Then the rotated secret and endpoint identity remain stable.
							AssertChannel(t, repeated, expectedResult)

							repeatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
								Namespace:    input.Namespace,
								ChannelID:    input.ChannelID,
								ProviderData: input.ProviderData,
							})
							require.NoError(t, err)
							AssertChannel(t, repeatedStored, expectedResult)

							repeatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), input.Namespace, endpointID)
							require.NoError(t, err)
							AssertChannelWithEndpoint(t, repeatedEndpoint, input)

							repeatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), input.Namespace, endpointID)
							require.NoError(t, err)
							require.NotNil(t, repeatedHeaders)
							require.Empty(t, repeatedHeaders.Sensitive)
							require.True(t, maps.Equal(input.Config.CustomHeaders, repeatedHeaders.Headers),
								"expected headers %v, got %v", input.Config.CustomHeaders, repeatedHeaders.Headers)

							repeatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), input.Namespace, endpointID)
							require.NoError(t, err)
							require.NotNil(t, repeatedSecret)
							require.NotEmpty(t, repeatedSecret.Key)
							require.NotEmpty(t, input.Config.SigningSecret)
							require.Equal(
								t,
								strings.TrimPrefix(input.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
								strings.TrimPrefix(repeatedSecret.Key, webhooksecret.SigningSecretPrefix),
							)
						})

						t.Run("PreserveSecretWhenOmitted", func(t *testing.T) {
							// Given the rotated secret established by the parent.
							withoutSecret := input
							withoutSecret.Config.SigningSecret = ""

							// When a later update omits the secret.
							updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(withoutSecret))
							require.NoError(t, err)

							// Then the rotated secret is retained, not reset or regenerated.
							AssertChannel(t, updated, expectedResult)

							updatedStored, err := provider.GetChannel(t.Context(), v4.GetChannelInput{
								Namespace:    input.Namespace,
								ChannelID:    input.ChannelID,
								ProviderData: input.ProviderData,
							})
							require.NoError(t, err)
							AssertChannel(t, updatedStored, expectedResult)

							updatedEndpoint, err := env.Client.Endpoint().Get(t.Context(), input.Namespace, endpointID)
							require.NoError(t, err)
							AssertChannelWithEndpoint(t, updatedEndpoint, input)

							updatedHeaders, err := env.Client.Endpoint().GetHeaders(t.Context(), input.Namespace, endpointID)
							require.NoError(t, err)
							require.NotNil(t, updatedHeaders)
							require.Empty(t, updatedHeaders.Sensitive)
							require.True(t, maps.Equal(input.Config.CustomHeaders, updatedHeaders.Headers),
								"expected headers %v, got %v", input.Config.CustomHeaders, updatedHeaders.Headers)

							updatedSecret, err := env.Client.Endpoint().GetSecret(t.Context(), input.Namespace, endpointID)
							require.NoError(t, err)
							require.NotNil(t, updatedSecret)
							require.NotEmpty(t, updatedSecret.Key)
							require.NotEmpty(t, input.Config.SigningSecret)
							require.Equal(
								t,
								strings.TrimPrefix(input.Config.SigningSecret, webhooksecret.SigningSecretPrefix),
								strings.TrimPrefix(updatedSecret.Key, webhooksecret.SigningSecretPrefix),
							)
						})
					})
				})

				t.Run("Delete", func(t *testing.T) {
					// Given the endpoint created by the parent, regardless of Update.
					input := v4.DeleteChannelInput{
						Namespace:    namespaceID,
						ChannelID:    expected.ChannelID,
						ProviderData: expected.ProviderData,
					}

					// When the endpoint is deleted.
					err := provider.DeleteChannel(t.Context(), input)
					require.NoError(t, err)

					// Then provider lookup reports absence and returns no provider data.
					deleted, err := provider.GetChannel(t.Context(), input)
					require.Error(t, err)
					require.Nil(t, deleted)

					var providerErr *v4.ProviderError
					require.ErrorAs(t, err, &providerErr)
					require.Equal(t, v4.ProviderErrorCodeResourceNotFound, providerErr.Code)

					t.Run("UpdateMissing", func(t *testing.T) {
						tests := []struct {
							name  string
							input v4.UpdateChannelInput[v4.WebhookConfigUpdate]
						}{
							{name: "NoChanges"},
							{name: "NameOnly", input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Name: new("Must not recreate")}},
							{name: "HeadersOnly", input: v4.UpdateChannelInput[v4.WebhookConfigUpdate]{Config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string{})}}},
						}

						for _, tt := range tests {
							t.Run(tt.name, func(t *testing.T) {
								// Given a deleted endpoint, when any update form is attempted.
								input := tt.input
								input.Namespace = namespaceID
								input.ChannelID = expected.ChannelID

								updated, err := provider.UpdateChannel(t.Context(), input)

								// Then updates never recreate the endpoint or report success.
								require.Nil(t, updated)

								var providerErr *v4.ProviderError
								require.ErrorAs(t, err, &providerErr)
								require.Equal(t, v4.ProviderErrorCodeResourceNotFound, providerErr.Code)
							})
						}
					})

					t.Run("DeleteAgain", func(t *testing.T) {
						// Given the endpoint already deleted by the parent.
						// When deletion is retried, then it still succeeds.
						err := provider.DeleteChannel(t.Context(), input)
						require.NoError(t, err)
					})
				})
			})
		})

		t.Run("Event", func(t *testing.T) {
			t.Parallel()

			// Given an isolated namespace, event type, and webhook receiver.
			namespaceID := ulid.Make().String()

			eventType := v4.EventType{
				Name:    "test." + strings.ToLower(namespaceID) + ".created",
				Version: 1,
			}

			_, err := env.Client.EventType().Create(t.Context(), svix.EventTypeIn{
				Name:        eventType.String(),
				Description: "Event integration test",
			}, nil)
			require.NoError(t, err)

			ctx := t.Context()

			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				defer cancel()

				err := env.Client.EventType().Delete(
					cleanupCtx,
					eventType.String(),
					&svix.EventTypeDeleteOptions{Expunge: new(true)},
				)
				require.NoError(t, err)
			})

			receiver := testutils.NewWebhookServer(t, env.WebhookHost)

			// Remove the application and its endpoints/messages before closing
			// the receiver and removing the event type.
			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				defer cancel()

				err := provider.DeleteNamespace(cleanupCtx, v4.DeleteNamespaceInput{
					ID: namespaceID,
				})
				require.NoError(t, err)
			})

			namespace, err := provider.CreateNamespace(t.Context(), v4.CreateNamespaceInput{
				ProviderNamespace: v4.ProviderNamespace{
					ID:   namespaceID,
					Name: "Event integration test",
				},
			})
			require.NoError(t, err)
			require.NotNil(t, namespace)

			signingSecret, err := webhooksecret.NewSigningSecretWithDefaultSize()
			require.NoError(t, err)

			verifier, err := svix.NewWebhook(signingSecret)
			require.NoError(t, err)

			channelID := ulid.Make().String()
			channelURL := receiver.URL + "/events"

			channel, err := provider.CreateChannel(t.Context(), v4.CreateChannelInput[v4.WebhookConfig]{
				Namespace: namespaceID,
				ChannelID: channelID,
				Name:      "Event integration test",
				Config: v4.WebhookConfig{
					URL:           channelURL,
					SigningSecret: signingSecret,
				},
				EventTypes: []v4.EventType{eventType},
			})
			require.NoError(t, err)
			require.NotNil(t, channel)
			require.Equal(t, v4.ChannelOperationStateInSync, channel.State)
			require.NotEmpty(t, channel.ProviderData["svix.endpoint_id"])

			// Observation never sends or resends; it waits for provider-reported
			// success and at least one visible attempt.
			waitForDelivery := func(t *testing.T, input v4.GetEventInput) *v4.GetEventResult {
				t.Helper()

				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()

				var observed *v4.GetEventResult

				require.EventuallyWithT(t, func(c *assert.CollectT) {
					result, err := provider.GetEvent(ctx, input)
					require.NoError(c, err)
					require.NotNil(c, result)
					require.Len(c, result.Destinations, len(input.Destinations))

					remaining := make(map[string]struct{}, len(input.Destinations))

					for _, destination := range input.Destinations {
						remaining[destination.ChannelID] = struct{}{}
					}

					for _, destination := range result.Destinations {
						require.Contains(c, remaining, destination.ChannelID)
						delete(remaining, destination.ChannelID)

						require.NoError(c, destination.Error)

						attempts, err := json.Marshal(destination.Attempts)
						require.NoError(c, err)
						require.Equal(c, v4.EventDeliveryStatusStateSuccess, destination.State,
							"delivery attempts: %s", attempts)
						require.NotEmpty(c, destination.Attempts)
					}

					require.Empty(c, remaining)

					observed = result
				}, 30*time.Second, 200*time.Millisecond)

				return observed
			}

			t.Run("Send", func(t *testing.T) {
				// Given a system event without a rule snapshot.
				event := v4.Event{
					NamespacedID: models.NamespacedID{
						Namespace: namespaceID,
						ID:        ulid.Make().String(),
					},
					EventType: eventType,
					CreatedAt: time.Now().UTC(),
					Payload:   json.RawMessage(`{"invoice_id":"invoice-1","amount":42}`),
					Annotations: models.Annotations{
						"test": true,
					},
				}

				expectedBody, err := json.Marshal(map[string]any{
					"id":          event.ID,
					"event_type":  event.EventType.String(),
					"created_at":  event.CreatedAt,
					"payload":     event.Payload,
					"annotations": event.Annotations,
				})
				require.NoError(t, err)

				// When the event is submitted for delivery.
				sendInput := v4.SendEventInput{
					Event: event,
					Destinations: []v4.EventDestination{
						{
							ChannelID:           channelID,
							ChannelProviderData: channel.ProviderData,
						},
					},
					IdempotencyKey: new(ulid.Make().String()),
				}

				sent, err := provider.SendEvent(t.Context(), sendInput)
				require.NoError(t, err)
				require.NotNil(t, sent)

				// Then acceptance returns message and delivery-status provider data.
				messageID := sent.EventProviderData["svix.message_id"]
				require.NotEmpty(t, messageID)
				require.NotEqual(t, event.ID, messageID)

				require.Len(t, sent.Destinations, 1)
				require.Equal(t, channelID, sent.Destinations[0].ChannelID)
				require.NoError(t, sent.Destinations[0].Error)
				require.Equal(t, v4.ProviderData{
					"svix.dispatch_generation": "1",
				}, sent.Destinations[0].EventDeliveryStatusProviderData)

				event.ProviderData = sent.EventProviderData

				getInput := v4.GetEventInput{
					Event: event,
					Destinations: []v4.GetEventDestination{
						{
							ChannelID:                       channelID,
							ChannelProviderData:             channel.ProviderData,
							EventDeliveryStatusProviderData: sent.Destinations[0].EventDeliveryStatusProviderData,
							DispatchGeneration:              1,
						},
					},
				}

				t.Run("Get", func(t *testing.T) {
					// Given the event accepted by the parent.
					// When its asynchronous delivery is observed.
					observed := waitForDelivery(t, getInput)

					// Then message identity and delivery-status provider data are retained.
					require.Equal(t, sent.EventProviderData, observed.EventProviderData)

					destination := observed.Destinations[0]
					require.Nil(t, destination.NextObservationAt)
					require.Subset(t,
						destination.EventDeliveryStatusProviderData,
						getInput.Destinations[0].EventDeliveryStatusProviderData,
					)

					attemptIDs := make(map[string]struct{})
					successfulAttempts := 0

					for _, attempt := range destination.Attempts {
						attemptID := attempt.ProviderData["svix.attempt_id"]
						require.NotEmpty(t, attemptID)
						require.NotContains(t, attemptIDs, attemptID)
						require.False(t, attempt.Timestamp.IsZero())

						attemptIDs[attemptID] = struct{}{}

						var response struct {
							StatusCode        int    `json:"status_code"`
							Body              string `json:"body"`
							DurationMs        int64  `json:"duration_ms"`
							URL               string `json:"url"`
							ResponseTruncated bool   `json:"response_truncated"`
						}

						require.NoError(t, json.Unmarshal(attempt.Response, &response))
						require.Equal(t, channelURL, response.URL)
						require.GreaterOrEqual(t, response.DurationMs, int64(0))
						require.False(t, response.ResponseTruncated)

						if attempt.State == v4.EventDeliveryStatusStateSuccess {
							successfulAttempts++

							require.Equal(t, http.StatusNoContent, response.StatusCode)
							require.Empty(t, response.Body)
						}
					}

					require.Positive(t, successfulAttempts)

					var observedIDs []string
					require.NoError(t, json.Unmarshal([]byte(destination.EventDeliveryStatusProviderData["svix.observed_attempts"]), &observedIDs))
					require.ElementsMatch(t, slices.Collect(maps.Keys(attemptIDs)), observedIDs)
					require.Equal(t, v4.ProviderData{"svix.dispatch_generation": "1"}, getInput.Destinations[0].EventDeliveryStatusProviderData)

					// The independently captured webhook verifies the actual wire envelope.
					requests := receiver.Requests()
					require.NotEmpty(t, requests)

					for _, request := range requests {
						require.JSONEq(t, string(expectedBody), string(request.Body))
						require.NoError(t, verifier.Verify(request.Body, request.Header))
						require.Equal(t, messageID, request.Header.Get("svix-id"))
					}
				})

				t.Run("RetryWithSameIdempotencyKey", func(t *testing.T) {
					// Given a completed initial delivery.
					before := waitForDelivery(t, getInput)
					requestCount := len(receiver.Requests())

					// When the exact send request is retried.
					repeated, err := provider.SendEvent(t.Context(), sendInput)
					require.NoError(t, err)
					require.Equal(t, sent, repeated)

					// Then the message is reused without another delivery during observation.
					require.Never(t, func() bool {
						return len(receiver.Requests()) > requestCount
					}, time.Second, 100*time.Millisecond)

					after := waitForDelivery(t, getInput)
					require.Equal(t, before.EventProviderData, after.EventProviderData)
					require.ElementsMatch(t, before.Destinations[0].Attempts, after.Destinations[0].Attempts)
				})

				t.Run("RecoverExistingWithoutIdempotencyKey", func(t *testing.T) {
					// Given a completed event and a retry without its original caller key.
					before := waitForDelivery(t, getInput)
					requestCount := len(receiver.Requests())

					input := sendInput
					input.IdempotencyKey = nil

					// When creation recovers the existing message by Event ID.
					repeated, err := provider.SendEvent(t.Context(), input)
					require.NoError(t, err)
					require.Equal(t, sent, repeated)

					// Then external identity prevents another delivery during observation.
					require.Never(t, func() bool {
						return len(receiver.Requests()) > requestCount
					}, time.Second, 100*time.Millisecond)

					after := waitForDelivery(t, getInput)
					require.Equal(t, before.EventProviderData, after.EventProviderData)
					require.ElementsMatch(t, before.Destinations[0].Attempts, after.Destinations[0].Attempts)
				})

				t.Run("Resend", func(t *testing.T) {
					// Given an observed initial delivery, independently of the Get sibling.
					before := waitForDelivery(t, getInput)

					previousAttemptIDs := make([]string, 0, len(before.Destinations[0].Attempts))

					for _, attempt := range before.Destinations[0].Attempts {
						attemptID := attempt.ProviderData["svix.attempt_id"]
						require.NotEmpty(t, attemptID)

						previousAttemptIDs = append(previousAttemptIDs, attemptID)
					}

					previousRequestCount := len(receiver.Requests())
					require.Positive(t, previousRequestCount)

					destination := getInput.Destinations[0]
					destination.DispatchGeneration = 2
					destination.EventDeliveryStatusProviderData = before.Destinations[0].EventDeliveryStatusProviderData

					// When the same event is explicitly resent to the same channel.
					resendInput := v4.ResendEventInput{
						Event:          event,
						Destinations:   []v4.GetEventDestination{destination},
						IdempotencyKey: new(ulid.Make().String()),
					}
					originalData := maps.Clone(destination.EventDeliveryStatusProviderData)

					resent, err := provider.ResendEvent(t.Context(), resendInput)
					require.NoError(t, err)
					require.NotNil(t, resent)
					require.Equal(t, originalData, resendInput.Destinations[0].EventDeliveryStatusProviderData)

					// Then resend retains the message and records the new generation's baseline.
					require.Equal(t, sent.EventProviderData, resent.EventProviderData)
					require.Len(t, resent.Destinations, 1)
					require.Equal(t, channelID, resent.Destinations[0].ChannelID)
					require.NoError(t, resent.Destinations[0].Error)

					resendData := resent.Destinations[0].EventDeliveryStatusProviderData
					require.Equal(t, "2", resendData["svix.dispatch_generation"])

					var baseline []string
					require.NoError(t, json.Unmarshal([]byte(resendData["svix.resend_baseline"]), &baseline))
					require.ElementsMatch(t, previousAttemptIDs, baseline)

					destination.EventDeliveryStatusProviderData = resendData

					after := waitForDelivery(t, v4.GetEventInput{
						Event:        event,
						Destinations: []v4.GetEventDestination{destination},
					})

					require.Equal(t, sent.EventProviderData, after.EventProviderData)
					require.Subset(t, after.Destinations[0].EventDeliveryStatusProviderData, resendData)
					require.NotContains(t, resendData, "svix.observed_attempts")
					require.Nil(t, after.Destinations[0].NextObservationAt)

					// Existing history remains, and an old success cannot satisfy the resend.
					observedAttemptIDs := make(map[string]struct{})
					newSuccessfulAttempts := 0

					for _, attempt := range after.Destinations[0].Attempts {
						attemptID := attempt.ProviderData["svix.attempt_id"]
						require.NotEmpty(t, attemptID)
						require.NotContains(t, observedAttemptIDs, attemptID)

						observedAttemptIDs[attemptID] = struct{}{}

						if !slices.Contains(baseline, attemptID) &&
							attempt.State == v4.EventDeliveryStatusStateSuccess {
							newSuccessfulAttempts++
						}
					}

					for _, attemptID := range previousAttemptIDs {
						require.Contains(t, observedAttemptIDs, attemptID)
					}

					require.Positive(t, newSuccessfulAttempts)

					var observedIDs []string
					require.NoError(t, json.Unmarshal([]byte(after.Destinations[0].EventDeliveryStatusProviderData["svix.observed_attempts"]), &observedIDs))
					require.ElementsMatch(t, slices.Collect(maps.Keys(observedAttemptIDs)), observedIDs)

					// Resend delivers the unchanged envelope with the same message identity.
					requests := receiver.Requests()
					require.Greater(t, len(requests), previousRequestCount)

					for _, request := range requests[previousRequestCount:] {
						require.JSONEq(t, string(expectedBody), string(request.Body))
						require.NoError(t, verifier.Verify(request.Body, request.Header))
						require.Equal(t, messageID, request.Header.Get("svix-id"))
					}

					t.Run("RetryWithSameIdempotencyKey", func(t *testing.T) {
						// Given a completed resend and its original request.
						requestCount := len(receiver.Requests())

						// When the same logical resend is retried after delivery.
						repeated, err := provider.ResendEvent(t.Context(), resendInput)
						require.NoError(t, err)
						require.NotNil(t, repeated)
						require.Equal(t, resent.EventProviderData, repeated.EventProviderData)
						require.Len(t, repeated.Destinations, 1)
						require.Equal(t, channelID, repeated.Destinations[0].ChannelID)
						require.NoError(t, repeated.Destinations[0].Error)

						// Then the retry creates no additional delivery during observation.
						require.Never(t, func() bool {
							return len(receiver.Requests()) > requestCount
						}, time.Second, 100*time.Millisecond)

						repeatedData := repeated.Destinations[0].EventDeliveryStatusProviderData
						require.Equal(t, "2", repeatedData["svix.dispatch_generation"])

						var originalBaseline, repeatedBaseline []string

						require.NoError(t, json.Unmarshal([]byte(resendData["svix.resend_baseline"]), &originalBaseline))
						require.NoError(t, json.Unmarshal([]byte(repeatedData["svix.resend_baseline"]), &repeatedBaseline))

						// Retrying must not absorb the completed resend into its own baseline.
						require.ElementsMatch(t, originalBaseline, repeatedBaseline)

						retryDestination := destination
						retryDestination.EventDeliveryStatusProviderData = repeatedData

						reobserved := waitForDelivery(t, v4.GetEventInput{
							Event:        event,
							Destinations: []v4.GetEventDestination{retryDestination},
						})

						require.ElementsMatch(t, after.Destinations[0].Attempts, reobserved.Destinations[0].Attempts)
					})

					t.Run("RecoverLostResendResponse", func(t *testing.T) {
						// Given only the previous generation's persisted observation survives.
						freshProvider, err := New(Config{
							Client: env.Client,
							Logger: env.Logger,
							Tracer: env.Tracer,
						})
						require.NoError(t, err)

						requestCount := len(receiver.Requests())

						// When a fresh provider observes the new generation without the resend result.
						recovered, err := freshProvider.GetEvent(t.Context(), v4.GetEventInput{
							Event:        event,
							Destinations: resendInput.Destinations,
						})
						require.NoError(t, err)
						require.NotNil(t, recovered)
						require.Len(t, recovered.Destinations, 1)

						// Then observation reconstructs the baseline without dispatching again.
						outcome := recovered.Destinations[0]
						require.NoError(t, outcome.Error)
						require.Equal(t, v4.EventDeliveryStatusStateSuccess, outcome.State)
						require.Equal(t, after.Destinations[0].EventDeliveryStatusProviderData, outcome.EventDeliveryStatusProviderData)
						require.ElementsMatch(t, after.Destinations[0].Attempts, outcome.Attempts)
						require.Equal(t, originalData, resendInput.Destinations[0].EventDeliveryStatusProviderData)
						require.Never(t, func() bool {
							return len(receiver.Requests()) > requestCount
						}, time.Second, 100*time.Millisecond)
					})

					t.Run("RetryWithCurrentGenerationObservation", func(t *testing.T) {
						// Given reconciliation persisted the successful resend observation.
						current := destination
						current.EventDeliveryStatusProviderData = after.Destinations[0].EventDeliveryStatusProviderData
						original := maps.Clone(current.EventDeliveryStatusProviderData)
						requestCount := len(receiver.Requests())

						// When retrying the same generation with its original key.
						repeated, err := provider.ResendEvent(t.Context(), v4.ResendEventInput{
							Event:          event,
							Destinations:   []v4.GetEventDestination{current},
							IdempotencyKey: resendInput.IdempotencyKey,
						})
						require.NoError(t, err)
						require.NotNil(t, repeated)
						require.Len(t, repeated.Destinations, 1)
						require.NoError(t, repeated.Destinations[0].Error)

						// Then the baseline excludes the completed resend, despite the newer observation.
						require.Equal(t, resendData, repeated.Destinations[0].EventDeliveryStatusProviderData)
						require.Equal(t, original, current.EventDeliveryStatusProviderData)
						require.Never(t, func() bool {
							return len(receiver.Requests()) > requestCount
						}, time.Second, 100*time.Millisecond)

						reobserved := waitForDelivery(t, v4.GetEventInput{
							Event:        event,
							Destinations: []v4.GetEventDestination{current},
						})
						require.Equal(t, original, reobserved.Destinations[0].EventDeliveryStatusProviderData)
					})

					t.Run("NextGeneration", func(t *testing.T) {
						// Given the latest persisted observation and a requested third generation.
						next := destination
						next.DispatchGeneration = 3
						next.EventDeliveryStatusProviderData = after.Destinations[0].EventDeliveryStatusProviderData
						original := maps.Clone(next.EventDeliveryStatusProviderData)

						// When observation runs before any new attempt exists.
						pending, err := provider.GetEvent(t.Context(), v4.GetEventInput{
							Event:        event,
							Destinations: []v4.GetEventDestination{next},
						})
						require.NoError(t, err)
						require.NotNil(t, pending)
						require.Len(t, pending.Destinations, 1)
						require.NoError(t, pending.Destinations[0].Error)

						// Then an old success cannot complete the new generation.
						require.Equal(t, v4.EventDeliveryStatusStateSending, pending.Destinations[0].State)
						require.Equal(t, "3", pending.Destinations[0].EventDeliveryStatusProviderData["svix.dispatch_generation"])
						require.JSONEq(t, original["svix.observed_attempts"], pending.Destinations[0].EventDeliveryStatusProviderData["svix.resend_baseline"])
						require.Equal(t, original, next.EventDeliveryStatusProviderData)

						// When the third generation is explicitly dispatched with a new key.
						resent, err := provider.ResendEvent(t.Context(), v4.ResendEventInput{
							Event:          event,
							Destinations:   []v4.GetEventDestination{next},
							IdempotencyKey: new(ulid.Make().String()),
						})
						require.NoError(t, err)
						require.NotNil(t, resent)
						require.Len(t, resent.Destinations, 1)
						require.NoError(t, resent.Destinations[0].Error)
						require.Equal(t, "3", resent.Destinations[0].EventDeliveryStatusProviderData["svix.dispatch_generation"])
						require.JSONEq(t, original["svix.observed_attempts"], resent.Destinations[0].EventDeliveryStatusProviderData["svix.resend_baseline"])

						next.EventDeliveryStatusProviderData = resent.Destinations[0].EventDeliveryStatusProviderData
						observed := waitForDelivery(t, v4.GetEventInput{
							Event:        event,
							Destinations: []v4.GetEventDestination{next},
						})

						// Then all prior attempts remain history and a new attempt completes this resend.
						require.Greater(t, len(observed.Destinations[0].Attempts), len(after.Destinations[0].Attempts))
						require.Subset(t, observed.Destinations[0].Attempts, after.Destinations[0].Attempts)
						require.Subset(t, observed.Destinations[0].EventDeliveryStatusProviderData, next.EventDeliveryStatusProviderData)
					})
				})
			})

			t.Run("FanOut", func(t *testing.T) {
				// Given two endpoints subscribed to the same event type.
				secondReceiver := testutils.NewWebhookServer(t, env.WebhookHost)
				secondChannelID := ulid.Make().String()

				ctx := t.Context()

				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
					defer cancel()

					err := provider.DeleteChannel(cleanupCtx, v4.DeleteChannelInput{
						Namespace: namespaceID,
						ChannelID: secondChannelID,
					})
					require.NoError(t, err)
				})

				secondChannel, err := provider.CreateChannel(t.Context(), v4.CreateChannelInput[v4.WebhookConfig]{
					Namespace: namespaceID,
					ChannelID: secondChannelID,
					Name:      "Second event receiver",
					Config: v4.WebhookConfig{
						URL:           secondReceiver.URL + "/events",
						SigningSecret: signingSecret,
					},
					EventTypes: []v4.EventType{eventType},
				})
				require.NoError(t, err)
				require.NotNil(t, secondChannel)
				require.Equal(t, v4.ChannelOperationStateInSync, secondChannel.State)
				require.NotEmpty(t, secondChannel.ProviderData["svix.endpoint_id"])

				event := v4.Event{
					NamespacedID: models.NamespacedID{
						Namespace: namespaceID,
						ID:        ulid.Make().String(),
					},
					EventType: eventType,
					CreatedAt: time.Now().UTC(),
					Payload:   json.RawMessage(`{"invoice_id":"fan-out","amount":42}`),
				}

				// When one message is submitted with two tracked destinations.
				sent, err := provider.SendEvent(t.Context(), v4.SendEventInput{
					Event: event,
					Destinations: []v4.EventDestination{
						{
							ChannelID:           channelID,
							ChannelProviderData: channel.ProviderData,
						},
						{
							ChannelID:           secondChannelID,
							ChannelProviderData: secondChannel.ProviderData,
						},
					},
					IdempotencyKey: new(ulid.Make().String()),
				})
				require.NoError(t, err)
				require.NotNil(t, sent)
				require.Len(t, sent.Destinations, 2)

				messageID := sent.EventProviderData["svix.message_id"]
				require.NotEmpty(t, messageID)

				channelData := map[string]v4.ProviderData{
					channelID:       channel.ProviderData,
					secondChannelID: secondChannel.ProviderData,
				}

				event.ProviderData = sent.EventProviderData

				getInput := v4.GetEventInput{Event: event}
				accepted := make(map[string]struct{})

				for _, destination := range sent.Destinations {
					require.Contains(t, channelData, destination.ChannelID)
					require.NotContains(t, accepted, destination.ChannelID)
					require.NoError(t, destination.Error)
					require.Equal(t, v4.ProviderData{
						"svix.dispatch_generation": "1",
					}, destination.EventDeliveryStatusProviderData)

					accepted[destination.ChannelID] = struct{}{}

					getInput.Destinations = append(getInput.Destinations, v4.GetEventDestination{
						ChannelID:                       destination.ChannelID,
						ChannelProviderData:             channelData[destination.ChannelID],
						EventDeliveryStatusProviderData: destination.EventDeliveryStatusProviderData,
						DispatchGeneration:              1,
					})
				}

				// Then both endpoints receive the same message, with independent attempts.
				observed := waitForDelivery(t, getInput)
				require.Equal(t, sent.EventProviderData, observed.EventProviderData)

				deliveries := make(map[string]v4.GetEventDestinationResult)
				attemptIDs := make(map[string]struct{})

				for _, destination := range observed.Destinations {
					deliveries[destination.ChannelID] = destination

					for _, attempt := range destination.Attempts {
						attemptID := attempt.ProviderData["svix.attempt_id"]
						require.NotEmpty(t, attemptID)
						require.NotContains(t, attemptIDs, attemptID)

						attemptIDs[attemptID] = struct{}{}
					}
				}

				var deliveredBody []byte

				for _, webhook := range []*testutils.WebhookServer{receiver, secondReceiver} {
					matchingRequests := 0

					for _, request := range webhook.Requests() {
						// The first receiver also contains the preceding lifecycle's event.
						if request.Header.Get("svix-id") != messageID {
							continue
						}

						matchingRequests++

						require.NoError(t, verifier.Verify(request.Body, request.Header))

						var envelope struct {
							ID        string          `json:"id"`
							EventType string          `json:"event_type"`
							Payload   json.RawMessage `json:"payload"`
						}

						require.NoError(t, json.Unmarshal(request.Body, &envelope))
						require.Equal(t, event.ID, envelope.ID)
						require.Equal(t, eventType.String(), envelope.EventType)
						require.JSONEq(t, string(event.Payload), string(envelope.Payload))

						if deliveredBody == nil {
							deliveredBody = request.Body
						} else {
							require.JSONEq(t, string(deliveredBody), string(request.Body))
						}
					}

					require.Positive(t, matchingRequests)
				}

				t.Run("GetWithMissingDestination", func(t *testing.T) {
					// Given two completed deliveries and an unknown channel.
					missingID := ulid.Make().String()

					input := v4.GetEventInput{
						Event: event,
						Destinations: append([]v4.GetEventDestination{
							{
								ChannelID:          missingID,
								DispatchGeneration: 1,
							},
						}, getInput.Destinations...),
					}

					// When all destinations are observed together.
					result, err := provider.GetEvent(t.Context(), input)
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Len(t, result.Destinations, 3)
					require.Equal(t, sent.EventProviderData, result.EventProviderData)

					// Then the missing destination does not discard successful observations.
					outcomes := make(map[string]v4.GetEventDestinationResult)

					for _, destination := range result.Destinations {
						require.NotContains(t, outcomes, destination.ChannelID)
						outcomes[destination.ChannelID] = destination
					}

					require.Contains(t, outcomes, missingID)

					missing := outcomes[missingID]
					require.Equal(t, v4.EventDeliveryStatusStateSending, missing.State)
					require.Empty(t, missing.Attempts)

					var providerErr *v4.ProviderError
					require.ErrorAs(t, missing.Error, &providerErr)
					require.Equal(t, v4.ProviderErrorKindRetryable, providerErr.Kind)
					require.Equal(t, v4.ProviderErrorCodeUnavailable, providerErr.Code)

					for _, id := range []string{channelID, secondChannelID} {
						require.Contains(t, outcomes, id)
						require.NoError(t, outcomes[id].Error)
						require.Equal(t, v4.EventDeliveryStatusStateSuccess, outcomes[id].State)
						require.ElementsMatch(t, deliveries[id].Attempts, outcomes[id].Attempts)
					}
				})

				t.Run("ResendWithMissingDestination", func(t *testing.T) {
					// Given completed fan-out, independently of the Get sibling.
					before := waitForDelivery(t, getInput)

					beforeByChannel := make(map[string]v4.GetEventDestinationResult)

					for _, destination := range before.Destinations {
						beforeByChannel[destination.ChannelID] = destination
					}

					missingID := ulid.Make().String()
					controlRequestCount := len(secondReceiver.Requests())

					target := v4.GetEventDestination{
						ChannelID:                       channelID,
						ChannelProviderData:             channel.ProviderData,
						EventDeliveryStatusProviderData: beforeByChannel[channelID].EventDeliveryStatusProviderData,
						DispatchGeneration:              2,
					}

					// When the missing destination is processed before the valid target.
					result, err := provider.ResendEvent(t.Context(), v4.ResendEventInput{
						Event: event,
						Destinations: []v4.GetEventDestination{
							{
								ChannelID:          missingID,
								DispatchGeneration: 2,
							},
							target,
						},
						IdempotencyKey: new(ulid.Make().String()),
					})
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Len(t, result.Destinations, 2)
					require.Equal(t, sent.EventProviderData, result.EventProviderData)

					outcomes := make(map[string]v4.ResendEventDestinationResult)

					for _, destination := range result.Destinations {
						require.NotContains(t, outcomes, destination.ChannelID)
						outcomes[destination.ChannelID] = destination
					}

					// Then the missing channel fails, but the valid target is accepted.
					require.Contains(t, outcomes, missingID)
					require.Contains(t, outcomes, channelID)

					var providerErr *v4.ProviderError
					require.ErrorAs(t, outcomes[missingID].Error, &providerErr)
					require.Equal(t, v4.ProviderErrorKindTerminal, providerErr.Kind)
					require.Equal(t, v4.ProviderErrorCodeResourceNotFound, providerErr.Code)
					require.Empty(t, outcomes[missingID].EventDeliveryStatusProviderData)

					require.NoError(t, outcomes[channelID].Error)

					target.EventDeliveryStatusProviderData = outcomes[channelID].EventDeliveryStatusProviderData

					require.Equal(t, "2", target.EventDeliveryStatusProviderData["svix.dispatch_generation"])

					after := waitForDelivery(t, v4.GetEventInput{
						Event:        event,
						Destinations: []v4.GetEventDestination{target},
					})

					oldAttemptIDs := make(map[string]struct{})

					for _, attempt := range beforeByChannel[channelID].Attempts {
						oldAttemptIDs[attempt.ProviderData["svix.attempt_id"]] = struct{}{}
					}

					newSuccessfulAttempts := 0

					for _, attempt := range after.Destinations[0].Attempts {
						id := attempt.ProviderData["svix.attempt_id"]
						require.NotEmpty(t, id)

						if _, existed := oldAttemptIDs[id]; !existed &&
							attempt.State == v4.EventDeliveryStatusStateSuccess {
							newSuccessfulAttempts++
						}
					}

					require.Positive(t, newSuccessfulAttempts)

					// A targeted resend must not fan out to the second subscribed endpoint.
					require.Never(t, func() bool {
						return len(secondReceiver.Requests()) > controlRequestCount
					}, time.Second, 100*time.Millisecond)

					control := waitForDelivery(t, v4.GetEventInput{
						Event: event,
						Destinations: []v4.GetEventDestination{
							{
								ChannelID:                       secondChannelID,
								ChannelProviderData:             secondChannel.ProviderData,
								EventDeliveryStatusProviderData: beforeByChannel[secondChannelID].EventDeliveryStatusProviderData,
								DispatchGeneration:              1,
							},
						},
					})

					require.ElementsMatch(t, beforeByChannel[secondChannelID].Attempts, control.Destinations[0].Attempts)
				})

				t.Run("ResendToBoth", func(t *testing.T) {
					// Given a fresh event delivered to the existing two channels.
					resendEvent := event
					resendEvent.ID = ulid.Make().String()
					resendEvent.CreatedAt = time.Now().UTC()
					resendEvent.ProviderData = nil

					sent, err := provider.SendEvent(t.Context(), v4.SendEventInput{
						Event: resendEvent,
						Destinations: []v4.EventDestination{
							{
								ChannelID:           channelID,
								ChannelProviderData: channelData[channelID],
							},
							{
								ChannelID:           secondChannelID,
								ChannelProviderData: channelData[secondChannelID],
							},
						},
						IdempotencyKey: new(ulid.Make().String()),
					})
					require.NoError(t, err)
					require.NotNil(t, sent)
					require.Len(t, sent.Destinations, 2)

					resendEvent.ProviderData = sent.EventProviderData

					messageID := sent.EventProviderData["svix.message_id"]
					require.NotEmpty(t, messageID)

					getInput := v4.GetEventInput{Event: resendEvent}
					accepted := make(map[string]struct{})

					for _, destination := range sent.Destinations {
						require.Contains(t, channelData, destination.ChannelID)
						require.NotContains(t, accepted, destination.ChannelID)
						require.NoError(t, destination.Error)

						accepted[destination.ChannelID] = struct{}{}

						getInput.Destinations = append(getInput.Destinations, v4.GetEventDestination{
							ChannelID:                       destination.ChannelID,
							ChannelProviderData:             channelData[destination.ChannelID],
							EventDeliveryStatusProviderData: destination.EventDeliveryStatusProviderData,
							DispatchGeneration:              1,
						})
					}

					before := waitForDelivery(t, getInput)
					beforeByChannel := make(map[string]v4.GetEventDestinationResult)

					for _, destination := range before.Destinations {
						beforeByChannel[destination.ChannelID] = destination
					}

					receivers := map[string]*testutils.WebhookServer{
						channelID:       receiver,
						secondChannelID: secondReceiver,
					}

					// Receivers also contain other scenarios' events; scope wire assertions
					// to this message rather than counting unrelated requests.
					messageRequests := func(webhook *testutils.WebhookServer) []testutils.WebhookRequest {
						var matching []testutils.WebhookRequest

						for _, request := range webhook.Requests() {
							if request.Header.Get("svix-id") == messageID {
								matching = append(matching, request)
							}
						}

						return matching
					}

					beforeRequests := make(map[string][]testutils.WebhookRequest)

					for id, webhook := range receivers {
						beforeRequests[id] = messageRequests(webhook)
						require.NotEmpty(t, beforeRequests[id])
					}

					resendInput := v4.ResendEventInput{
						Event:          resendEvent,
						IdempotencyKey: new(ulid.Make().String()),
					}

					for _, destination := range getInput.Destinations {
						destination.DispatchGeneration = 2
						destination.EventDeliveryStatusProviderData = beforeByChannel[destination.ChannelID].EventDeliveryStatusProviderData

						resendInput.Destinations = append(resendInput.Destinations, destination)
					}

					// When both destinations are resent using one caller-supplied key.
					resent, err := provider.ResendEvent(t.Context(), resendInput)
					require.NoError(t, err)
					require.NotNil(t, resent)
					require.Equal(t, sent.EventProviderData, resent.EventProviderData)
					require.Len(t, resent.Destinations, 2)

					resendData := make(map[string]v4.ProviderData)

					for _, destination := range resent.Destinations {
						require.Contains(t, beforeByChannel, destination.ChannelID)
						require.NotContains(t, resendData, destination.ChannelID)
						require.NoError(t, destination.Error)

						data := destination.EventDeliveryStatusProviderData
						require.Equal(t, "2", data["svix.dispatch_generation"])
						require.JSONEq(t,
							beforeByChannel[destination.ChannelID].EventDeliveryStatusProviderData["svix.observed_attempts"],
							data["svix.resend_baseline"],
						)

						resendData[destination.ChannelID] = data
					}

					observeInput := v4.GetEventInput{Event: resendEvent}

					for _, destination := range resendInput.Destinations {
						destination.EventDeliveryStatusProviderData = resendData[destination.ChannelID]
						observeInput.Destinations = append(observeInput.Destinations, destination)
					}

					after := waitForDelivery(t, observeInput)
					afterByChannel := make(map[string]v4.GetEventDestinationResult)

					// Then both channels have a new successful attempt outside their baseline.
					for _, destination := range after.Destinations {
						id := destination.ChannelID
						afterByChannel[id] = destination

						require.Subset(t, destination.Attempts, beforeByChannel[id].Attempts)
						require.Subset(t, destination.EventDeliveryStatusProviderData, resendData[id])

						var baseline []string
						require.NoError(t, json.Unmarshal([]byte(resendData[id]["svix.resend_baseline"]), &baseline))

						newSuccessfulAttempts := 0

						for _, attempt := range destination.Attempts {
							attemptID := attempt.ProviderData["svix.attempt_id"]
							require.NotEmpty(t, attemptID)

							if !slices.Contains(baseline, attemptID) &&
								attempt.State == v4.EventDeliveryStatusStateSuccess {
								newSuccessfulAttempts++
							}
						}

						require.Positive(t, newSuccessfulAttempts)

						requests := messageRequests(receivers[id])
						require.Greater(t, len(requests), len(beforeRequests[id]))

						for _, request := range requests {
							require.NoError(t, verifier.Verify(request.Body, request.Header))
							require.JSONEq(t, string(beforeRequests[id][0].Body), string(request.Body))
						}
					}

					t.Run("RetryWithSameIdempotencyKey", func(t *testing.T) {
						// Given both resends completed and the original input was retained.
						requestCounts := make(map[string]int)

						for id, webhook := range receivers {
							requestCounts[id] = len(messageRequests(webhook))
						}

						// When the exact request is retried with the same single key.
						repeated, err := provider.ResendEvent(t.Context(), resendInput)
						require.NoError(t, err)
						require.NotNil(t, repeated)
						require.Equal(t, resent.EventProviderData, repeated.EventProviderData)
						require.Len(t, repeated.Destinations, 2)

						repeatedData := make(map[string]v4.ProviderData)

						for _, destination := range repeated.Destinations {
							require.Contains(t, resendData, destination.ChannelID)
							require.NotContains(t, repeatedData, destination.ChannelID)
							require.NoError(t, destination.Error)
							require.Equal(t, resendData[destination.ChannelID], destination.EventDeliveryStatusProviderData)

							repeatedData[destination.ChannelID] = destination.EventDeliveryStatusProviderData
						}

						// Then neither receiver gets another delivery during observation.
						require.Never(t, func() bool {
							for id, webhook := range receivers {
								if len(messageRequests(webhook)) > requestCounts[id] {
									return true
								}
							}

							return false
						}, time.Second, 100*time.Millisecond)

						retryObservation := v4.GetEventInput{Event: resendEvent}

						for _, destination := range resendInput.Destinations {
							destination.EventDeliveryStatusProviderData = repeatedData[destination.ChannelID]
							retryObservation.Destinations = append(retryObservation.Destinations, destination)
						}

						reobserved := waitForDelivery(t, retryObservation)

						for _, destination := range reobserved.Destinations {
							require.ElementsMatch(t, afterByChannel[destination.ChannelID].Attempts, destination.Attempts)
						}

						for id, webhook := range receivers {
							require.Equal(t, requestCounts[id], len(messageRequests(webhook)))
						}
					})
				})
			})

			t.Run("RoutingExclusion", func(t *testing.T) {
				// Exercise the same Channel through exclusion, acceptance, and exclusion
				// again. The parent Channel remains a positive delivery control.
				testRoutingExclusion := func(t *testing.T, input v4.CreateChannelInput[v4.WebhookConfig]) {
					t.Helper()

					candidateReceiver := testutils.NewWebhookServer(t, env.WebhookHost)

					input.Namespace = namespaceID
					input.ChannelID = ulid.Make().String()
					input.Config = v4.WebhookConfig{
						URL:           candidateReceiver.URL + "/routing",
						SigningSecret: signingSecret,
					}

					ctx := t.Context()

					// Remove the endpoint before closing its receiver, including on failure.
					t.Cleanup(func() {
						cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
						defer cancel()

						require.NoError(t, provider.DeleteChannel(cleanupCtx, v4.DeleteChannelInput{
							Namespace: namespaceID,
							ChannelID: input.ChannelID,
						}))
					})

					candidate, err := provider.CreateChannel(t.Context(), input)
					require.NoError(t, err)
					require.NotNil(t, candidate)
					require.Equal(t, v4.ChannelOperationStateInSync, candidate.State)
					require.NotEmpty(t, candidate.ProviderData["svix.endpoint_id"])

					excludedInput := v4.CreateChannelInput[v4.WebhookConfig]{
						Namespace:    namespaceID,
						ChannelID:    input.ChannelID,
						Name:         input.Name,
						Config:       input.Config,
						Disabled:     input.Disabled,
						EventTypes:   append([]v4.EventType{}, input.EventTypes...),
						ProviderData: candidate.ProviderData,
					}

					endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, input.ChannelID)
					require.NoError(t, err)
					AssertChannelWithEndpoint(t, endpoint, excludedInput)

					// Send a fresh Event and observe only its expected destinations.
					// Exclusion is checked independently at the receiver: the tracking
					// set passed to SendEvent does not control Svix's fan-out.
					assertRouting := func(t *testing.T, accepts bool) {
						t.Helper()

						event := v4.Event{
							NamespacedID: models.NamespacedID{
								Namespace: namespaceID,
								ID:        ulid.Make().String(),
							},
							EventType: eventType,
							CreatedAt: time.Now().UTC(),
							Payload:   json.RawMessage(`{"invoice_id":"invoice-routing"}`),
						}

						destinations := []v4.EventDestination{
							{
								ChannelID:           channelID,
								ChannelProviderData: channel.ProviderData,
							},
						}

						if accepts {
							destinations = append(destinations, v4.EventDestination{
								ChannelID:           input.ChannelID,
								ChannelProviderData: candidate.ProviderData,
							})
						}

						sent, err := provider.SendEvent(t.Context(), v4.SendEventInput{
							Event:          event,
							Destinations:   destinations,
							IdempotencyKey: new(ulid.Make().String()),
						})
						require.NoError(t, err)
						require.NotNil(t, sent)
						require.Len(t, sent.Destinations, len(destinations))

						messageID := sent.EventProviderData["svix.message_id"]
						require.NotEmpty(t, messageID)

						event.ProviderData = sent.EventProviderData
						getInput := v4.GetEventInput{Event: event}

						remaining := make(map[string]v4.ProviderData, len(destinations))

						for _, destination := range destinations {
							remaining[destination.ChannelID] = destination.ChannelProviderData
						}

						for _, destination := range sent.Destinations {
							require.Contains(t, remaining, destination.ChannelID)
							require.NoError(t, destination.Error)

							getInput.Destinations = append(getInput.Destinations, v4.GetEventDestination{
								ChannelID:                       destination.ChannelID,
								ChannelProviderData:             remaining[destination.ChannelID],
								EventDeliveryStatusProviderData: destination.EventDeliveryStatusProviderData,
								DispatchGeneration:              1,
							})

							delete(remaining, destination.ChannelID)
						}

						require.Empty(t, remaining)
						waitForDelivery(t, getInput)

						require.Eventually(t, func() bool {
							return slices.ContainsFunc(receiver.Requests(), func(request testutils.WebhookRequest) bool {
								return request.Header.Get("svix-id") == messageID
							})
						}, 5*time.Second, 100*time.Millisecond, "control must receive the new Event")

						candidateReceived := func() bool {
							return slices.ContainsFunc(candidateReceiver.Requests(), func(request testutils.WebhookRequest) bool {
								return request.Header.Get("svix-id") == messageID
							})
						}

						if accepts {
							require.Eventually(t, candidateReceived, 5*time.Second, 100*time.Millisecond,
								"enabled matching Channel must receive the new Event")

							return
						}

						// Check all captured requests, including any preceding the control's
						// success, over a bounded asynchronous-delivery observation window.
						require.Never(t, candidateReceived, time.Second, 100*time.Millisecond,
							"excluded Channel must not receive the new Event")
					}

					// When a new Event is sent, then only the control receives it.
					assertRouting(t, false)

					t.Run("AcceptAfterUpdate", func(t *testing.T) {
						// Given the excluded Channel, enable it with a matching filter.
						enabledInput := excludedInput
						enabledInput.Disabled = false
						enabledInput.EventTypes = []v4.EventType{eventType}

						// When the configuration is updated, its endpoint identity is retained.
						updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(enabledInput))
						require.NoError(t, err)
						AssertChannel(t, updated, *candidate)

						endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, input.ChannelID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, endpoint, enabledInput)

						// Then both Channels receive a fresh Event, without a resend.
						assertRouting(t, true)

						t.Run("ExcludeAfterUpdate", func(t *testing.T) {
							// Given the now-matching Channel, restore its excluded configuration.
							updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(excludedInput))
							require.NoError(t, err)

							// Then the same endpoint is retained but only the control receives a new Event.
							AssertChannel(t, updated, *candidate)

							endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, input.ChannelID)
							require.NoError(t, err)
							AssertChannelWithEndpoint(t, endpoint, excludedInput)

							assertRouting(t, false)
						})
					})
				}

				t.Run("Nonmatching", func(t *testing.T) {
					// Given a Channel subscribed to a different registered Event type.
					otherEventType := v4.EventType{
						Name:    "test." + strings.ToLower(namespaceID) + ".updated",
						Version: 1,
					}

					ctx := t.Context()

					// Registered before Channel cleanup, so the endpoint is removed first.
					t.Cleanup(func() {
						cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
						defer cancel()

						require.NoError(t, env.Client.EventType().Delete(cleanupCtx, otherEventType.String(),
							&svix.EventTypeDeleteOptions{Expunge: new(true)}))
					})

					_, err := env.Client.EventType().Create(t.Context(), svix.EventTypeIn{
						Name:        otherEventType.String(),
						Description: "Nonmatching routing integration test",
					}, nil)
					require.NoError(t, err)

					testRoutingExclusion(t, v4.CreateChannelInput[v4.WebhookConfig]{
						Name:       "Nonmatching routing receiver",
						EventTypes: []v4.EventType{otherEventType},
					})
				})

				t.Run("Disabled", func(t *testing.T) {
					// Given a disabled Channel with a matching Event type.
					testRoutingExclusion(t, v4.CreateChannelInput[v4.WebhookConfig]{
						Name:       "Disabled routing receiver",
						Disabled:   true,
						EventTypes: []v4.EventType{eventType},
					})
				})

			})

			t.Run("CatchAll", func(t *testing.T) {
				// Given two registered Event types, catch-all routing must accept both.
				otherEventType := v4.EventType{
					Name:    "test." + strings.ToLower(namespaceID) + ".catch_all",
					Version: 1,
				}

				ctx := t.Context()

				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
					defer cancel()

					require.NoError(t, env.Client.EventType().Delete(cleanupCtx, otherEventType.String(),
						&svix.EventTypeDeleteOptions{Expunge: new(true)}))
				})

				_, err := env.Client.EventType().Create(t.Context(), svix.EventTypeIn{
					Name:        otherEventType.String(),
					Description: "Catch-all routing integration test",
				}, nil)
				require.NoError(t, err)

				filterCases := []struct {
					name       string
					eventTypes []v4.EventType
				}{
					{
						name:       "NilFilter",
						eventTypes: nil,
					},
					{
						name:       "EmptyFilter",
						eventTypes: []v4.EventType{},
					},
				}

				for _, tc := range filterCases {
					t.Run(tc.name, func(t *testing.T) {
						// Given an enabled Channel with no Event-type filter.
						candidateReceiver := testutils.NewWebhookServer(t, env.WebhookHost)
						candidateID := ulid.Make().String()
						ctx := t.Context()

						t.Cleanup(func() {
							cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
							defer cancel()

							require.NoError(t, provider.DeleteChannel(cleanupCtx, v4.DeleteChannelInput{
								Namespace: namespaceID,
								ChannelID: candidateID,
							}))
						})

						config := v4.WebhookConfig{
							URL:           candidateReceiver.URL + "/catch-all",
							SigningSecret: signingSecret,
						}

						candidate, err := provider.CreateChannel(t.Context(), v4.CreateChannelInput[v4.WebhookConfig]{
							Namespace:  namespaceID,
							ChannelID:  candidateID,
							Name:       "Catch-all receiver",
							Config:     config,
							EventTypes: tc.eventTypes,
						})
						require.NoError(t, err)
						require.NotNil(t, candidate)
						require.Equal(t, v4.ChannelOperationStateInSync, candidate.State)
						require.NotEmpty(t, candidate.ProviderData["svix.endpoint_id"])

						input := v4.CreateChannelInput[v4.WebhookConfig]{
							Namespace:    namespaceID,
							ChannelID:    candidateID,
							Name:         "Catch-all receiver",
							Config:       config,
							EventTypes:   tc.eventTypes,
							ProviderData: candidate.ProviderData,
						}

						endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, candidateID)
						require.NoError(t, err)
						AssertChannelWithEndpoint(t, endpoint, input)

						assertRouting := func(t *testing.T, sentType v4.EventType, accepts bool) {
							t.Helper()

							event := v4.Event{
								NamespacedID: models.NamespacedID{
									Namespace: namespaceID,
									ID:        ulid.Make().String(),
								},
								EventType: sentType,
								CreatedAt: time.Now().UTC(),
								Payload:   json.RawMessage(`{"invoice_id":"invoice-catch-all"}`),
							}

							var destinations []v4.EventDestination

							// The existing control subscribes only to the primary Event type.
							if sentType == eventType {
								destinations = append(destinations, v4.EventDestination{
									ChannelID:           channelID,
									ChannelProviderData: channel.ProviderData,
								})
							}

							if accepts {
								destinations = append(destinations, v4.EventDestination{
									ChannelID:           candidateID,
									ChannelProviderData: candidate.ProviderData,
								})
							}

							require.NotEmpty(t, destinations, "each send must have a positive delivery control")

							sent, err := provider.SendEvent(t.Context(), v4.SendEventInput{
								Event:          event,
								Destinations:   destinations,
								IdempotencyKey: new(ulid.Make().String()),
							})
							require.NoError(t, err)
							require.NotNil(t, sent)
							require.Len(t, sent.Destinations, len(destinations))

							messageID := sent.EventProviderData["svix.message_id"]
							require.NotEmpty(t, messageID)

							event.ProviderData = sent.EventProviderData
							getInput := v4.GetEventInput{Event: event}
							remaining := make(map[string]v4.ProviderData, len(destinations))

							for _, destination := range destinations {
								remaining[destination.ChannelID] = destination.ChannelProviderData
							}

							for _, destination := range sent.Destinations {
								require.Contains(t, remaining, destination.ChannelID)
								require.NoError(t, destination.Error)

								getInput.Destinations = append(getInput.Destinations, v4.GetEventDestination{
									ChannelID:                       destination.ChannelID,
									ChannelProviderData:             remaining[destination.ChannelID],
									EventDeliveryStatusProviderData: destination.EventDeliveryStatusProviderData,
									DispatchGeneration:              1,
								})

								delete(remaining, destination.ChannelID)
							}

							require.Empty(t, remaining)
							waitForDelivery(t, getInput)

							if sentType == eventType {
								require.Eventually(t, func() bool {
									return slices.ContainsFunc(receiver.Requests(), func(request testutils.WebhookRequest) bool {
										return request.Header.Get("svix-id") == messageID
									})
								}, 5*time.Second, 100*time.Millisecond)
							}

							candidateReceived := func() bool {
								return slices.ContainsFunc(candidateReceiver.Requests(), func(request testutils.WebhookRequest) bool {
									return request.Header.Get("svix-id") == messageID
								})
							}

							if accepts {
								require.Eventually(t, candidateReceived, 5*time.Second, 100*time.Millisecond)

								return
							}

							require.Never(t, candidateReceived, time.Second, 100*time.Millisecond)
						}

						// When Events of both types are sent, then the catch-all Channel receives both.
						assertRouting(t, eventType, true)
						assertRouting(t, otherEventType, true)

						t.Run("RestrictFilter", func(t *testing.T) {
							// Given a catch-all Channel, when a concrete filter is applied.
							restricted := input
							restricted.EventTypes = []v4.EventType{otherEventType}

							updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(restricted))
							require.NoError(t, err)
							AssertChannel(t, updated, *candidate)

							endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, candidateID)
							require.NoError(t, err)
							AssertChannelWithEndpoint(t, endpoint, restricted)

							// Then only matching Events reach it, while the control still receives the others.
							assertRouting(t, eventType, false)
							assertRouting(t, otherEventType, true)

							for _, clearCase := range filterCases {
								t.Run("ClearWith"+clearCase.name, func(t *testing.T) {
									// Given a populated filter, independently of any clearing sibling.
									updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(restricted))
									require.NoError(t, err)
									AssertChannel(t, updated, *candidate)

									// When nil or an empty slice clears it, the same endpoint becomes catch-all.
									cleared := input
									cleared.EventTypes = clearCase.eventTypes

									updated, err = provider.UpdateChannel(t.Context(), ToUpdateChannelInput(cleared))
									require.NoError(t, err)
									AssertChannel(t, updated, *candidate)

									endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, candidateID)
									require.NoError(t, err)
									AssertChannelWithEndpoint(t, endpoint, cleared)

									// Then fresh Events of both types are delivered again.
									assertRouting(t, eventType, true)
									assertRouting(t, otherEventType, true)
								})
							}
						})

						t.Run("DisableCatchAll", func(t *testing.T) {
							// Given a catch-all configuration, when it is explicitly disabled.
							disabled := input
							disabled.Disabled = true

							updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(disabled))
							require.NoError(t, err)
							AssertChannel(t, updated, *candidate)

							endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, candidateID)
							require.NoError(t, err)
							AssertChannelWithEndpoint(t, endpoint, disabled)

							// Then disabling takes precedence over catch-all routing.
							assertRouting(t, eventType, false)

							t.Run("Reenable", func(t *testing.T) {
								// Given the disabled catch-all Channel, when it is enabled again.
								updated, err := provider.UpdateChannel(t.Context(), ToUpdateChannelInput(input))
								require.NoError(t, err)
								AssertChannel(t, updated, *candidate)

								endpoint, err := env.Client.Endpoint().Get(t.Context(), namespaceID, candidateID)
								require.NoError(t, err)
								AssertChannelWithEndpoint(t, endpoint, input)

								// Then both Event types are delivered without re-creating the Channel.
								assertRouting(t, eventType, true)
								assertRouting(t, otherEventType, true)
							})
						})
					})
				}
			})

			t.Run("UnsuccessfulDelivery", func(t *testing.T) {
				// Given an isolated receiver/channel that rejects valid deliveries.
				receiver := testutils.NewWebhookServer(t, env.WebhookHost)
				require.NoError(t, receiver.SetResponse(http.StatusServiceUnavailable, "temporarily unavailable"))

				channelID := ulid.Make().String()
				ctx := t.Context()

				// Remove the endpoint before closing the receiver, including on failure.
				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
					defer cancel()

					require.NoError(t, provider.DeleteChannel(cleanupCtx, v4.DeleteChannelInput{
						Namespace: namespaceID,
						ChannelID: channelID,
					}))
				})

				channel, err := provider.CreateChannel(t.Context(), v4.CreateChannelInput[v4.WebhookConfig]{
					Namespace: namespaceID,
					ChannelID: channelID,
					Name:      "Delivery recovery receiver",
					Config: v4.WebhookConfig{
						URL:           receiver.URL + "/recovery",
						SigningSecret: signingSecret,
					},
					EventTypes: []v4.EventType{eventType},
				})
				require.NoError(t, err)
				require.NotNil(t, channel)
				require.Equal(t, v4.ChannelOperationStateInSync, channel.State)

				event := v4.Event{
					NamespacedID: models.NamespacedID{
						Namespace: namespaceID,
						ID:        ulid.Make().String(),
					},
					EventType: eventType,
					CreatedAt: time.Now().UTC(),
					Payload:   json.RawMessage(`{"invoice_id":"invoice-recovery"}`),
				}

				sent, err := provider.SendEvent(t.Context(), v4.SendEventInput{
					Event: event,
					Destinations: []v4.EventDestination{
						{
							ChannelID:           channelID,
							ChannelProviderData: channel.ProviderData,
						},
					},
					IdempotencyKey: new(ulid.Make().String()),
				})
				require.NoError(t, err)
				require.NotNil(t, sent)
				require.Len(t, sent.Destinations, 1)
				require.NoError(t, sent.Destinations[0].Error)
				require.NotEmpty(t, sent.EventProviderData["svix.message_id"])

				event.ProviderData = sent.EventProviderData

				getInput := v4.GetEventInput{
					Event: event,
					Destinations: []v4.GetEventDestination{
						{
							ChannelID:                       channelID,
							ChannelProviderData:             channel.ProviderData,
							EventDeliveryStatusProviderData: sent.Destinations[0].EventDeliveryStatusProviderData,
							DispatchGeneration:              1,
						},
					},
				}

				// When Svix records the failed attempt and schedules an automatic retry.
				observationCtx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()

				var unsuccessful *v4.GetEventResult

				require.EventuallyWithT(t, func(c *assert.CollectT) {
					result, err := provider.GetEvent(observationCtx, getInput)
					require.NoError(c, err)
					require.NotNil(c, result)
					require.Len(c, result.Destinations, 1)

					destination := result.Destinations[0]

					// Then a failed attempt is not terminal while another attempt is scheduled.
					require.NoError(c, destination.Error)
					require.Equal(c, channelID, destination.ChannelID)
					require.Equal(c, v4.EventDeliveryStatusStateSending, destination.State)
					require.NotNil(c, destination.NextObservationAt)
					require.False(c, destination.NextObservationAt.IsZero())
					require.NotEmpty(c, destination.Attempts)

					for _, attempt := range destination.Attempts {
						require.Equal(c, v4.EventDeliveryStatusStateFailed, attempt.State)
						require.NotEmpty(c, attempt.ProviderData["svix.attempt_id"])

						var response struct {
							StatusCode int    `json:"status_code"`
							Body       string `json:"body"`
						}

						require.NoError(c, json.Unmarshal(attempt.Response, &response))
						require.Equal(c, http.StatusServiceUnavailable, response.StatusCode)
						require.Equal(c, "temporarily unavailable", response.Body)
					}

					unsuccessful = result
				}, 30*time.Second, 200*time.Millisecond)

				t.Run("RecoverOnAutomaticRetry", func(t *testing.T) {
					// Given the failed attempt's observation is retained for reconciliation.
					input := getInput
					input.Destinations = slices.Clone(getInput.Destinations)
					input.Destinations[0].EventDeliveryStatusProviderData = unsuccessful.Destinations[0].EventDeliveryStatusProviderData

					t.Logf("Svix scheduled the next attempt at %s", unsuccessful.Destinations[0].NextObservationAt)

					// When the receiver recovers, without another send or explicit resend.
					require.NoError(t, receiver.SetResponse(http.StatusNoContent, ""))

					ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
					defer cancel()

					require.EventuallyWithT(t, func(c *assert.CollectT) {
						result, err := provider.GetEvent(ctx, input)
						require.NoError(c, err)
						require.NotNil(c, result)
						require.Len(c, result.Destinations, 1)
						require.Equal(c, sent.EventProviderData, result.EventProviderData)

						destination := result.Destinations[0]

						// Then the automatic retry succeeds while retaining the failed history.
						require.NoError(c, destination.Error)
						require.Equal(c, channelID, destination.ChannelID)
						require.Equal(c, v4.EventDeliveryStatusStateSuccess, destination.State)
						require.Nil(c, destination.NextObservationAt)
						require.Equal(c, "1", destination.EventDeliveryStatusProviderData["svix.dispatch_generation"])
						require.Subset(c, destination.Attempts, unsuccessful.Destinations[0].Attempts)

						successfulAttempts := 0

						for _, attempt := range destination.Attempts {
							if attempt.State != v4.EventDeliveryStatusStateSuccess {
								continue
							}

							var response struct {
								StatusCode int `json:"status_code"`
							}

							require.NoError(c, json.Unmarshal(attempt.Response, &response))
							require.Equal(c, http.StatusNoContent, response.StatusCode)

							successfulAttempts++
						}

						require.Positive(c, successfulAttempts)
					}, time.Minute, 200*time.Millisecond)

					requests := receiver.Requests()
					require.GreaterOrEqual(t, len(requests), 2)

					for _, request := range requests {
						require.NoError(t, verifier.Verify(request.Body, request.Header))
						require.Equal(t, sent.EventProviderData["svix.message_id"], request.Header.Get("svix-id"))
						require.JSONEq(t, string(requests[0].Body), string(request.Body))
					}
				})
			})
		})
	})
}

// AssertNamespace compares namespace state while keeping user metadata and provider data separate.
func AssertNamespace(t *testing.T, actual *v4.ProviderNamespace, expected v4.ProviderNamespace) {
	t.Helper()

	require.NotNil(t, actual)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Name, actual.Name)
	require.True(t, expected.Metadata.Equal(actual.Metadata),
		"expected metadata %v, got %v", expected.Metadata, actual.Metadata)
	require.Equal(t, expected.ProviderData, actual.ProviderData)
}

// AssertNamespaceWithApplication checks raw Svix state independently of the provider's mapping.
func AssertNamespaceWithApplication(t *testing.T, actual *svix.ApplicationOut, expected v4.ProviderNamespace) {
	t.Helper()

	applicationID := expected.ProviderData["svix.application_id"]
	require.NotEmpty(t, applicationID)

	require.NotNil(t, actual)
	require.NotNil(t, actual.Uid)
	require.Equal(t, expected.ID, *actual.Uid)
	require.Equal(t, applicationID, actual.Id)
	require.Equal(t, expected.Name, actual.Name)
	require.True(t, expected.Metadata.Equal(models.Metadata(actual.Metadata)),
		"expected Svix metadata %v, got %v", expected.Metadata, actual.Metadata)
}

// AssertChannel compares reconciliation state and provider data without fetching remote state.
// ToUpdateNamespaceInput explicitly reapplies the complete test fixture state.
func ToUpdateNamespaceInput(namespace v4.ProviderNamespace) v4.UpdateNamespaceInput {
	return v4.UpdateNamespaceInput{
		ID:           namespace.ID,
		Name:         &namespace.Name,
		Metadata:     &namespace.Metadata,
		ProviderData: namespace.ProviderData,
	}
}

// ToUpdateChannelInput explicitly reapplies a full fixture, including empty
// headers and filters. An unspecified creation secret preserves the current key.
func ToUpdateChannelInput(channel v4.CreateChannelInput[v4.WebhookConfig]) v4.UpdateChannelInput[v4.WebhookConfigUpdate] {
	config := v4.WebhookConfigUpdate{
		URL:           &channel.Config.URL,
		CustomHeaders: &channel.Config.CustomHeaders,
	}
	if channel.Config.SigningSecret != "" {
		config.SigningSecret = &channel.Config.SigningSecret
	}

	return v4.UpdateChannelInput[v4.WebhookConfigUpdate]{
		Namespace:    channel.Namespace,
		ChannelID:    channel.ChannelID,
		Name:         &channel.Name,
		Config:       config,
		Disabled:     &channel.Disabled,
		EventTypes:   &channel.EventTypes,
		ProviderData: channel.ProviderData,
	}
}

func AssertChannel(t *testing.T, actual *v4.ChannelOperationResult, expected v4.ChannelOperationResult) {
	t.Helper()

	require.NotNil(t, actual)
	require.Equal(t, expected.State, actual.State)
	require.Equal(t, expected.NextObservationAt, actual.NextObservationAt)
	require.Equal(t, expected.ProviderData, actual.ProviderData)
}

// AssertChannelWithEndpoint checks endpoint identity and routing independently of the provider's mapping.
// Headers and signing secrets are separate Svix resources and are asserted by the calling test.
func AssertChannelWithEndpoint(t *testing.T, actual *svix.EndpointOut, expected v4.CreateChannelInput[v4.WebhookConfig]) {
	t.Helper()

	endpointID := expected.ProviderData["svix.endpoint_id"]
	require.NotEmpty(t, endpointID)

	require.NotNil(t, actual)
	require.NotNil(t, actual.Uid)
	require.Equal(t, expected.ChannelID, *actual.Uid)
	require.Equal(t, endpointID, actual.Id)
	require.Equal(t, expected.Name, actual.Description)
	require.Equal(t, expected.Config.URL, actual.Url)
	require.Empty(t, actual.Metadata)
	require.Empty(t, actual.Channels)

	disabled := actual.Disabled != nil && *actual.Disabled
	require.Equal(t, expected.Disabled, disabled)

	eventTypes := make([]string, 0, len(expected.EventTypes))

	for _, eventType := range expected.EventTypes {
		eventTypes = append(eventTypes, eventType.String())
	}

	require.ElementsMatch(t, eventTypes, actual.EventTypes)
}
