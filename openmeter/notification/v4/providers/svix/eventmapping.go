package svix

import (
	"encoding/json"
	"fmt"
	"strings"

	svixv2 "github.com/svix/svix-webhooks/v2/go"
	svixmodels "github.com/svix/svix-webhooks/v2/go/models"

	"github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/models"
)

const maxAttemptResponseSize = 64 * 1024

func toAPIMessageIn(event v4.Event) svixv2.MessageIn {
	annotations := event.Annotations
	if annotations == nil {
		annotations = models.Annotations{}
	}

	// RawMessage preserves the domain JSON, including large numeric values.
	// Internal provider data and live Rule identity never enter the wire envelope.
	payload := map[string]any{
		"id":          event.ID,
		"event_type":  event.EventType.String(),
		"created_at":  event.CreatedAt,
		"payload":     event.Payload,
		"annotations": annotations,
	}

	if snapshot := event.RuleSnapshot; snapshot != nil {
		payload["rule_snapshot"] = map[string]any{
			"name":          snapshot.Name,
			"event_type":    snapshot.EventType.String(),
			"revision":      snapshot.Revision,
			"configuration": snapshot.Configuration,
		}
	}

	return svixv2.MessageIn{
		EventId:   &event.ID,
		EventType: event.EventType.String(),
		Payload:   payload,
	}
}

func fromAPIMessageStatus(namespace string, status svixmodels.MessageStatus) (v4.EventDeliveryStatusState, error) {
	switch status {
	case svixmodels.MESSAGESTATUS_SUCCESS:
		return v4.EventDeliveryStatusStateSuccess, nil
	case svixmodels.MESSAGESTATUS_PENDING, svixmodels.MESSAGESTATUS_SENDING:
		return v4.EventDeliveryStatusStateSending, nil
	case svixmodels.MESSAGESTATUS_FAIL, svixmodels.MESSAGESTATUS_CANCELED:
		return v4.EventDeliveryStatusStateFailed, nil
	default:
		return "", fmt.Errorf(
			"invalid message status [svix.message.status=%d svix.application.uid=%s]",
			status,
			namespace,
		)
	}
}

func fromAPIMessageAttemptOut(namespace string, attempt svixmodels.MessageAttemptOut) (*v4.EventDeliveryAttempt, error) {
	state, err := fromAPIMessageStatus(namespace, attempt.Status)
	if err != nil {
		return nil, err
	}

	response := struct {
		StatusCode        int16  `json:"status_code"`
		Body              string `json:"body"`
		DurationMs        int64  `json:"duration_ms"`
		URL               string `json:"url"`
		ResponseTruncated bool   `json:"response_truncated"`
	}{
		StatusCode: attempt.ResponseStatusCode,
		Body:       attempt.Response,
		DurationMs: attempt.ResponseDurationMs,
		URL:        attempt.Url,
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to encode message attempt response [svix.attempt.id=%s svix.message.id=%s svix.endpoint.id=%s svix.application.uid=%s]: %w",
			attempt.Id,
			attempt.MsgId,
			attempt.EndpointId,
			namespace,
			err,
		)
	}

	if len(encoded) > maxAttemptResponseSize {
		// Drop oversized content rather than slicing serialized JSON, which
		// could leave invalid escapes or broken UTF-8 in the stored response.
		response.Body = ""
		response.ResponseTruncated = true

		encoded, err = json.Marshal(response)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to encode truncated message attempt response [svix.attempt.id=%s svix.message.id=%s svix.endpoint.id=%s svix.application.uid=%s]: %w",
				attempt.Id,
				attempt.MsgId,
				attempt.EndpointId,
				namespace,
				err,
			)
		}

		if len(encoded) > maxAttemptResponseSize {
			encoded = []byte(`{"response_truncated":true}`)
		}
	}

	return &v4.EventDeliveryAttempt{
		State:        state,
		Timestamp:    attempt.Timestamp,
		Response:     encoded,
		ProviderData: v4.ProviderData{providerDataAttemptID: attempt.Id},
	}, nil
}

// Results replace this provider's subset without returning another provider's keys.
func svixProviderData(data v4.ProviderData) v4.ProviderData {
	result := make(v4.ProviderData)

	for key, value := range data {
		if strings.HasPrefix(key, "svix.") {
			result[key] = value
		}
	}

	return result
}
