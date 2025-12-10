package apierrors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/openmeterio/openmeter/api/v3/render"
)

// BaseAPIError is the schema for all API apierrors.
type BaseAPIError struct {
	// A unique identifier for this error. When dereferenced it must provide
	// human-readable documentation for the problem. The URL must follow
	// (#122 - Resource Names) and must not contain URI fragments type.
	Type string `json:"type"`

	// The HTTP status code of the error. Useful when passing the response body
	// to child properties in a frontend UI. Must be returned as an integer.
	Status int `json:"status"`

	// A short, human-readable summary of the problem. It should not change
	// between occurrences of a problem, except for localization. Should be
	// provided as "Sentence case" for direct use in the UI.
	Title string `json:"title"`

	// Used to return the correlation ID back to the user, in the format
	// {product}:trace:<trace_id>. This helps us find the relevant logs when a
	// customer reports an issue.
	Instance string `json:"instance"`

	// A human-readable explanation specific to this occurrence of the problem.
	// This field may contain request/entity data to help the user understand
	// what went wrong. Enclose variable values in square brackets. Should be
	// provided as "Sentence case" for direct use in the UI.
	Detail string `json:"detail"`

	// Used to indicate which fields have invalid values when validated. Both a
	// human-readable value (reason) and a type that can be used for localized
	// results (rule) are provided.
	InvalidParameters InvalidParameters `json:"invalid_parameters,omitempty"`

	// UnderlyingError is the underlying error stack to be logged.
	// NOTE: this should not be returned to callers.
	UnderlyingError error `json:"-"`

	// The context used to extract a logger.
	ctx context.Context
}

// InvalidParameters is a collection of fields that failed input validation.
type InvalidParameters []InvalidParameter

type InvalidParameterSource uint8

const (
	InvalidParamSourcePath InvalidParameterSource = iota + 1
	InvalidParamSourceQuery
	InvalidParamSourceBody
	InvalidParamSourceHeader
)

func (i InvalidParameterSource) String() string {
	switch i {
	case InvalidParamSourceBody:
		return "body"
	case InvalidParamSourcePath:
		return "path"
	case InvalidParamSourceHeader:
		return "header"
	case InvalidParamSourceQuery:
		return "query"
	}
	return ""
}

func ToInvalid(s string) InvalidParameterSource {
	switch s {
	case "query":
		return InvalidParamSourceQuery
	case "path":
		return InvalidParamSourcePath
	case "body":
		return InvalidParamSourceBody
	case "header":
		return InvalidParamSourceHeader
	}
	return InvalidParameterSource(0)
}

func (i InvalidParameterSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *InvalidParameterSource) UnmarshalJSON(data []byte) error {
	var source string
	if err := json.Unmarshal(data, &source); err != nil {
		return err
	}
	*i = ToInvalid(source)
	return nil
}

// InvalidParameter is a single field that failed input validation.
type InvalidParameter struct {
	// Field concerned by the error.
	Field string `json:"field"`

	// Rule represents the rule that has triggered the error.
	Rule string `json:"rule,omitempty"`

	// Reason describes why the error has been triggered.
	Reason string `json:"reason"`

	// Source describes where the error has been triggered: body, header, path.
	Source InvalidParameterSource `json:"source"`

	// Choices represents the available choices for value in a case of an enum.
	Choices []string `json:"choices,omitempty"`

	// Minimum is an optional field for setting the minimum required value for
	// an attribute.
	Minimum *int `json:"minimum,omitempty"`

	// Maximum is an optional field for setting the maximum required value for
	// an attribute.
	Maximum *int `json:"maximum,omitempty"`

	// Dependents is an optional field for when the rule "dependent_fields" is
	// applied.
	Dependents []string `json:"dependents,omitempty"`
}

// Stringer method for a collection of InvalidParameter entities.
func (ips InvalidParameters) String() string {
	out := new(strings.Builder)
	for i, param := range ips {
		out.WriteString(param.Field)
		if param.Rule != "" {
			_, _ = fmt.Fprintf(out, " [%s]", param.Rule)
		}
		_, _ = fmt.Fprintf(out, ": %s", param.Reason)
		if i != len(ips)-1 {
			_, _ = fmt.Fprintf(out, ", ")
		}
	}
	return out.String()
}

// Error satisfies the error interface.
func (bae *BaseAPIError) Error() string {
	switch {
	case bae.InvalidParameters != nil:
		return fmt.Sprintf("%s: %s", bae.UnderlyingError, bae.InvalidParameters.String())
	case bae.Detail != "" && bae.UnderlyingError != nil:
		return fmt.Sprintf("%s: %s", bae.Detail, bae.UnderlyingError)
	case bae.Detail != "":
		return bae.Detail
	}
	return bae.UnderlyingError.Error()
}

// Unwrap returns the underlying error
func (bae *BaseAPIError) Unwrap() error {
	return bae.UnderlyingError
}

// Context is the context that created the error
func (bae *BaseAPIError) Context() context.Context {
	return bae.ctx
}

// HandleAPIError is a helper function that accepts an error
func (bae *BaseAPIError) HandleAPIError(
	w http.ResponseWriter,
	r *http.Request,
) {
	_ = render.RenderJSON(w, bae, render.WithContentType(ContentTypeProblemValue), render.WithStatus(bae.Status))
}
