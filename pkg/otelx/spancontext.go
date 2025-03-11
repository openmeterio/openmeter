package otelx

import (
	"encoding/base64"
	"encoding/json"

	"go.opentelemetry.io/otel/trace"
)

const OTelSpanContextKey = "otel.span.context"

func SerializeSpanContext(c trace.SpanContext) ([]byte, error) {
	b, err := c.MarshalJSON()
	if err != nil {
		return nil, err
	}

	s := base64.StdEncoding.EncodeToString(b)

	return []byte(s), nil
}

func DeserializeSpanContext(b []byte) (*trace.SpanContext, error) {
	b, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		return nil, err
	}

	config := trace.SpanContextConfig{}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	ctx := trace.NewSpanContext(config)

	return &ctx, nil
}
