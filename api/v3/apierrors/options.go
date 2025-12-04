package apierrors

type configOption func(*config)

// ContextCanceledError is a sample error to be used with
// WithRecastContextCancelled option
var ContextCanceledError = &BaseAPIError{
	Status: 499,
	Title:  "Client Closed Request",
	Detail: "context cancelled",
}

// WithRecastContextCancelled sets the BaseAPIError that would be recasted into
// if an APIError is a context.Canceled. This is to avoid false 500 errors
func WithRecastContextCancelled(e BaseAPIError) configOption { // nolint:gocritic
	return func(c *config) {
		c.recastContextCanceled = e
	}
}
