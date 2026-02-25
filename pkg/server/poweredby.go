package server

import "net/http"

const poweredByHeader = "X-Powered-By"
const poweredByValue = "OpenMeter by Kong, Inc."

// NewPoweredByMiddleware returns a middleware that adds the X-Powered-By header
// to every response.
func NewPoweredByMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(poweredByHeader, poweredByValue)
			next.ServeHTTP(w, r)
		})
	}
}
