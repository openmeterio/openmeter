package server

import "net/http"

// MiddlewareFunc is a shared middleware type used across API versions.
type MiddlewareFunc func(http.Handler) http.Handler
