package httpdriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlerConfigValidate(t *testing.T) {
	err := HandlerConfig{}.Validate()
	require.Error(t, err)

	for _, want := range []string{
		"subscription workflow service is required",
		"subscription service is required",
		"customer service is required",
		"plan subscription service is required",
		"namespace decoder is required",
		"logger is required",
	} {
		require.ErrorContains(t, err, want)
	}
}

func TestNewHandlerInvalidConfig(t *testing.T) {
	handler, err := NewHandler(HandlerConfig{})
	require.Error(t, err)
	require.Nil(t, handler)
	require.ErrorContains(t, err, "invalid subscription handler config")
	require.ErrorContains(t, err, "plan subscription service is required")
}

func TestHandlerConfigValidateTypedNilDependency(t *testing.T) {
	var namespaceDecoder *nilNamespaceDecoder

	err := HandlerConfig{
		NamespaceDecoder: namespaceDecoder,
	}.Validate()

	require.Error(t, err)
	require.ErrorContains(t, err, "namespace decoder is required")
}

type nilNamespaceDecoder struct{}

func (*nilNamespaceDecoder) GetNamespace(context.Context) (string, bool) {
	return "", false
}
