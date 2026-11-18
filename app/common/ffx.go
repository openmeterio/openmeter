package common

import (
	"log/slog"
	"maps"
	"net/http"

	"github.com/google/wire"

	apiv1 "github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/ffx"
)

var FFX = wire.NewSet(
	ffx.NewContextService,
)

type FFXConfigContextMiddleware apiv1.MiddlewareFunc

// NewFFXConfigContextMiddleware creates a middleware hook that sets the feature flag access context on the request context.
// This hook MUST register after any session authentication step so user namespaces are available.
func NewFFXConfigContextMiddleware(
	subsConfig config.SubscriptionConfiguration,
	namespaceDriver namespacedriver.NamespaceDecoder,
	logger *slog.Logger,
) FFXConfigContextMiddleware {
	return func(next http.Handler) http.Handler {
		namespacesByFeature := map[ffx.Feature][]string{
			subscription.MultiSubscriptionEnabledFF: subsConfig.MultiSubscriptionNamespaces,
			subscription.BillingStartsAtEnabledFF:   subsConfig.BillingStartsAtNamespaces,
		}

		// Looking up a feature that is missing from the access config is an error,
		// so every namespace gets a value for every feature.
		noAccess := make(ffx.AccessConfig, len(namespacesByFeature))
		for feature := range namespacesByFeature {
			noAccess[feature] = false
		}

		accessMap := make(map[string]ffx.AccessConfig)
		for feature, namespaces := range namespacesByFeature {
			for _, ns := range namespaces {
				if _, ok := accessMap[ns]; !ok {
					accessMap[ns] = maps.Clone(noAccess)
				}

				accessMap[ns][feature] = true
			}
		}

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
	}
}
