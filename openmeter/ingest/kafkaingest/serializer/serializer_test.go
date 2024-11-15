package serializer

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
)

func TestToCloudEventsKafkaPayload(t *testing.T) {
	tm := time.Now()

	tests := []struct {
		description string
		event       func() event.Event
		want        CloudEventsKafkaPayload
		error       error
	}{
		{
			description: "should serialize cloud event",
			event: func() event.Event {
				ev := event.New()
				ev.SetID("123")
				ev.SetSource("test")
				ev.SetType("test")
				ev.SetSubject("test")
				ev.SetTime(tm)
				err := ev.SetData(event.ApplicationJSON, map[string]string{"key": "value"})
				assert.Nil(t, err)
				return ev
			},
			want: CloudEventsKafkaPayload{
				Id:      "123",
				Type:    "test",
				Source:  "test",
				Subject: "test",
				Time:    tm.Unix(),
				Data:    `{"key":"value"}`,
			},
		},
		{
			description: "should handle when data is not set",
			event: func() event.Event {
				ev := event.New()
				ev.SetID("123")
				ev.SetSource("test")
				ev.SetType("test")
				ev.SetSubject("test")
				ev.SetTime(tm)
				return ev
			},
			want: CloudEventsKafkaPayload{
				Id:      "123",
				Type:    "test",
				Source:  "test",
				Subject: "test",
				Time:    tm.Unix(),
				Data:    "",
			},
		},
		{
			description: "should return error when data is invalid",
			event: func() event.Event {
				ev := event.New()
				ev.SetID("123")
				ev.SetSource("test")
				ev.SetType("test")
				ev.SetSubject("test")
				ev.SetTime(tm)
				// We use byte array otherwise SetData validates the data
				err := ev.SetData(event.ApplicationJSON, []byte("invalid"))
				assert.Nil(t, err)
				return ev
			},
			want:  CloudEventsKafkaPayload{},
			error: fmt.Errorf("invalid character 'i' looking for beginning of value"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCloudEventsKafkaPayload(tt.event())

			if tt.error != nil {
				assert.Errorf(t, tt.error, err.Error())

				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.want, payload)
		})
	}
}

func TestFromKafkaPayloadToCloudEvents(t *testing.T) {
	// Clear location information from time
	tm := time.Unix(time.Now().Unix(), 0)

	tests := []struct {
		description string
		payload     CloudEventsKafkaPayload
		want        func() event.Event
		error       error
	}{
		{
			description: "should parse to cloud event",
			payload: CloudEventsKafkaPayload{
				Id:      "123",
				Type:    "test",
				Source:  "test",
				Subject: "test",
				Time:    tm.Unix(),
				Data:    `{"key":"value"}`,
			},
			want: func() event.Event {
				ev := event.New()
				ev.SetID("123")
				ev.SetSource("test")
				ev.SetType("test")
				ev.SetSubject("test")
				ev.SetTime(tm)
				err := ev.SetData(event.ApplicationJSON, map[string]string{"key": "value"})
				assert.Nil(t, err)
				return ev
			},
		},
		{
			description: "should handle when data is not set",
			payload: CloudEventsKafkaPayload{
				Id:      "123",
				Type:    "test",
				Source:  "test",
				Subject: "test",
				Time:    tm.Unix(),
				Data:    "",
			},
			want: func() event.Event {
				ev := event.New()
				ev.SetID("123")
				ev.SetSource("test")
				ev.SetType("test")
				ev.SetSubject("test")
				ev.SetTime(tm)
				return ev
			},
		},
		{
			description: "should handle when data is null",
			payload: CloudEventsKafkaPayload{
				Id:      "123",
				Type:    "test",
				Source:  "test",
				Subject: "test",
				Time:    tm.Unix(),
				Data:    "null",
			},
			want: func() event.Event {
				ev := event.New()
				ev.SetID("123")
				ev.SetSource("test")
				ev.SetType("test")
				ev.SetSubject("test")
				ev.SetTime(tm)
				err := ev.SetData(event.ApplicationJSON, nil)
				assert.Nil(t, err)
				return ev
			},
		},
		{
			description: "should return error when data is invalid",
			payload: CloudEventsKafkaPayload{
				Id:      "123",
				Type:    "test",
				Source:  "test",
				Subject: "test",
				Time:    tm.Unix(),
				Data:    "invalid",
			},
			want: func() event.Event {
				ev := event.New()
				return ev
			},
			error: fmt.Errorf("invalid character 'i' looking for beginning of value"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			ev, err := FromKafkaPayloadToCloudEvents(tt.payload)

			if tt.error != nil {
				assert.Errorf(t, tt.error, err.Error())

				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.want(), ev)
		})
	}
}
