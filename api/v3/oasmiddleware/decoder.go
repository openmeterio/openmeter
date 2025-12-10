package oasmiddleware

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// ref: https://github.com/getkin/kin-openapi/blob/994d4f01c1e8dd613805668a7c10b568547f7789/openapi3filter/req_resp_decoder.go#L1031-L1047

// JsonBodyDecoder is meant to be used with openapi3filter.RegisterBodyDecoder
// to register a decoder for a custom vendor type like "application/konnect.foo+json"
func JsonBodyDecoder(body io.Reader, _ http.Header, _ *openapi3.SchemaRef, _ openapi3filter.EncodingFn) (any, error) {
	var value any
	dec := json.NewDecoder(body)
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, &openapi3filter.ParseError{Kind: openapi3filter.KindInvalidFormat, Cause: err}
	}

	return value, nil
}
