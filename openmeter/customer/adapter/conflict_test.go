package adapter_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newConflictTestEnv(t *testing.T) (*testEnv, *bytes.Buffer) {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	return newTestEnvWithLogger(t, logger), logs
}

func TestCustomerConflictRedaction(t *testing.T) {
	t.Run("CreateKeyCustomerIDOverlap", func(t *testing.T) {
		env, logs := newConflictTestEnv(t)
		const namespace = "sensitive-create-id-namespace"
		conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key")

		_, err := env.adapter.CreateCustomer(t.Context(), customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr(conflictingCustomerID),
				Name: "new customer",
			},
		})

		assertCustomerConflictResponse(
			t,
			err,
			fmt.Sprintf("customer key %q overlaps with the ID of another customer", conflictingCustomerID),
			namespace,
		)
		assertConflictLog(t, logs, "customer key overlaps with customer ID", map[string]any{
			"namespace":               namespace,
			"customer_key":            conflictingCustomerID,
			"conflicting_customer_id": conflictingCustomerID,
		})
	})

	t.Run("CreateKeySubjectOverlap", func(t *testing.T) {
		env, logs := newConflictTestEnv(t)
		const namespace = "sensitive-create-subject-namespace"
		conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key", "existing-subject")

		_, err := env.adapter.CreateCustomer(t.Context(), customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("existing-subject"),
				Name: "new customer",
			},
		})

		assertCustomerConflictResponse(
			t,
			err,
			fmt.Sprintf(`customer key "existing-subject" overlaps with a usage attribution key of another customer: %s`, conflictingCustomerID),
			namespace,
		)
		assertConflictLog(t, logs, "customer key overlaps with customer usage attribution key", map[string]any{
			"namespace":                         namespace,
			"customer_key":                      "existing-subject",
			"conflicting_usage_attribution_key": "existing-subject",
			"conflicting_customer_id":           conflictingCustomerID,
		})
	})

	t.Run("CreateKeyAndSubjectUniqueness", func(t *testing.T) {
		t.Run("Key", func(t *testing.T) {
			env, logs := newConflictTestEnv(t)
			const namespace = "sensitive-create-key-namespace"
			conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key")

			_, err := env.adapter.CreateCustomer(t.Context(), customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("existing-key"),
					Name: "new customer",
				},
			})

			require.True(t, customer.IsKeyConflictError(err))
			assertCustomerConflictResponse(t, err, `customer key "existing-key" is already in use`, namespace, conflictingCustomerID)
			assertConflictLog(t, logs, "customer key conflict while creating customer", map[string]any{
				"namespace":    namespace,
				"customer_key": "existing-key",
			}, "db: constraint failed", "customer_namespace_key", "SQLSTATE 23505")
		})

		t.Run("Subject", func(t *testing.T) {
			env, logs := newConflictTestEnv(t)
			const namespace = "sensitive-create-subject-uniqueness-namespace"
			conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key", "existing-subject")

			_, err := env.adapter.CreateCustomer(t.Context(), customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("new-key"),
					Name: "new customer",
					UsageAttribution: &customer.CustomerUsageAttribution{
						SubjectKeys: []string{"existing-subject"},
					},
				},
			})

			require.True(t, customer.IsSubjectKeyConflictError(err))
			assertCustomerConflictResponse(t, err, `one or more customer usage attribution keys are already in use: ["existing-subject"]`, namespace, conflictingCustomerID)
			assertConflictLog(t, logs, "customer usage attribution key conflict while creating customer", map[string]any{
				"namespace":              namespace,
				"usage_attribution_keys": []string{"existing-subject"},
			}, "db: constraint failed", "customersubjects_namespace_subject_key", "SQLSTATE 23505")
		})
	})

	t.Run("UpdateKeyCustomerIDAndSubjectOverlap", func(t *testing.T) {
		t.Run("CustomerID", func(t *testing.T) {
			env, logs := newConflictTestEnv(t)
			const namespace = "sensitive-update-id-namespace"
			conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key")
			requestedCustomerID := env.seedCustomerWithKey(namespace, "updating-key")

			_, err := env.adapter.UpdateCustomer(t.Context(), customer.UpdateCustomerInput{
				CustomerID: customer.CustomerID{Namespace: namespace, ID: requestedCustomerID},
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr(conflictingCustomerID),
					Name: "updated customer",
				},
			})

			assertCustomerConflictResponse(
				t,
				err,
				fmt.Sprintf("customer key %q overlaps with the ID of another customer", conflictingCustomerID),
				namespace,
				requestedCustomerID,
			)
			assertConflictLog(t, logs, "customer key overlaps with customer ID", map[string]any{
				"namespace":               namespace,
				"customer_id":             requestedCustomerID,
				"customer_key":            conflictingCustomerID,
				"conflicting_customer_id": conflictingCustomerID,
			})
		})

		t.Run("Subject", func(t *testing.T) {
			env, logs := newConflictTestEnv(t)
			const namespace = "sensitive-update-subject-namespace"
			conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key", "existing-subject")
			requestedCustomerID := env.seedCustomerWithKey(namespace, "updating-key")

			_, err := env.adapter.UpdateCustomer(t.Context(), customer.UpdateCustomerInput{
				CustomerID: customer.CustomerID{Namespace: namespace, ID: requestedCustomerID},
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("existing-subject"),
					Name: "updated customer",
				},
			})

			assertCustomerConflictResponse(
				t,
				err,
				fmt.Sprintf(`customer key "existing-subject" overlaps with a usage attribution key of another customer: %s`, conflictingCustomerID),
				namespace,
				requestedCustomerID,
			)
			assertConflictLog(t, logs, "customer key overlaps with customer usage attribution key", map[string]any{
				"namespace":                         namespace,
				"customer_id":                       requestedCustomerID,
				"customer_key":                      "existing-subject",
				"conflicting_usage_attribution_key": "existing-subject",
				"conflicting_customer_id":           conflictingCustomerID,
			})
		})
	})

	t.Run("UpdateKeyAndSubjectUniqueness", func(t *testing.T) {
		t.Run("Key", func(t *testing.T) {
			env, logs := newConflictTestEnv(t)
			const namespace = "sensitive-update-key-namespace"
			conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key")
			requestedCustomerID := env.seedCustomerWithKey(namespace, "updating-key")

			_, err := env.adapter.UpdateCustomer(t.Context(), customer.UpdateCustomerInput{
				CustomerID: customer.CustomerID{Namespace: namespace, ID: requestedCustomerID},
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("existing-key"),
					Name: "updated customer",
				},
			})

			require.True(t, customer.IsKeyConflictError(err))
			assertCustomerConflictResponse(t, err, `customer key "existing-key" is already in use`, namespace, requestedCustomerID, conflictingCustomerID)
			assertConflictLog(t, logs, "customer key conflict while updating customer", map[string]any{
				"namespace":    namespace,
				"customer_id":  requestedCustomerID,
				"customer_key": "existing-key",
			}, "db: constraint failed", "customer_namespace_key", "SQLSTATE 23505")
		})

		t.Run("Subject", func(t *testing.T) {
			env, logs := newConflictTestEnv(t)
			const namespace = "sensitive-update-subject-uniqueness-namespace"
			conflictingCustomerID := env.seedCustomerWithKey(namespace, "existing-key", "existing-subject")
			requestedCustomerID := env.seedCustomerWithKey(namespace, "updating-key")

			_, err := env.adapter.UpdateCustomer(t.Context(), customer.UpdateCustomerInput{
				CustomerID: customer.CustomerID{Namespace: namespace, ID: requestedCustomerID},
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("updating-key"),
					Name: "updated customer",
					UsageAttribution: &customer.CustomerUsageAttribution{
						SubjectKeys: []string{"existing-subject"},
					},
				},
			})

			require.True(t, customer.IsSubjectKeyConflictError(err))
			assertCustomerConflictResponse(t, err, `one or more customer usage attribution keys are already in use: ["existing-subject"]`, namespace, requestedCustomerID, conflictingCustomerID)
			assertConflictLog(t, logs, "customer usage attribution key conflict while updating customer", map[string]any{
				"namespace":              namespace,
				"customer_id":            requestedCustomerID,
				"usage_attribution_keys": []string{"existing-subject"},
			}, "db: constraint failed", "customersubjects_namespace_subject_key", "SQLSTATE 23505")
		})
	})
}

