package common_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/ffx"
)

func TestFFXConfigContextMiddleware(t *testing.T) {
	subsConfig := config.SubscriptionConfiguration{
		MultiSubscriptionNamespaces: []string{"multi-subscription", "both"},
		BillingStartsAtNamespaces:   []string{"billing-starts-at", "both"},
	}

	for _, tc := range []struct {
		namespace       string
		multiSub        bool
		billingStartsAt bool
	}{
		{namespace: "multi-subscription", multiSub: true, billingStartsAt: false},
		{namespace: "billing-starts-at", multiSub: false, billingStartsAt: true},
		{namespace: "both", multiSub: true, billingStartsAt: true},
		{namespace: "neither", multiSub: false, billingStartsAt: false},
	} {
		t.Run(tc.namespace, func(t *testing.T) {
			middleware := common.NewFFXConfigContextMiddleware(subsConfig, namespacedriver.StaticNamespaceDecoder(tc.namespace), slog.New(slog.DiscardHandler))

			var access ffx.AccessConfig
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				var err error
				access, err = ffx.GetAccessFromContext(r.Context())
				require.NoError(t, err)
			}))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

			// Every feature has a value, as looking up a missing one is an error.
			require.Equal(t, ffx.AccessConfig{
				subscription.MultiSubscriptionEnabledFF: tc.multiSub,
				subscription.BillingStartsAtEnabledFF:   tc.billingStartsAt,
			}, access)
		})
	}
}
