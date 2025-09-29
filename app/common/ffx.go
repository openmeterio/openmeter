package common

import (
	"log/slog"
	"net/http"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/ffx"
)

var FFX = wire.NewSet(
	ffx.NewContextService,
)

type FFXConfigContextMiddlewareHook server.MiddlewareHook

// NewFFXConfigContextMiddlewareHook creates a middleware hook that sets the feature flag access context on the request context.
// This hook MUST register after any session authentication step so user namespaces are available.
func NewFFXConfigContextMiddlewareHook(
	subsConfig config.SubscriptionConfiguration,
	namespaceDriver namespacedriver.NamespaceDecoder,
	logger *slog.Logger,
) FFXConfigContextMiddlewareHook {
	return func(m server.MiddlewareManager) {
		accessMap := make(map[string]ffx.AccessConfig)
		for _, ns := range subsConfig.MultiSubscriptionNamespaces {
			acc := make(ffx.AccessConfig)
			acc[subscription.MultiSubscriptionEnabledFF] = true

			accessMap[ns] = acc
		}

		noAccess := make(ffx.AccessConfig)
		noAccess[subscription.MultiSubscriptionEnabledFF] = false

		m.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				// Let's try to figure out which namespace we're in
				namespace, ok := namespaceDriver.GetNamespace(ctx)
				if !ok {
					logger.WarnContext(ctx, "no namespace found in request, continuing without feature flag access")
				}

				acc, ok := accessMap[namespace]
				if !ok {
					acc = noAccess
				}

				ctx = ffx.SetAccessOnContext(ctx, acc)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
	}
}
