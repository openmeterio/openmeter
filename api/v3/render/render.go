package render

import (
	"encoding/json"
	"net/http"

	"github.com/invopop/yaml"
)

const (
	ContentTypeKey       = "Content-Type"
	ContentTypeJSONValue = "application/json"
	ContentTypeYAMLValue = "application/yaml"
	defaultStatusCode    = http.StatusOK
)

type config struct {
	statusCode int
}

type Option func(w http.ResponseWriter, c *config)

func WithStatus(status int) Option {
	return func(_ http.ResponseWriter, c *config) {
		c.statusCode = status
	}
}

func WithHeader(key, value string) Option {
	return func(w http.ResponseWriter, _ *config) {
		w.Header().Set(key, value)
	}
}

func WithContentType(contentType string) Option {
	return WithHeader(ContentTypeKey, contentType)
}

// RenderJSON renders json object with returned apierrors
func RenderJSON[O any](w http.ResponseWriter, o O, opts ...Option) error {
	c := &config{
		statusCode: defaultStatusCode,
	}

	for _, opt := range opts {
		opt(w, c)
	}

	if w.Header().Get(ContentTypeKey) == "" {
		w.Header().Set(ContentTypeKey, ContentTypeJSONValue)
	}

	w.WriteHeader(c.statusCode)

	body, err := json.Marshal(o)
	if err != nil {
		return err
	}

	_, err = w.Write(body)
	return err
}

func RenderYAML[O any](w http.ResponseWriter, o O, opts ...Option) error {
	c := &config{
		statusCode: defaultStatusCode,
	}

	for _, opt := range opts {
		opt(w, c)
	}

	if w.Header().Get(ContentTypeKey) == "" {
		w.Header().Set(ContentTypeKey, ContentTypeYAMLValue)
	}

	w.WriteHeader(c.statusCode)

	body, err := yaml.Marshal(o)
	if err != nil {
		return err
	}

	_, err = w.Write(body)
	return err
}
