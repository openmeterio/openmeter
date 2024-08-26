package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"
)

type corsOptions struct {
	cors.Options
	AllowedPaths []string
}

func corsHandler(options corsOptions) func(next http.Handler) http.Handler {
	ch := cors.Handler(options.Options)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If AllowedPaths is empty, apply CORS to all paths
			if len(options.AllowedPaths) == 0 {
				ch(next).ServeHTTP(w, r)
				return
			}

			// Check if the request path starts with any of the allowed prefixes
			for _, path := range options.AllowedPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					ch(next).ServeHTTP(w, r)
					return
				}
			}

			// If none of the prefixes match, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