func assertCustomerConflictResponse(t *testing.T, err error, publicMessage string, excludedValues ...string) {
	t.Helper()

	require.Error(t, err)
	require.True(t, models.IsGenericConflictError(err))
	require.EqualError(t, err, "conflict error: "+publicMessage)

	recorder := httptest.NewRecorder()
	handled := commonhttp.GenericErrorEncoder()(t.Context(), err, recorder, httptest.NewRequest(http.MethodPost, "/", nil))
	require.True(t, handled)
	require.Equal(t, http.StatusConflict, recorder.Code)

	response := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "conflict error: "+publicMessage, response["detail"])

	for _, value := range excludedValues {
		require.NotContains(t, err.Error(), value)
		require.NotContains(t, response["detail"], value)
	}
}

func assertConflictLog(t *testing.T, logs *bytes.Buffer, message string, expectedFields map[string]any, expectedErrorValues ...string) {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, 1)

	logEntry := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &logEntry))
	require.Equal(t, "WARN", logEntry["level"])
	require.Equal(t, message, logEntry["msg"])

	for field, expectedValue := range expectedFields {
		if expectedStrings, ok := expectedValue.([]string); ok {
			actualValues, ok := logEntry[field].([]any)
			require.True(t, ok, "log field %q must be an array", field)

			actualStrings := lo.Map(actualValues, func(value any, _ int) string {
				actualString, ok := value.(string)
				require.True(t, ok, "log field %q must contain strings", field)

				return actualString
			})
			require.Equal(t, expectedStrings, actualStrings)

			continue
		}

		require.Equal(t, expectedValue, logEntry[field], "unexpected value for log field %q", field)
	}

	for _, value := range expectedErrorValues {
		require.Contains(t, logEntry["error"], value)
	}
}
